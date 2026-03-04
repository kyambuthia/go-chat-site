package migrate

import (
	"database/sql"
	"os"
	"path/filepath"
	"testing"

	_ "github.com/mattn/go-sqlite3"
)

func TestRunMigrations_AppliesInOrderAndIsIdempotent(t *testing.T) {
	db, err := sql.Open("sqlite3", ":memory:")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = db.Close() })

	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "0001_create.sql"), []byte(`CREATE TABLE test_items (id INTEGER PRIMARY KEY, name TEXT);`), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "0002_seed.sql"), []byte(`INSERT INTO test_items (name) VALUES ('hello');`), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "README.txt"), []byte(`ignore me`), 0o644); err != nil {
		t.Fatal(err)
	}

	if err := RunMigrations(db, dir); err != nil {
		t.Fatalf("first RunMigrations failed: %v", err)
	}
	if err := RunMigrations(db, dir); err != nil {
		t.Fatalf("second RunMigrations failed: %v", err)
	}

	var rowCount int
	if err := db.QueryRow(`SELECT COUNT(*) FROM test_items`).Scan(&rowCount); err != nil {
		t.Fatal(err)
	}
	if rowCount != 1 {
		t.Fatalf("seed row count = %d, want 1", rowCount)
	}

	var migrationCount int
	if err := db.QueryRow(`SELECT COUNT(*) FROM schema_migrations`).Scan(&migrationCount); err != nil {
		t.Fatal(err)
	}
	if migrationCount != 2 {
		t.Fatalf("schema_migrations count = %d, want 2", migrationCount)
	}
}

func TestRunMigrations_FailsOnInvalidSQL(t *testing.T) {
	db, err := sql.Open("sqlite3", ":memory:")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = db.Close() })

	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "0001_bad.sql"), []byte(`CREAT TABLE broken (`), 0o644); err != nil {
		t.Fatal(err)
	}

	if err := RunMigrations(db, dir); err == nil {
		t.Fatal("expected migration failure for invalid sql")
	}
}
