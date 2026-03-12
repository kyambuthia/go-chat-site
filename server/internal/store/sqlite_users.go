package store

import (
	"database/sql"
	"errors"
	"strings"

	"github.com/kyambuthia/go-chat-site/server/internal/crypto"
)

func (s *SqliteStore) CreateUser(username, password string) (int, error) {
	username = strings.TrimSpace(username)
	if username == "" {
		return 0, errors.New("username is required")
	}

	hashedPassword, err := crypto.HashPassword(password)
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

func (s *SqliteStore) GetUserByUsername(username string) (*User, error) {
	row := s.DB.QueryRow(`
		SELECT id, username, COALESCE(display_name, ''), COALESCE(avatar_url, ''), password_hash
		FROM users
		WHERE username = ?
	`, strings.TrimSpace(username))

	user := &User{}
	if err := row.Scan(&user.ID, &user.Username, &user.DisplayName, &user.AvatarURL, &user.PasswordHash); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, err
	}
	return user, nil
}

func (s *SqliteStore) GetUserByID(id int) (*User, error) {
	row := s.DB.QueryRow(`
		SELECT id, username, COALESCE(display_name, ''), COALESCE(avatar_url, ''), password_hash
		FROM users
		WHERE id = ?
	`, id)

	user := &User{}
	if err := row.Scan(&user.ID, &user.Username, &user.DisplayName, &user.AvatarURL, &user.PasswordHash); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, err
	}
	return user, nil
}

func (s *SqliteStore) UpdateUserProfile(id int, displayName, avatarURL string) (*User, error) {
	result, err := s.DB.Exec(`
		UPDATE users
		SET display_name = ?, avatar_url = ?
		WHERE id = ?
	`, strings.TrimSpace(displayName), strings.TrimSpace(avatarURL), id)
	if err != nil {
		return nil, err
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return nil, err
	}
	if rowsAffected == 0 {
		return nil, ErrNotFound
	}

	return s.GetUserByID(id)
}
