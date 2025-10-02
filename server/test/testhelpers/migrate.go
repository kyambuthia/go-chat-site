package testhelpers

import (
	"database/sql"
	"fmt"
	"io/ioutil"
	"path/filepath"
	"sort"
	"strings"
)

// RunMigrations applies all .sql files in a directory that have not yet been applied.
// It uses a schema_migrations table to track which migrations are already done.
func RunMigrations(db *sql.DB, dir string) error {
	// Ensure the migrations table exists.
	_, err := db.Exec(`CREATE TABLE IF NOT EXISTS schema_migrations (version TEXT PRIMARY KEY)`)
	if err != nil {
		return fmt.Errorf("could not create schema_migrations table: %w", err)
	}

	files, err := ioutil.ReadDir(dir)
	if err != nil {
		return err
	}

	sort.Slice(files, func(i, j int) bool {
		return files[i].Name() < files[j].Name()
	})

	tx, err := db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback() // Rollback on error, commit on success

	for _, file := range files {
		if file.IsDir() || !strings.HasSuffix(file.Name(), ".sql") {
			continue
		}

		version := strings.Split(file.Name(), "_")[0]

		var appliedVersion string
		err := tx.QueryRow("SELECT version FROM schema_migrations WHERE version = ?", version).Scan(&appliedVersion)
		if err != nil && err != sql.ErrNoRows {
			return fmt.Errorf("could not query schema_migrations for version %s: %w", version, err)
		}

		// If the migration has already been applied, skip it.
		if appliedVersion == version {
			continue
		}

		content, err := ioutil.ReadFile(filepath.Join(dir, file.Name()))
		if err != nil {
			return err
		}

		if _, err = tx.Exec(string(content)); err != nil {
			return fmt.Errorf("failed to execute migration %s: %w", file.Name(), err)
		}

		if _, err = tx.Exec("INSERT INTO schema_migrations (version) VALUES (?)", version); err != nil {
			return fmt.Errorf("failed to record migration version %s: %w", version, err)
		}
	}

	return tx.Commit()
}
