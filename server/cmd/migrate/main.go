package main

import (
	"database/sql"
	"errors"
	"log"
	"os"
	"path/filepath"

	_ "github.com/mattn/go-sqlite3"
	"github.com/kyambuthia/go-chat-site/server/internal/migrate"
)

func main() {
	root, err := findProjectRoot()
	if err != nil {
		log.Fatal("Failed to find project root:", err)
	}
	dbPath := filepath.Join(root, "chat.db")

	db, err := sql.Open("sqlite3", dbPath)
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	migrationsPath := filepath.Join(root, "server", "migrations")
	if err := migrate.RunMigrations(db, migrationsPath); err != nil {
		log.Fatal(err)
	}
}

func findProjectRoot() (string, error) {
	dir, err := os.Getwd()
	if err != nil {
		return "", err
	}
	for {
		if _, err := os.Stat(filepath.Join(dir, "go.mod")); err == nil {
			return dir, nil
		}
		if dir == "/" {
			return "", errors.New("go.mod not found")
		}
		dir = filepath.Dir(dir)
	}
}