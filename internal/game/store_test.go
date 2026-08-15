package game

import (
	"context"
	"database/sql"
	"os"
	"path/filepath"
	"testing"

	_ "modernc.org/sqlite"
)

func TestPlayIsIdempotentAndDecrementsStock(t *testing.T) {
	db, err := sql.Open("sqlite", filepath.Join(t.TempDir(), "game.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	db.SetMaxOpenConns(1)
	if _, err = db.Exec("PRAGMA foreign_keys=ON"); err != nil {
		t.Fatal(err)
	}
	executeFile(t, db, filepath.Join("..", "..", "migrations", "001_initial_schema.sql"))
	executeFile(t, db, filepath.Join("..", "..", "migrations", "002_campaign_games.sql"))
	executeFile(t, db, filepath.Join("..", "..", "seeds", "demo.sql"))
	executeFile(t, db, filepath.Join("..", "..", "seeds", "demo.sql"))
	store := NewStore(db)
	ctx := context.Background()
	wheelCampaign, err := store.CampaignForGame(ctx, "festival-hadiah-ceria", "wheel")
	if err != nil {
		t.Fatal(err)
	}
	clawCampaign, err := store.CampaignForGame(ctx, "festival-hadiah-ceria", "claw")
	if err != nil {
		t.Fatal(err)
	}
	if wheelCampaign.ID != clawCampaign.ID || wheelCampaign.GameType != "wheel" || clawCampaign.GameType != "claw" {
		t.Fatalf("relasi multi-game tidak valid: wheel=%#v claw=%#v", wheelCampaign, clawCampaign)
	}
	token, err := store.CreateSession(ctx, "festival-hadiah-ceria")
	if err != nil {
		t.Fatal(err)
	}
	if _, err = db.Exec(`UPDATE prizes SET is_active=CASE WHEN name='Voucher Rp50.000' THEN 1 ELSE 0 END`); err != nil {
		t.Fatal(err)
	}
	first, err := store.Play(ctx, token)
	if err != nil {
		t.Fatal(err)
	}
	second, err := store.Play(ctx, token)
	if err != nil {
		t.Fatal(err)
	}
	if first.PrizeID != second.PrizeID || first.ClaimCode != second.ClaimCode {
		t.Fatalf("hasil sesi berubah: %#v menjadi %#v", first, second)
	}
	if first.PrizeName != "Voucher Rp50.000" {
		t.Fatalf("hadiah = %q", first.PrizeName)
	}
	var results, stock int
	if err = db.QueryRow(`SELECT COUNT(*) FROM game_results`).Scan(&results); err != nil {
		t.Fatal(err)
	}
	if err = db.QueryRow(`SELECT remaining_stock FROM prizes WHERE name='Voucher Rp50.000'`).Scan(&stock); err != nil {
		t.Fatal(err)
	}
	if results != 1 {
		t.Fatalf("jumlah hasil = %d, ingin 1", results)
	}
	if stock != 9 {
		t.Fatalf("sisa stok = %d, ingin 9", stock)
	}
}

func executeFile(t *testing.T, db *sql.DB, path string) {
	t.Helper()
	content, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if _, err = db.Exec(string(content)); err != nil {
		t.Fatal(err)
	}
}
