package store

import (
	"database/sql"
	"errors"
	"fmt"
	"math"
	"strings"

	"github.com/kyambuthia/go-chat-site/server/internal/crypto"
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

type Wallet struct {
	ID           int   `json:"id"`
	UserID       int   `json:"user_id"`
	BalanceCents int64 `json:"balance_cents"`
}

func (w Wallet) BalanceFloat() float64 {
	return float64(w.BalanceCents) / 100.0
}

func DollarsToCents(amount float64) (int64, error) {
	if amount <= 0 {
		return 0, errors.New("amount must be greater than zero")
	}
	cents := int64(math.Round(amount * 100.0))
	if cents <= 0 {
		return 0, errors.New("amount is too small")
	}
	return cents, nil
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

func (s *SqliteStore) ListContacts(userID int) ([]User, error) {
	rows, err := s.DB.Query(`
		SELECT u.id, u.username, COALESCE(u.display_name, ''), COALESCE(u.avatar_url, '')
		FROM users u
		INNER JOIN contacts c ON u.id = c.contact_id
		WHERE c.user_id = ?
		ORDER BY COALESCE(u.display_name, u.username), u.username
	`, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	contacts := make([]User, 0)
	for rows.Next() {
		var contact User
		if err := rows.Scan(&contact.ID, &contact.Username, &contact.DisplayName, &contact.AvatarURL); err != nil {
			return nil, err
		}
		contacts = append(contacts, contact)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	return contacts, nil
}

func (s *SqliteStore) AddContact(userID, contactID int) error {
	if userID == contactID {
		return errors.New("you cannot add yourself as a contact")
	}
	_, err := s.DB.Exec(`INSERT OR IGNORE INTO contacts (user_id, contact_id) VALUES (?, ?)`, userID, contactID)
	return err
}

func (s *SqliteStore) RemoveContact(userID, contactID int) error {
	_, err := s.DB.Exec(`DELETE FROM contacts WHERE user_id = ? AND contact_id = ?`, userID, contactID)
	return err
}

func (s *SqliteStore) CreateInvite(inviterID, inviteeID int) error {
	if inviterID == inviteeID {
		return errors.New("you cannot invite yourself")
	}

	_, err := s.DB.Exec(`
		INSERT INTO contact_invites (requester_id, recipient_id, status)
		VALUES (?, ?, 'pending')
	`, inviterID, inviteeID)
	if err != nil {
		if strings.Contains(strings.ToLower(err.Error()), "unique") {
			return ErrInviteExists
		}
		return err
	}
	return nil
}

func (s *SqliteStore) ListInvites(userID int) ([]Invite, error) {
	rows, err := s.DB.Query(`
		SELECT ci.id, u.username
		FROM contact_invites ci
		INNER JOIN users u ON ci.requester_id = u.id
		WHERE ci.recipient_id = ? AND ci.status = 'pending'
		ORDER BY ci.created_at DESC
	`, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	invites := make([]Invite, 0)
	for rows.Next() {
		var invite Invite
		if err := rows.Scan(&invite.ID, &invite.InviterUsername); err != nil {
			return nil, err
		}
		invites = append(invites, invite)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return invites, nil
}

func (s *SqliteStore) UpdateInviteStatus(inviteID, userID int, status string) error {
	if status != "accepted" && status != "rejected" {
		return fmt.Errorf("unsupported status: %s", status)
	}

	tx, err := s.DB.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	var inviterID int
	row := tx.QueryRow(`
		SELECT requester_id
		FROM contact_invites
		WHERE id = ? AND recipient_id = ? AND status = 'pending'
	`, inviteID, userID)

	if err := row.Scan(&inviterID); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return ErrNotFound
		}
		return err
	}

	_, err = tx.Exec(`
		UPDATE contact_invites
		SET status = ?, updated_at = CURRENT_TIMESTAMP
		WHERE id = ?
	`, status, inviteID)
	if err != nil {
		return err
	}

	if status == "accepted" {
		_, err = tx.Exec(`
			INSERT OR IGNORE INTO contacts (user_id, contact_id)
			VALUES (?, ?), (?, ?)
		`, userID, inviterID, inviterID, userID)
		if err != nil {
			return err
		}
	}

	return tx.Commit()
}

func (s *SqliteStore) GetWallet(userID int) (*Wallet, error) {
	if _, err := s.DB.Exec(`INSERT OR IGNORE INTO wallet_accounts (user_id) VALUES (?)`, userID); err != nil {
		return nil, err
	}

	row := s.DB.QueryRow(`
		SELECT id, user_id, balance_cents
		FROM wallet_accounts
		WHERE user_id = ?
	`, userID)

	var wallet Wallet
	if err := row.Scan(&wallet.ID, &wallet.UserID, &wallet.BalanceCents); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, err
	}

	return &wallet, nil
}

func (s *SqliteStore) SendMoney(senderID, recipientID int, amountCents int64) error {
	if senderID == recipientID {
		return errors.New("cannot transfer to yourself")
	}
	if amountCents <= 0 {
		return errors.New("amount must be greater than zero")
	}

	tx, err := s.DB.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	if _, err := tx.Exec(`INSERT OR IGNORE INTO wallet_accounts (user_id) VALUES (?)`, senderID); err != nil {
		return err
	}
	if _, err := tx.Exec(`INSERT OR IGNORE INTO wallet_accounts (user_id) VALUES (?)`, recipientID); err != nil {
		return err
	}

	var senderBalance int64
	if err := tx.QueryRow(`SELECT balance_cents FROM wallet_accounts WHERE user_id = ?`, senderID).Scan(&senderBalance); err != nil {
		return err
	}
	if senderBalance < amountCents {
		return ErrInsufficientFund
	}

	if _, err := tx.Exec(`
		UPDATE wallet_accounts
		SET balance_cents = balance_cents - ?, updated_at = CURRENT_TIMESTAMP
		WHERE user_id = ?
	`, amountCents, senderID); err != nil {
		return err
	}

	if _, err := tx.Exec(`
		UPDATE wallet_accounts
		SET balance_cents = balance_cents + ?, updated_at = CURRENT_TIMESTAMP
		WHERE user_id = ?
	`, amountCents, recipientID); err != nil {
		return err
	}

	if _, err := tx.Exec(`
		INSERT INTO wallet_transfers (sender_user_id, recipient_user_id, amount_cents)
		VALUES (?, ?, ?)
	`, senderID, recipientID, amountCents); err != nil {
		return err
	}

	return tx.Commit()
}
