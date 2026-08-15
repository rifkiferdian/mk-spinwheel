package game

import (
	"context"
	"crypto/rand"
	"database/sql"
	"encoding/base64"
	"errors"
	"fmt"
	"math"
	"math/big"
	"strings"
)

var ErrNotFound = errors.New("campaign tidak ditemukan atau sedang tidak aktif")

type Store struct{ db *sql.DB }

func NewStore(db *sql.DB) *Store { return &Store{db: db} }

type Campaign struct {
	ID         int64          `json:"id"`
	Name       string         `json:"name"`
	Slug       string         `json:"slug"`
	GameType   string         `json:"gameType"`
	GameConfig string         `json:"-"`
	Config     map[string]any `json:"config"`
	Prizes     []Prize        `json:"prizes"`
}

type Prize struct {
	ID            int64   `json:"id"`
	Name          string  `json:"name"`
	Description   string  `json:"description"`
	Color         string  `json:"color"`
	ImagePath     string  `json:"imagePath,omitempty"`
	Weight        float64 `json:"-"`
	RequiresClaim bool    `json:"-"`
	VisualCount   int     `json:"visualCount"`
}

type SpinResult struct {
	PrizeID          int64  `json:"prizeId"`
	PrizeName        string `json:"prizeName"`
	PrizeDescription string `json:"prizeDescription"`
	ClaimCode        string `json:"claimCode,omitempty"`
	ClaimStatus      string `json:"claimStatus"`
}

type prizeCandidate struct {
	id            int64
	name          string
	description   string
	weight        float64
	requiresClaim bool
}

func (s *Store) Campaign(ctx context.Context, slug string) (Campaign, error) {
	var item Campaign
	err := s.db.QueryRowContext(ctx, `
		SELECT id,name,slug,game_type_code,game_config
		FROM campaigns
		WHERE slug=? AND is_active=1
		  AND (starts_at IS NULL OR starts_at <= strftime('%Y-%m-%dT%H:%M','now','localtime'))
		  AND (ends_at IS NULL OR ends_at >= strftime('%Y-%m-%dT%H:%M','now','localtime'))
	`, slug).Scan(&item.ID, &item.Name, &item.Slug, &item.GameType, &item.GameConfig)
	if errors.Is(err, sql.ErrNoRows) {
		return item, ErrNotFound
	}
	if err != nil {
		return item, err
	}
	rows, err := s.db.QueryContext(ctx, `
		SELECT id,name,COALESCE(description,''),COALESCE(color,'#6366F1'),COALESCE(image_path,''),weight,requires_claim
		FROM prizes WHERE campaign_id=? AND is_active=1 AND (is_unlimited=1 OR remaining_stock>0)
		ORDER BY display_order,name
	`, item.ID)
	if err != nil {
		return item, err
	}
	defer rows.Close()
	for rows.Next() {
		var p Prize
		if err := rows.Scan(&p.ID, &p.Name, &p.Description, &p.Color, &p.ImagePath, &p.Weight, &p.RequiresClaim); err != nil {
			return item, err
		}
		item.Prizes = append(item.Prizes, p)
	}
	if err := rows.Err(); err != nil {
		return item, err
	}
	if len(item.Prizes) < 2 {
		return item, errors.New("campaign membutuhkan minimal dua hadiah aktif")
	}
	allocateVisualCounts(item.Prizes)
	return item, nil
}

func allocateVisualCounts(prizes []Prize) {
	totalWeight := 0.0
	for _, prize := range prizes {
		totalWeight += prize.Weight
	}
	if totalWeight <= 0 {
		return
	}
	for index := range prizes {
		count := int(math.Round((prizes[index].Weight / totalWeight) * 20))
		if count < 2 {
			count = 2
		}
		if count > 8 {
			count = 8
		}
		prizes[index].VisualCount = count
	}
}

func (s *Store) CreateSession(ctx context.Context, slug string) (string, error) {
	campaign, err := s.Campaign(ctx, slug)
	if err != nil {
		return "", err
	}
	token, err := secureToken(32)
	if err != nil {
		return "", err
	}
	_, err = s.db.ExecContext(ctx, `INSERT INTO game_sessions (campaign_id,session_token,status,expires_at) VALUES (?,?,'created',strftime('%Y-%m-%dT%H:%M:%fZ','now','+15 minutes'))`, campaign.ID, token)
	return token, err
}

func (s *Store) Play(ctx context.Context, token string) (SpinResult, error) {
	if strings.TrimSpace(token) == "" {
		return SpinResult{}, errors.New("token sesi wajib diisi")
	}
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return SpinResult{}, err
	}
	defer tx.Rollback()

	if result, found, err := existingResult(ctx, tx, token); err != nil {
		return SpinResult{}, err
	} else if found {
		return result, tx.Commit()
	}

	var sessionID, campaignID int64
	var status string
	err = tx.QueryRowContext(ctx, `
		SELECT s.id,s.campaign_id,s.status FROM game_sessions s
		JOIN campaigns c ON c.id=s.campaign_id
		WHERE s.session_token=? AND c.is_active=1
		  AND (s.expires_at IS NULL OR s.expires_at > strftime('%Y-%m-%dT%H:%M:%fZ','now'))
	`, token).Scan(&sessionID, &campaignID, &status)
	if errors.Is(err, sql.ErrNoRows) {
		return SpinResult{}, errors.New("sesi tidak ditemukan atau sudah kedaluwarsa")
	}
	if err != nil {
		return SpinResult{}, err
	}
	if status != "created" && status != "playing" {
		return SpinResult{}, errors.New("sesi permainan sudah tidak dapat digunakan")
	}
	if _, err = tx.ExecContext(ctx, `UPDATE game_sessions SET status='playing',started_at=COALESCE(started_at,strftime('%Y-%m-%dT%H:%M:%fZ','now')) WHERE id=?`, sessionID); err != nil {
		return SpinResult{}, err
	}

	rows, err := tx.QueryContext(ctx, `
		SELECT id,name,COALESCE(description,''),weight,requires_claim
		FROM prizes WHERE campaign_id=? AND is_active=1 AND (is_unlimited=1 OR remaining_stock>0)
		ORDER BY display_order,id
	`, campaignID)
	if err != nil {
		return SpinResult{}, err
	}
	var candidates []prizeCandidate
	total := 0.0
	for rows.Next() {
		var p prizeCandidate
		if err := rows.Scan(&p.id, &p.name, &p.description, &p.weight, &p.requiresClaim); err != nil {
			rows.Close()
			return SpinResult{}, err
		}
		candidates = append(candidates, p)
		total += p.weight
	}
	if err := rows.Close(); err != nil {
		return SpinResult{}, err
	}
	if len(candidates) == 0 || total <= 0 {
		return SpinResult{}, errors.New("tidak ada hadiah yang tersedia")
	}
	selected, err := weightedCandidate(candidates, total)
	if err != nil {
		return SpinResult{}, err
	}

	result := SpinResult{PrizeID: selected.id, PrizeName: selected.name, PrizeDescription: selected.description, ClaimStatus: "not_required"}
	if selected.requiresClaim {
		result.ClaimStatus = "pending"
		result.ClaimCode, err = claimCode()
		if err != nil {
			return SpinResult{}, err
		}
	}
	_, err = tx.ExecContext(ctx, `INSERT INTO game_results (game_session_id,campaign_id,prize_id,claim_code,claim_status) VALUES (?,?,?,?,?)`, sessionID, campaignID, selected.id, nullIfEmpty(result.ClaimCode), result.ClaimStatus)
	if err != nil {
		return SpinResult{}, fmt.Errorf("menyimpan hasil permainan: %w", err)
	}
	if err = tx.Commit(); err != nil {
		return SpinResult{}, err
	}
	return result, nil
}

func existingResult(ctx context.Context, tx *sql.Tx, token string) (SpinResult, bool, error) {
	var result SpinResult
	err := tx.QueryRowContext(ctx, `
	SELECT p.id,p.name,COALESCE(p.description,''),COALESCE(r.claim_code,''),r.claim_status
	FROM game_results r JOIN prizes p ON p.id=r.prize_id JOIN game_sessions s ON s.id=r.game_session_id
	WHERE s.session_token=?`, token).Scan(&result.PrizeID, &result.PrizeName, &result.PrizeDescription, &result.ClaimCode, &result.ClaimStatus)
	if errors.Is(err, sql.ErrNoRows) {
		return result, false, nil
	}
	return result, err == nil, err
}

func weightedCandidate(items []prizeCandidate, total float64) (prizeCandidate, error) {
	var zero prizeCandidate
	n, err := rand.Int(rand.Reader, big.NewInt(1_000_000_000))
	if err != nil {
		return zero, err
	}
	target := (float64(n.Int64()) / 1_000_000_000) * total
	cumulative := 0.0
	for _, item := range items {
		cumulative += item.weight
		if target < cumulative {
			return item, nil
		}
	}
	return items[len(items)-1], nil
}

func secureToken(size int) (string, error) {
	buffer := make([]byte, size)
	if _, err := rand.Read(buffer); err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(buffer), nil
}
func claimCode() (string, error) {
	const chars = "ABCDEFGHJKLMNPQRSTUVWXYZ23456789"
	buffer := make([]byte, 8)
	for i := range buffer {
		n, err := rand.Int(rand.Reader, big.NewInt(int64(len(chars))))
		if err != nil {
			return "", err
		}
		buffer[i] = chars[n.Int64()]
	}
	return "HDH-" + string(buffer[:4]) + "-" + string(buffer[4:]), nil
}
func nullIfEmpty(value string) any {
	if value == "" {
		return nil
	}
	return value
}
