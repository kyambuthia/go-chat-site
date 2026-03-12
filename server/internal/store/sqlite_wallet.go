package store

import (
	"database/sql"
	"errors"
	"math"
	"time"
)

type Wallet struct {
	ID           int   `json:"id"`
	UserID       int   `json:"user_id"`
	BalanceCents int64 `json:"balance_cents"`
}

type WalletTransfer struct {
	ID                      int
	Direction               string
	CounterpartyUserID      int
	CounterpartyUsername    string
	CounterpartyDisplayName string
	CounterpartyAvatarURL   string
	AmountCents             int64
	CreatedAt               time.Time
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

func (s *SqliteStore) ListTransfers(userID, limit int) ([]WalletTransfer, error) {
	if limit <= 0 {
		limit = 20
	}

	rows, err := s.DB.Query(`
		SELECT
			wt.id,
			CASE WHEN wt.sender_user_id = ? THEN 'sent' ELSE 'received' END AS direction,
			CASE WHEN wt.sender_user_id = ? THEN wt.recipient_user_id ELSE wt.sender_user_id END AS counterparty_user_id,
			u.username,
			COALESCE(u.display_name, ''),
			COALESCE(u.avatar_url, ''),
			wt.amount_cents,
			wt.created_at
		FROM wallet_transfers wt
		INNER JOIN users u
			ON u.id = CASE WHEN wt.sender_user_id = ? THEN wt.recipient_user_id ELSE wt.sender_user_id END
		WHERE wt.sender_user_id = ? OR wt.recipient_user_id = ?
		ORDER BY wt.created_at DESC, wt.id DESC
		LIMIT ?
	`, userID, userID, userID, userID, userID, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	transfers := make([]WalletTransfer, 0)
	for rows.Next() {
		var transfer WalletTransfer
		if err := rows.Scan(
			&transfer.ID,
			&transfer.Direction,
			&transfer.CounterpartyUserID,
			&transfer.CounterpartyUsername,
			&transfer.CounterpartyDisplayName,
			&transfer.CounterpartyAvatarURL,
			&transfer.AmountCents,
			&transfer.CreatedAt,
		); err != nil {
			return nil, err
		}
		transfers = append(transfers, transfer)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	return transfers, nil
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
