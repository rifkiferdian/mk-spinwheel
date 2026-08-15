package admin

import (
	"context"
	"database/sql"
	"os"
	"path/filepath"
	"testing"

	_ "modernc.org/sqlite"
)

func TestSaveCampaignSupportsMultipleGames(t *testing.T) {
	db, err := sql.Open("sqlite", filepath.Join(t.TempDir(), "admin.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	db.SetMaxOpenConns(1)
	if _, err = db.Exec("PRAGMA foreign_keys=ON"); err != nil {
		t.Fatal(err)
	}
	for _, name := range []string{"001_initial_schema.sql", "002_campaign_games.sql"} {
		content, readErr := os.ReadFile(filepath.Join("..", "..", "migrations", name))
		if readErr != nil {
			t.Fatal(readErr)
		}
		if _, err = db.Exec(string(content)); err != nil {
			t.Fatal(err)
		}
	}

	store := NewStore(db)
	item := Campaign{
		Name:       "Campaign Multi Game",
		Slug:       "campaign-multi-game",
		GameConfig: `{"headline":"Hadiah Ceria"}`,
		GameCodes:  []string{"wheel", "claw"},
		IsActive:   true,
	}
	if err = store.SaveCampaign(context.Background(), item); err != nil {
		t.Fatal(err)
	}
	items, err := store.Campaigns(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if len(items) != 1 || len(items[0].Games) != 2 {
		t.Fatalf("campaign=%d game=%v", len(items), items[0].Games)
	}
	if !items[0].HasGame("wheel") || !items[0].HasGame("claw") {
		t.Fatalf("game campaign tidak lengkap: %#v", items[0].Games)
	}

	campaignID := items[0].ID
	if err = store.SavePrize(context.Background(), Prize{CampaignID: campaignID, Name: "Voucher", Weight: 1, InitialStock: 5, RemainingStock: 5, RequiresClaim: true, IsActive: true}); err != nil {
		t.Fatal(err)
	}
	if err = store.SavePrize(context.Background(), Prize{CampaignID: campaignID, Name: "Belum Beruntung", Weight: 1, IsUnlimited: true, IsActive: true}); err != nil {
		t.Fatal(err)
	}
	var voucherID, zonkID int64
	if err = db.QueryRow(`SELECT id FROM prizes WHERE name='Voucher'`).Scan(&voucherID); err != nil {
		t.Fatal(err)
	}
	if err = db.QueryRow(`SELECT id FROM prizes WHERE name='Belum Beruntung'`).Scan(&zonkID); err != nil {
		t.Fatal(err)
	}
	mustAdminExec(t, db, `INSERT INTO game_sessions (campaign_id,session_token) VALUES (?,?)`, campaignID, "report-session-1")
	mustAdminExec(t, db, `INSERT INTO game_sessions (campaign_id,session_token) VALUES (?,?)`, campaignID, "report-session-2")
	mustAdminExec(t, db, `INSERT INTO game_results (game_session_id,campaign_id,prize_id,claim_code,claim_status) VALUES ((SELECT id FROM game_sessions WHERE session_token=?),?,?,?,'pending')`, "report-session-1", campaignID, voucherID, "REPORT-1")
	mustAdminExec(t, db, `INSERT INTO game_results (game_session_id,campaign_id,prize_id,claim_status) VALUES ((SELECT id FROM game_sessions WHERE session_token=?),?,?,'not_required')`, "report-session-2", campaignID, zonkID)

	var reportDate string
	if err = db.QueryRow(`SELECT date(played_at) FROM game_results LIMIT 1`).Scan(&reportDate); err != nil {
		t.Fatal(err)
	}
	summary, prizes, daily, details, err := store.Report(context.Background(), ReportFilter{CampaignID: campaignID, From: reportDate, To: reportDate}, 200)
	if err != nil {
		t.Fatal(err)
	}
	if summary.TotalPlays != 2 || summary.ClaimableWins != 1 || summary.Pending != 1 || summary.NotRequired != 1 {
		t.Fatalf("summary report tidak valid: %#v", summary)
	}
	if len(prizes) != 2 || len(daily) != 1 || len(details) != 2 {
		t.Fatalf("report prizes=%d daily=%d details=%d", len(prizes), len(daily), len(details))
	}
}

func mustAdminExec(t *testing.T, db *sql.DB, query string, args ...any) {
	t.Helper()
	if _, err := db.Exec(query, args...); err != nil {
		t.Fatal(err)
	}
}
