package test

import (
	"database/sql"
	"testing"

	"github.com/kyambuthia/go-chat-site/server/test/testhelpers"
	_ "github.com/mattn/go-sqlite3"
)

func TestMigrations(t *testing.T) {
	db, err := sql.Open("sqlite3", ":memory:")
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	err = testhelpers.RunMigrations(db, "../migrations")
	if err != nil {
		t.Fatal(err)
	}

	t.Run("tables exist", func(t *testing.T) {
		rows, err := db.Query("SELECT name FROM sqlite_master WHERE type='table' AND name IN ('users', 'messages')")
		if err != nil {
			t.Fatal(err)
		}
		defer rows.Close()

		var count int
		for rows.Next() {
			count++
		}

		if count != 2 {
			t.Fatalf("expected 2 tables, got %d", count)
		}
	})

	t.Run("unique username constraint", func(t *testing.T) {
		_, err := db.Exec(`INSERT INTO users (username, password_hash) VALUES ('test', 'test')`)
		if err != nil {
			t.Fatal(err)
		}

		_, err = db.Exec(`INSERT INTO users (username, password_hash) VALUES ('test', 'test')`)
		if err == nil {
			t.Fatal("expected unique constraint error, got nil")
		}
	})

	t.Run("foreign key constraint", func(t *testing.T) {
		_, err := db.Exec(`INSERT INTO messages (from_user_id, to_user_id, body) VALUES (999, 999, 'test')`)
		if err == nil {
			t.Fatal("expected foreign key constraint error, got nil")
		}
	})
}
