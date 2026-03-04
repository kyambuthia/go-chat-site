package health

import (
	"context"
	"database/sql"
	"path/filepath"
	"testing"
	"time"

	"github.com/kyambuthia/go-chat-site/server/internal/migrate"
	_ "github.com/mattn/go-sqlite3"
)

func TestCheckDBReady_RejectsNilDB(t *testing.T) {
	err := CheckDBReady(context.Background(), nil)
	if err == nil {
		t.Fatal("expected error for nil db")
	}
}

func TestCheckDBReady_FailsWhenMigrationsTableMissing(t *testing.T) {
	db, err := sql.Open("sqlite3", ":memory:")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = db.Close() })

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	err = CheckDBReady(ctx, db)
	if err == nil {
		t.Fatal("expected readiness error without schema_migrations")
	}
}

func TestCheckDBReady_SucceedsAfterMigrations(t *testing.T) {
	db, err := sql.Open("sqlite3", ":memory:")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = db.Close() })

	if err := migrate.RunMigrations(db, filepath.Join("..", "..", "migrations")); err != nil {
		t.Fatal(err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	if err := CheckDBReady(ctx, db); err != nil {
		t.Fatalf("expected ready db, got %v", err)
	}
}
