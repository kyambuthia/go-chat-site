package store

import (
	"database/sql"
	"errors"

	_ "github.com/mattn/go-sqlite3"
)

var (
	ErrNotFound         = errors.New("not found")
	ErrInviteExists     = errors.New("an invite already exists between these users")
	ErrInsufficientFund = errors.New("insufficient funds")
)

type SqliteStore struct {
	DB *sql.DB
}

// SQLDB exposes the underlying database handle for adapter composition.
func (s *SqliteStore) SQLDB() *sql.DB {
	if s == nil {
		return nil
	}
	return s.DB
}

type User struct {
	ID           int    `json:"id"`
	Username     string `json:"username"`
	DisplayName  string `json:"display_name"`
	AvatarURL    string `json:"avatar_url"`
	PasswordHash string `json:"-"`
}

type Invite struct {
	ID              int    `json:"id"`
	InviterUsername string `json:"inviter_username"`
}

func NewSqliteStore(dataSourceName string) (*SqliteStore, error) {
	db, err := sql.Open("sqlite3", dataSourceName)
	if err != nil {
		return nil, err
	}

	if _, err := db.Exec("PRAGMA foreign_keys = ON;"); err != nil {
		return nil, err
	}

	if err := db.Ping(); err != nil {
		return nil, err
	}

	return &SqliteStore{DB: db}, nil
}
