package main

import (
	"database/sql"
	"log"

	_ "github.com/mattn/go-sqlite3"
	"github.com/kyambuthia/go-chat-site/server/internal/migrate"
)

func main() {
	db, err := sql.Open("sqlite3", "../../chat.db")
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	if err := migrate.RunMigrations(db, "/home/brutus/projects/go-chat-site/server/migrations"); err != nil {
		log.Fatal(err)
	}
}
