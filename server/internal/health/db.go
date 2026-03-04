package health

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
)

var ErrNilDB = errors.New("nil database handle")

// CheckDBReady verifies critical DB startup/readiness conditions.
func CheckDBReady(ctx context.Context, db *sql.DB) error {
	if db == nil {
		return ErrNilDB
	}
	if err := db.PingContext(ctx); err != nil {
		return fmt.Errorf("database ping failed: %w", err)
	}

	var count int
	if err := db.QueryRowContext(ctx, `SELECT COUNT(1) FROM sqlite_master WHERE type='table' AND name='schema_migrations'`).Scan(&count); err != nil {
		return fmt.Errorf("schema_migrations readiness check failed: %w", err)
	}
	if count != 1 {
		return errors.New("schema_migrations table missing")
	}

	return nil
}
