package store

import (
	"database/sql"

	"github.com/kyambuthia/go-chat-site/server/internal/auth"
	_ "github.com/mattn/go-sqlite3"
)

type SqliteStore struct {
	DB *sql.DB
}

type User struct {
	ID       int    `json:"id"`
	Username string `json:"username"`
}

func NewSqliteStore(dataSourceName string) (*SqliteStore, error) {
	db, err := sql.Open("sqlite3", dataSourceName)
	if err != nil {
		return nil, err
	}

	if err := db.Ping(); err != nil {
		return nil, err
	}

	return &SqliteStore{DB: db}, nil
}

func (s *SqliteStore) CreateUser(username, password string) (int, error) {
	hashedPassword, err := auth.HashPassword(password)
	if err != nil {
		return 0, err
	}

	result, err := s.DB.Exec(`INSERT INTO users (username, password_hash) VALUES (?, ?)`, username, hashedPassword)
	if err != nil {
		return 0, err
	}

	id, err := result.LastInsertId()
	if err != nil {
		return 0, err
	}

	return int(id), nil
}

func (s *SqliteStore) GetUserByUsername(username string) (*sql.Row, error) {
	row := s.DB.QueryRow(`SELECT id, password_hash FROM users WHERE username = ?`, username)
	return row, nil
}

func (s *SqliteStore) GetContacts(userID int) (*sql.Rows, error) {
	rows, err := s.DB.Query(`
		SELECT u.id, u.username
		FROM users u
		INNER JOIN contacts c ON u.id = c.contact_id
		WHERE c.user_id = ?
	`, userID)
	if err != nil {
		return nil, err
	}
	return rows, nil
}

func (s *SqliteStore) AddContact(userID, contactID int) error {
	_, err := s.DB.Exec(`INSERT OR IGNORE INTO contacts (user_id, contact_id) VALUES (?, ?)`, userID, contactID)
	return err
}

func (s *SqliteStore) RemoveContact(userID, contactID int) error {
	_, err := s.DB.Exec(`DELETE FROM contacts WHERE user_id = ? AND contact_id = ?`, userID, contactID)
	return err
}