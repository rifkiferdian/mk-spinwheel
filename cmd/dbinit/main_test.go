package main

import (
	"database/sql"
	"path/filepath"
	"testing"
)

func TestSchemaDecrementsAndProtectsPrizeStock(t *testing.T) {
	databasePath := filepath.Join(t.TempDir(), "test.db")
	db, err := sql.Open("sqlite", databasePath)
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	db.SetMaxOpenConns(1)

	if _, err := db.Exec("PRAGMA foreign_keys = ON"); err != nil {
		t.Fatal(err)
	}
	if err := executeSQLFile(db, filepath.Join("..", "..", "migrations", "001_initial_schema.sql")); err != nil {
		t.Fatal(err)
	}

	mustExec(t, db, `
		INSERT INTO campaigns (game_type_code, name, slug, is_active)
		VALUES ('wheel', 'Test Campaign', 'test-campaign', 1)
	`)
	mustExec(t, db, `
		INSERT INTO prizes (
			campaign_id, name, weight, initial_stock, remaining_stock
		) VALUES (1, 'Hadiah Terbatas', 1, 1, 1)
	`)
	mustExec(t, db, `
		INSERT INTO game_sessions (campaign_id, session_token)
		VALUES (1, 'session-1')
	`)
	mustExec(t, db, `
		INSERT INTO game_results (
			game_session_id, campaign_id, prize_id, claim_code
		) VALUES (1, 1, 1, 'CLAIM-1')
	`)

	var stock int
	if err := db.QueryRow("SELECT remaining_stock FROM prizes WHERE id = 1").Scan(&stock); err != nil {
		t.Fatal(err)
	}
	if stock != 0 {
		t.Fatalf("remaining_stock = %d, ingin 0", stock)
	}

	var sessionStatus string
	if err := db.QueryRow("SELECT status FROM game_sessions WHERE id = 1").Scan(&sessionStatus); err != nil {
		t.Fatal(err)
	}
	if sessionStatus != "completed" {
		t.Fatalf("status session = %q, ingin completed", sessionStatus)
	}

	mustExec(t, db, `
		INSERT INTO game_sessions (campaign_id, session_token)
		VALUES (1, 'session-2')
	`)
	if _, err := db.Exec(`
		INSERT INTO game_results (
			game_session_id, campaign_id, prize_id, claim_code
		) VALUES (2, 1, 1, 'CLAIM-2')
	`); err == nil {
		t.Fatal("database seharusnya menolak hadiah yang stoknya habis")
	}
}

func TestRenameMysteryBoxMigration(t *testing.T) {
	databasePath := filepath.Join(t.TempDir(), "rename.db")
	db, err := sql.Open("sqlite", databasePath)
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	db.SetMaxOpenConns(1)
	if _, err = db.Exec("PRAGMA foreign_keys = ON"); err != nil {
		t.Fatal(err)
	}
	if err = executeSQLFile(db, filepath.Join("..", "..", "migrations", "001_initial_schema.sql")); err != nil {
		t.Fatal(err)
	}
	mustExec(t, db, `INSERT INTO game_types (code,name,frontend_module) VALUES ('mystery-box','Kotak Misteri','games/mystery-box.js')`)
	mustExec(t, db, `INSERT INTO campaigns (game_type_code,name,slug,is_active) VALUES ('mystery-box','Campaign Lama','campaign-lama',1)`)
	if err = executeSQLFile(db, filepath.Join("..", "..", "migrations", "002_campaign_games.sql")); err != nil {
		t.Fatal(err)
	}
	if err = executeSQLFile(db, filepath.Join("..", "..", "migrations", "003_rename_mystery_box.sql")); err != nil {
		t.Fatal(err)
	}
	var campaignType, campaignGameType string
	if err = db.QueryRow(`SELECT game_type_code FROM campaigns WHERE slug='campaign-lama'`).Scan(&campaignType); err != nil {
		t.Fatal(err)
	}
	if err = db.QueryRow(`SELECT game_type_code FROM campaign_games WHERE campaign_id=(SELECT id FROM campaigns WHERE slug='campaign-lama')`).Scan(&campaignGameType); err != nil {
		t.Fatal(err)
	}
	if campaignType != "lucky-dip" || campaignGameType != "lucky-dip" {
		t.Fatalf("campaign=%q relasi=%q", campaignType, campaignGameType)
	}
	var oldCount int
	if err = db.QueryRow(`SELECT COUNT(*) FROM game_types WHERE code='mystery-box'`).Scan(&oldCount); err != nil {
		t.Fatal(err)
	}
	if oldCount != 0 {
		t.Fatalf("mystery-box masih tersisa: %d", oldCount)
	}
}

func mustExec(t *testing.T, db *sql.DB, query string) {
	t.Helper()
	if _, err := db.Exec(query); err != nil {
		t.Fatal(err)
	}
}
