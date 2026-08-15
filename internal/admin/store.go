package admin

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"

	"golang.org/x/crypto/bcrypt"
)

type Store struct {
	db *sql.DB
}

func NewStore(db *sql.DB) *Store { return &Store{db: db} }

type DashboardStats struct {
	Campaigns       int
	ActiveCampaigns int
	Prizes          int
	RemainingStock  int
	TotalPlays      int
	PendingClaims   int
}

type GameType struct {
	Code, Name, FrontendModule, CreatedAt string
	IsActive                              bool
}

type CampaignGame struct {
	Code, Name, FrontendModule, GameConfig string
	IsActive                               bool
	DisplayOrder                           int
}

type Campaign struct {
	ID                                                 int64
	GameTypeCode, GameTypeName, Name, Slug, GameConfig string
	StartsAt, EndsAt, CreatedAt, UpdatedAt             string
	IsActive                                           bool
	Games                                              []CampaignGame
	GameCodes                                          []string
}

func (c Campaign) HasGame(code string) bool {
	for _, game := range c.Games {
		if game.Code == code {
			return true
		}
	}
	for _, selected := range c.GameCodes {
		if selected == code {
			return true
		}
	}
	return false
}

type Prize struct {
	ID                                                int64
	CampaignID                                        int64
	CampaignName, Name, Description, ImagePath, Color string
	Weight                                            float64
	InitialStock, RemainingStock, DisplayOrder        int
	IsUnlimited, RequiresClaim, IsActive              bool
	CreatedAt, UpdatedAt                              string
}

type AccessCode struct {
	ID, CampaignID                     int64
	CampaignName, Code, Status, UsedAt string
	CreatedAt                          string
}

type GameSession struct {
	ID, CampaignID                                           int64
	CampaignName, SessionToken, Status                       string
	AccessCode, CreatedAt, StartedAt, CompletedAt, ExpiresAt string
}

type GameResult struct {
	ID, SessionID, CampaignID, PrizeID              int64
	CampaignName, PrizeName, ClaimCode, ClaimStatus string
	PlayedAt, ClaimedAt                             string
}

type AdminUser struct {
	ID                             int64
	Username, CreatedAt, UpdatedAt string
	IsActive                       bool
}

func (s *Store) Dashboard(ctx context.Context) (DashboardStats, []GameResult, error) {
	var stats DashboardStats
	err := s.db.QueryRowContext(ctx, `
		SELECT
			(SELECT COUNT(*) FROM campaigns),
			(SELECT COUNT(*) FROM campaigns WHERE is_active = 1),
			(SELECT COUNT(*) FROM prizes),
			COALESCE((SELECT SUM(remaining_stock) FROM prizes WHERE is_unlimited = 0), 0),
			(SELECT COUNT(*) FROM game_results),
			(SELECT COUNT(*) FROM game_results WHERE claim_status = 'pending')
	`).Scan(&stats.Campaigns, &stats.ActiveCampaigns, &stats.Prizes, &stats.RemainingStock, &stats.TotalPlays, &stats.PendingClaims)
	if err != nil {
		return stats, nil, err
	}
	results, err := s.Results(ctx, 8)
	return stats, results, err
}

func (s *Store) GameTypes(ctx context.Context) ([]GameType, error) {
	rows, err := s.db.QueryContext(ctx, `SELECT code, name, frontend_module, is_active, created_at FROM game_types ORDER BY name`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var items []GameType
	for rows.Next() {
		var item GameType
		if err := rows.Scan(&item.Code, &item.Name, &item.FrontendModule, &item.IsActive, &item.CreatedAt); err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	return items, rows.Err()
}

func (s *Store) SaveGameType(ctx context.Context, item GameType) error {
	item.Code = strings.ToLower(strings.TrimSpace(item.Code))
	if item.Code == "" || item.Name == "" || item.FrontendModule == "" {
		return errors.New("kode, nama, dan modul frontend wajib diisi")
	}
	_, err := s.db.ExecContext(ctx, `
		INSERT INTO game_types (code, name, frontend_module, is_active)
		VALUES (?, ?, ?, ?)
		ON CONFLICT(code) DO UPDATE SET name = excluded.name, frontend_module = excluded.frontend_module, is_active = excluded.is_active
	`, item.Code, strings.TrimSpace(item.Name), strings.TrimSpace(item.FrontendModule), item.IsActive)
	return err
}

func (s *Store) Campaigns(ctx context.Context) ([]Campaign, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT c.id, c.game_type_code, gt.name, c.name, c.slug, c.game_config,
		       COALESCE(c.starts_at, ''), COALESCE(c.ends_at, ''), c.is_active, c.created_at, c.updated_at
		FROM campaigns c JOIN game_types gt ON gt.code = c.game_type_code
		ORDER BY c.created_at DESC
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var items []Campaign
	for rows.Next() {
		var item Campaign
		if err := rows.Scan(&item.ID, &item.GameTypeCode, &item.GameTypeName, &item.Name, &item.Slug, &item.GameConfig, &item.StartsAt, &item.EndsAt, &item.IsActive, &item.CreatedAt, &item.UpdatedAt); err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	if err := rows.Close(); err != nil {
		return nil, err
	}
	for index := range items {
		items[index].Games, err = s.CampaignGames(ctx, items[index].ID)
		if err != nil {
			return nil, err
		}
	}
	return items, nil
}

func (s *Store) Campaign(ctx context.Context, id int64) (Campaign, error) {
	var item Campaign
	err := s.db.QueryRowContext(ctx, `
		SELECT c.id, c.game_type_code, gt.name, c.name, c.slug, c.game_config,
		       COALESCE(c.starts_at, ''), COALESCE(c.ends_at, ''), c.is_active, c.created_at, c.updated_at
		FROM campaigns c JOIN game_types gt ON gt.code = c.game_type_code WHERE c.id = ?
	`, id).Scan(&item.ID, &item.GameTypeCode, &item.GameTypeName, &item.Name, &item.Slug, &item.GameConfig, &item.StartsAt, &item.EndsAt, &item.IsActive, &item.CreatedAt, &item.UpdatedAt)
	if err == nil {
		item.Games, err = s.CampaignGames(ctx, item.ID)
	}
	return item, err
}

func (s *Store) CampaignGames(ctx context.Context, campaignID int64) ([]CampaignGame, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT cg.game_type_code,gt.name,gt.frontend_module,cg.game_config,cg.is_active,cg.display_order
		FROM campaign_games cg
		JOIN game_types gt ON gt.code=cg.game_type_code
		WHERE cg.campaign_id=? AND cg.is_active=1
		ORDER BY cg.display_order,gt.name
	`, campaignID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var games []CampaignGame
	for rows.Next() {
		var game CampaignGame
		if err := rows.Scan(&game.Code, &game.Name, &game.FrontendModule, &game.GameConfig, &game.IsActive, &game.DisplayOrder); err != nil {
			return nil, err
		}
		games = append(games, game)
	}
	return games, rows.Err()
}

func (s *Store) SaveCampaign(ctx context.Context, item Campaign) error {
	var gameCodes []string
	seen := make(map[string]bool)
	for _, code := range item.GameCodes {
		code = strings.ToLower(strings.TrimSpace(code))
		if code != "" && !seen[code] {
			seen[code] = true
			gameCodes = append(gameCodes, code)
		}
	}
	if item.Name == "" || item.Slug == "" || len(gameCodes) == 0 {
		return errors.New("nama, slug, dan minimal satu jenis game wajib diisi")
	}
	item.GameTypeCode = gameCodes[0]
	if item.GameConfig == "" {
		item.GameConfig = "{}"
	}
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	if item.ID == 0 {
		result, err := tx.ExecContext(ctx, `
			INSERT INTO campaigns (game_type_code, name, slug, game_config, starts_at, ends_at, is_active)
			VALUES (?, ?, ?, ?, NULLIF(?, ''), NULLIF(?, ''), ?)
		`, item.GameTypeCode, strings.TrimSpace(item.Name), strings.TrimSpace(item.Slug), item.GameConfig, item.StartsAt, item.EndsAt, item.IsActive)
		if err != nil {
			return err
		}
		item.ID, err = result.LastInsertId()
		if err != nil {
			return err
		}
	} else {
		_, err = tx.ExecContext(ctx, `
			UPDATE campaigns SET game_type_code=?, name=?, slug=?, game_config=?, starts_at=NULLIF(?, ''), ends_at=NULLIF(?, ''),
			is_active=?, updated_at=strftime('%Y-%m-%dT%H:%M:%fZ', 'now') WHERE id=?
		`, item.GameTypeCode, strings.TrimSpace(item.Name), strings.TrimSpace(item.Slug), item.GameConfig, item.StartsAt, item.EndsAt, item.IsActive, item.ID)
		if err != nil {
			return err
		}
	}
	if _, err = tx.ExecContext(ctx, `UPDATE campaign_games SET is_active=0,updated_at=strftime('%Y-%m-%dT%H:%M:%fZ','now') WHERE campaign_id=?`, item.ID); err != nil {
		return err
	}
	for order, code := range gameCodes {
		if _, err = tx.ExecContext(ctx, `
			INSERT INTO campaign_games (campaign_id,game_type_code,game_config,is_active,display_order)
			VALUES (?,?,?,?,?)
			ON CONFLICT(campaign_id,game_type_code) DO UPDATE SET
			is_active=1,display_order=excluded.display_order,
			updated_at=strftime('%Y-%m-%dT%H:%M:%fZ','now')
		`, item.ID, code, defaultGameConfig(code, item.GameConfig), true, order); err != nil {
			return err
		}
	}
	return tx.Commit()
}

func defaultGameConfig(code, fallback string) string {
	if code == "claw" {
		return `{"theme":"manna-claw","duration_ms":5200,"show_confetti":true,"headline":"Capit & Bawa Pulang Hadiahnya!"}`
	}
	return fallback
}

func (s *Store) Prizes(ctx context.Context, campaignID int64) ([]Prize, error) {
	query := `
		SELECT p.id, p.campaign_id, c.name, p.name, COALESCE(p.description,''), COALESCE(p.image_path,''), COALESCE(p.color,''),
		p.weight, p.initial_stock, p.remaining_stock, p.is_unlimited, p.requires_claim, p.display_order, p.is_active, p.created_at, p.updated_at
		FROM prizes p JOIN campaigns c ON c.id=p.campaign_id`
	var args []any
	if campaignID > 0 {
		query += " WHERE p.campaign_id=?"
		args = append(args, campaignID)
	}
	query += " ORDER BY c.name, p.display_order, p.name"
	rows, err := s.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var items []Prize
	for rows.Next() {
		var item Prize
		if err := rows.Scan(&item.ID, &item.CampaignID, &item.CampaignName, &item.Name, &item.Description, &item.ImagePath, &item.Color, &item.Weight, &item.InitialStock, &item.RemainingStock, &item.IsUnlimited, &item.RequiresClaim, &item.DisplayOrder, &item.IsActive, &item.CreatedAt, &item.UpdatedAt); err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	return items, rows.Err()
}

func (s *Store) Prize(ctx context.Context, id int64) (Prize, error) {
	var item Prize
	err := s.db.QueryRowContext(ctx, `
		SELECT p.id, p.campaign_id, c.name, p.name, COALESCE(p.description,''), COALESCE(p.image_path,''), COALESCE(p.color,''),
		p.weight, p.initial_stock, p.remaining_stock, p.is_unlimited, p.requires_claim, p.display_order, p.is_active, p.created_at, p.updated_at
		FROM prizes p JOIN campaigns c ON c.id=p.campaign_id WHERE p.id=?
	`, id).Scan(&item.ID, &item.CampaignID, &item.CampaignName, &item.Name, &item.Description, &item.ImagePath, &item.Color, &item.Weight, &item.InitialStock, &item.RemainingStock, &item.IsUnlimited, &item.RequiresClaim, &item.DisplayOrder, &item.IsActive, &item.CreatedAt, &item.UpdatedAt)
	return item, err
}

func (s *Store) SavePrize(ctx context.Context, item Prize) error {
	if item.CampaignID == 0 || strings.TrimSpace(item.Name) == "" || item.Weight <= 0 {
		return errors.New("campaign, nama, dan bobot hadiah wajib valid")
	}
	if item.InitialStock < 0 || item.RemainingStock < 0 {
		return errors.New("stok tidak boleh negatif")
	}
	if item.ID == 0 {
		_, err := s.db.ExecContext(ctx, `
			INSERT INTO prizes (campaign_id,name,description,image_path,color,weight,initial_stock,remaining_stock,is_unlimited,requires_claim,display_order,is_active)
			VALUES (?,?,?,?,?,?,?,?,?,?,?,?)
		`, item.CampaignID, strings.TrimSpace(item.Name), nullString(item.Description), nullString(item.ImagePath), nullString(item.Color), item.Weight, item.InitialStock, item.RemainingStock, item.IsUnlimited, item.RequiresClaim, item.DisplayOrder, item.IsActive)
		return err
	}
	_, err := s.db.ExecContext(ctx, `
		UPDATE prizes SET campaign_id=?,name=?,description=?,image_path=?,color=?,weight=?,initial_stock=?,remaining_stock=?,
		is_unlimited=?,requires_claim=?,display_order=?,is_active=?,updated_at=strftime('%Y-%m-%dT%H:%M:%fZ','now') WHERE id=?
	`, item.CampaignID, strings.TrimSpace(item.Name), nullString(item.Description), nullString(item.ImagePath), nullString(item.Color), item.Weight, item.InitialStock, item.RemainingStock, item.IsUnlimited, item.RequiresClaim, item.DisplayOrder, item.IsActive, item.ID)
	return err
}

func (s *Store) AccessCodes(ctx context.Context, campaignID int64) ([]AccessCode, error) {
	query := `SELECT a.id,a.campaign_id,c.name,a.code,a.status,COALESCE(a.used_at,''),a.created_at FROM access_codes a JOIN campaigns c ON c.id=a.campaign_id`
	var args []any
	if campaignID > 0 {
		query += " WHERE a.campaign_id=?"
		args = append(args, campaignID)
	}
	query += " ORDER BY a.created_at DESC LIMIT 500"
	rows, err := s.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var items []AccessCode
	for rows.Next() {
		var item AccessCode
		if err := rows.Scan(&item.ID, &item.CampaignID, &item.CampaignName, &item.Code, &item.Status, &item.UsedAt, &item.CreatedAt); err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	return items, rows.Err()
}

func (s *Store) AddAccessCodes(ctx context.Context, campaignID int64, raw string) (int, error) {
	if campaignID == 0 {
		return 0, errors.New("campaign wajib dipilih")
	}
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return 0, err
	}
	defer tx.Rollback()
	stmt, err := tx.PrepareContext(ctx, `INSERT OR IGNORE INTO access_codes (campaign_id,code) VALUES (?,?)`)
	if err != nil {
		return 0, err
	}
	defer stmt.Close()
	count := 0
	for _, code := range strings.FieldsFunc(raw, func(r rune) bool { return r == '\n' || r == '\r' || r == ',' || r == ';' }) {
		code = strings.TrimSpace(code)
		if code == "" {
			continue
		}
		result, err := stmt.ExecContext(ctx, campaignID, code)
		if err != nil {
			return 0, err
		}
		n, _ := result.RowsAffected()
		count += int(n)
	}
	if count == 0 {
		return 0, errors.New("tidak ada kode baru yang valid")
	}
	return count, tx.Commit()
}

func (s *Store) SetAccessCodeStatus(ctx context.Context, id int64, status string) error {
	if status != "unused" && status != "expired" && status != "disabled" {
		return errors.New("status kode tidak valid")
	}
	_, err := s.db.ExecContext(ctx, `UPDATE access_codes SET status=?, used_at=CASE WHEN ?='unused' THEN NULL ELSE used_at END WHERE id=?`, status, status, id)
	return err
}

func (s *Store) Sessions(ctx context.Context, limit int) ([]GameSession, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT s.id,s.campaign_id,c.name,s.session_token,s.status,COALESCE(a.code,''),s.created_at,
		COALESCE(s.started_at,''),COALESCE(s.completed_at,''),COALESCE(s.expires_at,'')
		FROM game_sessions s JOIN campaigns c ON c.id=s.campaign_id LEFT JOIN access_codes a ON a.id=s.access_code_id
		ORDER BY s.created_at DESC LIMIT ?
	`, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var items []GameSession
	for rows.Next() {
		var item GameSession
		if err := rows.Scan(&item.ID, &item.CampaignID, &item.CampaignName, &item.SessionToken, &item.Status, &item.AccessCode, &item.CreatedAt, &item.StartedAt, &item.CompletedAt, &item.ExpiresAt); err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	return items, rows.Err()
}

func (s *Store) Results(ctx context.Context, limit int) ([]GameResult, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT r.id,r.game_session_id,r.campaign_id,r.prize_id,c.name,p.name,COALESCE(r.claim_code,''),r.claim_status,r.played_at,COALESCE(r.claimed_at,'')
		FROM game_results r JOIN campaigns c ON c.id=r.campaign_id JOIN prizes p ON p.id=r.prize_id
		ORDER BY r.played_at DESC LIMIT ?
	`, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var items []GameResult
	for rows.Next() {
		var item GameResult
		if err := rows.Scan(&item.ID, &item.SessionID, &item.CampaignID, &item.PrizeID, &item.CampaignName, &item.PrizeName, &item.ClaimCode, &item.ClaimStatus, &item.PlayedAt, &item.ClaimedAt); err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	return items, rows.Err()
}

func (s *Store) SetClaimStatus(ctx context.Context, id int64, status string) error {
	if status != "claimed" && status != "pending" && status != "cancelled" && status != "not_required" {
		return errors.New("status klaim tidak valid")
	}
	_, err := s.db.ExecContext(ctx, `UPDATE game_results SET claim_status=?, claimed_at=CASE WHEN ?='claimed' THEN strftime('%Y-%m-%dT%H:%M:%fZ','now') ELSE NULL END WHERE id=?`, status, status, id)
	return err
}

func (s *Store) Admins(ctx context.Context) ([]AdminUser, error) {
	rows, err := s.db.QueryContext(ctx, `SELECT id,username,is_active,created_at,updated_at FROM admins ORDER BY username`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var items []AdminUser
	for rows.Next() {
		var item AdminUser
		if err := rows.Scan(&item.ID, &item.Username, &item.IsActive, &item.CreatedAt, &item.UpdatedAt); err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	return items, rows.Err()
}

func (s *Store) CreateAdmin(ctx context.Context, username, password string) error {
	username = strings.TrimSpace(username)
	if len(username) < 3 {
		return errors.New("username minimal 3 karakter")
	}
	if len(password) < 8 {
		return errors.New("password minimal 8 karakter")
	}
	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return err
	}
	_, err = s.db.ExecContext(ctx, `INSERT INTO admins (username,password_hash) VALUES (?,?)`, username, string(hash))
	return err
}

func (s *Store) SetAdminActive(ctx context.Context, id int64, active bool) error {
	if !active {
		var activeCount int
		if err := s.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM admins WHERE is_active=1`).Scan(&activeCount); err != nil {
			return err
		}
		if activeCount <= 1 {
			return errors.New("minimal satu admin harus tetap aktif")
		}
	}
	_, err := s.db.ExecContext(ctx, `UPDATE admins SET is_active=?,updated_at=strftime('%Y-%m-%dT%H:%M:%fZ','now') WHERE id=?`, active, id)
	return err
}

func (s *Store) Authenticate(ctx context.Context, username, password string) (AdminUser, error) {
	var item AdminUser
	var hash string
	err := s.db.QueryRowContext(ctx, `SELECT id,username,password_hash,is_active,created_at,updated_at FROM admins WHERE username=?`, strings.TrimSpace(username)).Scan(&item.ID, &item.Username, &hash, &item.IsActive, &item.CreatedAt, &item.UpdatedAt)
	if err != nil {
		return item, errors.New("username atau password salah")
	}
	if !item.IsActive || bcrypt.CompareHashAndPassword([]byte(hash), []byte(password)) != nil {
		return item, errors.New("username atau password salah")
	}
	return item, nil
}

func (s *Store) AdminCount(ctx context.Context) (int, error) {
	var n int
	err := s.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM admins`).Scan(&n)
	return n, err
}

func nullString(value string) any {
	if strings.TrimSpace(value) == "" {
		return nil
	}
	return strings.TrimSpace(value)
}

func friendlyDBError(err error) error {
	if err == nil {
		return nil
	}
	message := err.Error()
	switch {
	case strings.Contains(message, "UNIQUE constraint failed"):
		return errors.New("data dengan nilai tersebut sudah ada")
	case strings.Contains(message, "FOREIGN KEY constraint failed"):
		return errors.New("data masih digunakan dan tidak dapat diubah")
	case strings.Contains(message, "CHECK constraint failed"):
		return errors.New("data tidak memenuhi aturan database")
	default:
		return fmt.Errorf("operasi database gagal: %w", err)
	}
}
