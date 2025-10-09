package store

import (
	"database/sql"
	"errors"

	"github.com/kyambuthia/go-chat-site/server/internal/crypto"
	_ "github.com/mattn/go-sqlite3"
)

type SqliteStore struct {
	DB *sql.DB
}

type User struct {
	ID           int    `json:"id"`
	Username     string `json:"username"`
	DisplayName  string `json:"display_name"`
	AvatarURL    string `json:"avatar_url"`
	PasswordHash string `json:"-"`
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
	row := s.DB.QueryRow(`SELECT id, password_hash FROM users WHERE username = ?`, username)
	user := &User{Username: username}
	err := row.Scan(&user.ID, &user.PasswordHash)
	if err != nil {
		return nil, err
	}
	return user, nil
}

func (s *SqliteStore) GetUserByID(id int) (*User, error) {
	row := s.DB.QueryRow(`SELECT id, username FROM users WHERE id = ?`, id)
	user := &User{}
	err := row.Scan(&user.ID, &user.Username)
	if err != nil {
		return nil, err
	}
	return user, nil
}

func (s *SqliteStore) GetContacts(userID int) (*sql.Rows, error) {
	rows, err := s.DB.Query(`
		SELECT u.id, u.username, COALESCE(u.display_name, ''), COALESCE(u.avatar_url, '')
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

func (s *SqliteStore) CreateInvite(inviterID, inviteeID int) error {
	_, err := s.DB.Exec(`INSERT INTO invites (inviter_id, invitee_id) VALUES (?, ?)`, inviterID, inviteeID)
	return err
}

func (s *SqliteStore) GetInvites(userID int) (*sql.Rows, error) {
	rows, err := s.DB.Query(`
		SELECT i.id, u.username
		FROM invites i
		INNER JOIN users u ON i.inviter_id = u.id
		WHERE i.invitee_id = ? AND i.status = 'pending'
	`, userID)
	if err != nil {
		return nil, err
	}
	return rows, nil
}

func (s *SqliteStore) UpdateInviteStatus(inviteID, userID int, status string) error {
	tx, err := s.DB.Begin()
	if err != nil {
		return err
	}

	var inviterID int
	row := tx.QueryRow(`SELECT inviter_id FROM invites WHERE id = ? AND invitee_id = ?`, inviteID, userID)
	if err := row.Scan(&inviterID); err != nil {
		tx.Rollback()
		return err
	}

	_, err = tx.Exec(`UPDATE invites SET status = ? WHERE id = ?`, status, inviteID)
	if err != nil {
		tx.Rollback()
		return err
	}

	if status == "accepted" {
		_, err = tx.Exec(`INSERT INTO contacts (user_id, contact_id) VALUES (?, ?), (?, ?)`, userID, inviterID, inviterID, userID)
		if err != nil {
			tx.Rollback()
			return err
		}
	}

	return tx.Commit()
}

type Wallet struct {
	ID      int     `json:"id"`
	UserID  int     `json:"user_id"`
	Balance float64 `json:"balance"`
}

func (s *SqliteStore) GetWallet(userID int) (*Wallet, error) {
	row := s.DB.QueryRow(`SELECT id, user_id, balance FROM wallets WHERE user_id = ?`, userID)

	var wallet Wallet
	if err := row.Scan(&wallet.ID, &wallet.UserID, &wallet.Balance); err != nil {
		if err == sql.ErrNoRows {
			// Create a new wallet if one doesn't exist
			_, err := s.DB.Exec(`INSERT INTO wallets (user_id) VALUES (?)`, userID)
			if err != nil {
				return nil, err
			}
			return s.GetWallet(userID)
		}
		return nil, err
	}

	return &wallet, nil
}

func (s *SqliteStore) SendMoney(senderID, recipientID int, amount float64) error {
	tx, err := s.DB.Begin()
	if err != nil {
		return err
	}

	// Get sender's balance
	var senderBalance float64
	row := tx.QueryRow(`SELECT balance FROM wallets WHERE user_id = ?`, senderID)
	if err := row.Scan(&senderBalance); err != nil {
		tx.Rollback()
		return err
	}

	if senderBalance < amount {
		tx.Rollback()
		return errors.New("insufficient funds")
	}

	// Get recipient's balance
	var recipientBalance float64
	row = tx.QueryRow(`SELECT balance FROM wallets WHERE user_id = ?`, recipientID)
	if err := row.Scan(&recipientBalance); err != nil {
		tx.Rollback()
		return err
	}

	// Update balances
	_, err = tx.Exec(`UPDATE wallets SET balance = ? WHERE user_id = ?`, senderBalance-amount, senderID)
	if err != nil {
		tx.Rollback()
		return err
	}

	_, err = tx.Exec(`UPDATE wallets SET balance = ? WHERE user_id = ?`, recipientBalance+amount, recipientID)
	if err != nil {
		tx.Rollback()
		return err
	}

	return tx.Commit()
}