package main

import (
	"database/sql"
	"flag"
	"fmt"
	"log"
	"os"
	"path/filepath"

	_ "modernc.org/sqlite"
)

func main() {
	databasePath := flag.String("db", "data/game.db", "lokasi file database SQLite")
	withDemo := flag.Bool("seed", false, "tambahkan campaign dan hadiah demo")
	flag.Parse()

	if err := os.MkdirAll(filepath.Dir(*databasePath), 0o755); err != nil {
		log.Fatalf("membuat direktori database: %v", err)
	}

	db, err := sql.Open("sqlite", *databasePath)
	if err != nil {
		log.Fatalf("membuka database: %v", err)
	}
	defer db.Close()

	// Satu koneksi membuat perilaku transaksi SQLite lebih mudah diprediksi.
	db.SetMaxOpenConns(1)

	if _, err := db.Exec(`
		PRAGMA foreign_keys = ON;
		PRAGMA journal_mode = WAL;
		PRAGMA busy_timeout = 5000;
	`); err != nil {
		log.Fatalf("mengatur SQLite: %v", err)
	}

	for _, migrationPath := range []string{"migrations/001_initial_schema.sql", "migrations/002_campaign_games.sql"} {
		if err := executeSQLFile(db, migrationPath); err != nil {
			log.Fatalf("menjalankan migration: %v", err)
		}
	}

	if *withDemo {
		for _, seedPath := range []string{"seeds/demo.sql", "seeds/claw_demo.sql"} {
			if err := executeSQLFile(db, seedPath); err != nil {
				log.Fatalf("menambahkan data demo: %v", err)
			}
		}
	}

	var campaignCount, prizeCount int
	if err := db.QueryRow("SELECT COUNT(*) FROM campaigns").Scan(&campaignCount); err != nil {
		log.Fatalf("memeriksa campaign: %v", err)
	}
	if err := db.QueryRow("SELECT COUNT(*) FROM prizes").Scan(&prizeCount); err != nil {
		log.Fatalf("memeriksa hadiah: %v", err)
	}

	fmt.Printf("Database siap: %s\n", *databasePath)
	fmt.Printf("Campaign: %d, hadiah: %d\n", campaignCount, prizeCount)
}

func executeSQLFile(db *sql.DB, path string) error {
	content, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("membaca %s: %w", path, err)
	}

	if _, err := db.Exec(string(content)); err != nil {
		return fmt.Errorf("mengeksekusi %s: %w", path, err)
	}
	return nil
}
