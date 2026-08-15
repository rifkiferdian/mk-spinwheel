package main

import (
	"context"
	"crypto/rand"
	"database/sql"
	"flag"
	"fmt"
	"log"
	"math/big"

	"game-spinwheel/internal/admin"
	_ "modernc.org/sqlite"
)

func main() {
	databasePath := flag.String("db", "data/game.db", "lokasi database SQLite")
	username := flag.String("username", "campaignadmin", "username akun")
	updateFrom := flag.String("update-from", "", "username lama yang akan diperbarui")
	password := flag.String("password", "", "password akun; kosong untuk membuat otomatis")
	role := flag.String("role", admin.RoleCampaignAdmin, "role akun")
	list := flag.Bool("list", false, "tampilkan akun dan role tanpa password")
	flag.Parse()

	db, err := sql.Open("sqlite", *databasePath)
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()
	db.SetMaxOpenConns(1)
	if _, err = db.Exec("PRAGMA foreign_keys=ON; PRAGMA busy_timeout=5000;"); err != nil {
		log.Fatal(err)
	}
	store := admin.NewStore(db)
	if *list {
		users, listErr := store.Admins(context.Background())
		if listErr != nil {
			log.Fatal(listErr)
		}
		for _, user := range users {
			fmt.Printf("%s\t%s\taktif=%t\n", user.Username, user.Role, user.IsActive)
		}
		return
	}

	generated := false
	if *password == "" {
		*password, err = generatePassword(18)
		if err != nil {
			log.Fatal(err)
		}
		generated = true
	}
	if *updateFrom != "" {
		if err = store.UpdateAdminCredentials(context.Background(), *updateFrom, *username, *password); err != nil {
			log.Fatal(err)
		}
		fmt.Printf("Akun berhasil diperbarui\nUsername lama: %s\nUsername baru: %s\n", *updateFrom, *username)
		if generated {
			fmt.Printf("Password sementara: %s\n", *password)
		}
		return
	}
	if err = store.CreateAdmin(context.Background(), *username, *password, *role); err != nil {
		log.Fatal(err)
	}

	fmt.Printf("Akun berhasil dibuat\nUsername: %s\nRole: %s\n", *username, *role)
	if generated {
		fmt.Printf("Password sementara: %s\n", *password)
	}
}

func generatePassword(length int) (string, error) {
	const characters = "ABCDEFGHJKLMNPQRSTUVWXYZabcdefghijkmnopqrstuvwxyz23456789!@#$%"
	result := make([]byte, length)
	for index := range result {
		value, err := rand.Int(rand.Reader, big.NewInt(int64(len(characters))))
		if err != nil {
			return "", err
		}
		result[index] = characters[value.Int64()]
	}
	return string(result), nil
}
