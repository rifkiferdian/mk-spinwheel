package main

import (
	"database/sql"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"game-spinwheel/internal/admin"
	"game-spinwheel/internal/game"
	_ "modernc.org/sqlite"
)

func main() {
	logger := log.New(os.Stdout, "spinwheel ", log.LstdFlags|log.LUTC)
	databasePath := env("DATABASE_PATH", "data/game.db")
	if err := os.MkdirAll(filepath.Dir(databasePath), 0o755); err != nil {
		logger.Fatal(err)
	}
	db, err := sql.Open("sqlite", databasePath)
	if err != nil {
		logger.Fatal(err)
	}
	defer db.Close()
	db.SetMaxOpenConns(1)
	db.SetConnMaxLifetime(0)
	if _, err = db.Exec(`PRAGMA foreign_keys=ON; PRAGMA journal_mode=WAL; PRAGMA busy_timeout=5000;`); err != nil {
		logger.Fatal(err)
	}
	for _, migrationPath := range []string{"migrations/001_initial_schema.sql", "migrations/002_campaign_games.sql", "migrations/003_rename_mystery_box.sql", "migrations/004_admin_roles.sql"} {
		if err = applySchema(db, migrationPath); err != nil {
			logger.Fatal(err)
		}
	}
	views, err := admin.NewViews()
	if err != nil {
		logger.Fatal(err)
	}
	secureCookie := strings.EqualFold(env("SECURE_COOKIE", "false"), "true")
	gameServer, err := game.NewServer(game.NewStore(db), logger)
	if err != nil {
		logger.Fatal(err)
	}
	server := admin.NewServer(admin.NewStore(db), views, admin.NewSessionManager(secureCookie), logger)
	httpServer := &http.Server{Addr: env("APP_ADDR", ":8080"), Handler: server.Routes(http.FileServer(http.Dir("static")), gameServer.Handler()), ReadHeaderTimeout: 5 * time.Second, ReadTimeout: 15 * time.Second, WriteTimeout: 30 * time.Second, IdleTimeout: 60 * time.Second}
	logger.Printf("admin tersedia di http://localhost%s/admin", httpServer.Addr)
	if err := httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		logger.Fatal(err)
	}
}

func applySchema(db *sql.DB, path string) error {
	content, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	_, err = db.Exec(string(content))
	return err
}
func env(name, fallback string) string {
	if value := strings.TrimSpace(os.Getenv(name)); value != "" {
		return value
	}
	return fallback
}
