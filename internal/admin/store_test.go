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
}
