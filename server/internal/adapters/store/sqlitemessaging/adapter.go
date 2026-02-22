package sqlitemessaging

import (
	"context"
	"database/sql"
	"errors"
	"time"

	coremsg "github.com/kyambuthia/go-chat-site/server/internal/core/messaging"
)

type Adapter struct {
	DB *sql.DB
}

var _ coremsg.MessageRepository = (*Adapter)(nil)

func (a *Adapter) SaveDirectMessage(ctx context.Context, msg coremsg.StoredMessage) (coremsg.StoredMessage, error) {
	result, err := a.DB.ExecContext(ctx, `
		INSERT INTO messages (from_user_id, to_user_id, body)
		VALUES (?, ?, ?)
	`, msg.FromUserID, msg.ToUserID, msg.Body)
	if err != nil {
		return coremsg.StoredMessage{}, err
	}
	id, err := result.LastInsertId()
	if err != nil {
		return coremsg.StoredMessage{}, err
	}

	stored, err := a.getByID(ctx, id)
	if err != nil {
		return coremsg.StoredMessage{}, err
	}
	return stored, nil
}

func (a *Adapter) MarkDelivered(ctx context.Context, messageID int64, deliveredAt time.Time) error {
	_, err := a.DB.ExecContext(ctx, `
		INSERT INTO message_deliveries (message_id, delivered_at)
		VALUES (?, ?)
		ON CONFLICT(message_id) DO UPDATE SET
			delivered_at = COALESCE(message_deliveries.delivered_at, excluded.delivered_at)
	`, messageID, deliveredAt.UTC())
	return err
}

func (a *Adapter) MarkDeliveredForRecipient(ctx context.Context, recipientUserID int, messageID int64, deliveredAt time.Time) error {
	if err := a.ensureRecipientOwnsMessage(ctx, recipientUserID, messageID); err != nil {
		return err
	}
	return a.MarkDelivered(ctx, messageID, deliveredAt)
}

func (a *Adapter) MarkRead(ctx context.Context, messageID int64, readAt time.Time) error {
	readAt = readAt.UTC()
	_, err := a.DB.ExecContext(ctx, `
		INSERT INTO message_deliveries (message_id, delivered_at, read_at)
		VALUES (?, ?, ?)
		ON CONFLICT(message_id) DO UPDATE SET
			delivered_at = COALESCE(message_deliveries.delivered_at, excluded.delivered_at),
			read_at = COALESCE(message_deliveries.read_at, excluded.read_at)
	`, messageID, readAt, readAt)
	return err
}

func (a *Adapter) MarkReadForRecipient(ctx context.Context, recipientUserID int, messageID int64, readAt time.Time) error {
	if err := a.ensureRecipientOwnsMessage(ctx, recipientUserID, messageID); err != nil {
		return err
	}
	return a.MarkRead(ctx, messageID, readAt)
}

func (a *Adapter) ListInbox(ctx context.Context, userID int, limit int) ([]coremsg.StoredMessage, error) {
	return a.listInboxQuery(ctx, userID, 0, 0, limit)
}

func (a *Adapter) ListInboxWithUser(ctx context.Context, userID int, withUserID int, limit int) ([]coremsg.StoredMessage, error) {
	return a.listInboxQuery(ctx, userID, withUserID, 0, limit)
}

func (a *Adapter) ListInboxBefore(ctx context.Context, userID int, beforeID int64, limit int) ([]coremsg.StoredMessage, error) {
	return a.listInboxQuery(ctx, userID, 0, beforeID, limit)
}

func (a *Adapter) ListInboxBeforeWithUser(ctx context.Context, userID int, withUserID int, beforeID int64, limit int) ([]coremsg.StoredMessage, error) {
	return a.listInboxQuery(ctx, userID, withUserID, beforeID, limit)
}

func (a *Adapter) ListInboxAfter(ctx context.Context, userID int, afterID int64, limit int) ([]coremsg.StoredMessage, error) {
	return a.listInboxAfterQuery(ctx, userID, 0, afterID, limit)
}

func (a *Adapter) ListInboxAfterWithUser(ctx context.Context, userID int, withUserID int, afterID int64, limit int) ([]coremsg.StoredMessage, error) {
	return a.listInboxAfterQuery(ctx, userID, withUserID, afterID, limit)
}

func (a *Adapter) listInboxAfterQuery(ctx context.Context, userID int, withUserID int, afterID int64, limit int) ([]coremsg.StoredMessage, error) {
	query := `
		SELECT m.id, m.from_user_id, m.to_user_id, m.body, m.created_at,
		       md.delivered_at, md.read_at
		FROM messages m
		LEFT JOIN message_deliveries md ON md.message_id = m.id
		WHERE m.to_user_id = ? AND m.id > ?`
	args := []any{userID, afterID}
	if withUserID > 0 {
		query += ` AND (m.from_user_id = ? OR m.to_user_id = ?)`
		args = append(args, withUserID, withUserID)
	}
	query += `
		ORDER BY m.id ASC
		LIMIT ?`
	args = append(args, limit)

	rows, err := a.DB.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	out := make([]coremsg.StoredMessage, 0)
	for rows.Next() {
		msg, err := scanStoredMessage(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, msg)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return out, nil
}

func (a *Adapter) GetMessageForRecipient(ctx context.Context, recipientUserID int, messageID int64) (coremsg.StoredMessage, error) {
	row := a.DB.QueryRowContext(ctx, `
		SELECT m.id, m.from_user_id, m.to_user_id, m.body, m.created_at,
		       md.delivered_at, md.read_at
		FROM messages m
		LEFT JOIN message_deliveries md ON md.message_id = m.id
		WHERE m.id = ? AND m.to_user_id = ?
	`, messageID, recipientUserID)
	msg, err := scanStoredMessage(row)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return coremsg.StoredMessage{}, coremsg.ErrMessageNotFound
		}
		return coremsg.StoredMessage{}, err
	}
	return msg, nil
}

func (a *Adapter) listInboxQuery(ctx context.Context, userID int, withUserID int, beforeID int64, limit int) ([]coremsg.StoredMessage, error) {
	query := `
		SELECT m.id, m.from_user_id, m.to_user_id, m.body, m.created_at,
		       md.delivered_at, md.read_at
		FROM messages m
		LEFT JOIN message_deliveries md ON md.message_id = m.id
		WHERE m.to_user_id = ?`
	args := []any{userID}
	if withUserID > 0 {
		query += ` AND (m.from_user_id = ? OR m.to_user_id = ?)`
		args = append(args, withUserID, withUserID)
	}
	if beforeID > 0 {
		query += ` AND m.id < ?`
		args = append(args, beforeID)
	}
	query += `
		ORDER BY m.id DESC
		LIMIT ?`
	args = append(args, limit)

	rows, err := a.DB.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	out := make([]coremsg.StoredMessage, 0)
	for rows.Next() {
		msg, err := scanStoredMessage(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, msg)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return out, nil
}

func (a *Adapter) getByID(ctx context.Context, id int64) (coremsg.StoredMessage, error) {
	row := a.DB.QueryRowContext(ctx, `
		SELECT m.id, m.from_user_id, m.to_user_id, m.body, m.created_at,
		       md.delivered_at, md.read_at
		FROM messages m
		LEFT JOIN message_deliveries md ON md.message_id = m.id
		WHERE m.id = ?
	`, id)
	return scanStoredMessage(row)
}

type scanner interface {
	Scan(dest ...any) error
}

func scanStoredMessage(s scanner) (coremsg.StoredMessage, error) {
	var msg coremsg.StoredMessage
	var createdAt time.Time
	var deliveredAt sql.NullTime
	var readAt sql.NullTime
	if err := s.Scan(&msg.ID, &msg.FromUserID, &msg.ToUserID, &msg.Body, &createdAt, &deliveredAt, &readAt); err != nil {
		return coremsg.StoredMessage{}, err
	}
	msg.CreatedAt = createdAt
	if deliveredAt.Valid {
		t := deliveredAt.Time
		msg.DeliveredAt = &t
	}
	if readAt.Valid {
		t := readAt.Time
		msg.ReadAt = &t
	}
	return msg, nil
}

func (a *Adapter) ensureRecipientOwnsMessage(ctx context.Context, recipientUserID int, messageID int64) error {
	var exists int
	err := a.DB.QueryRowContext(ctx, `
		SELECT 1
		FROM messages
		WHERE id = ? AND to_user_id = ?
	`, messageID, recipientUserID).Scan(&exists)
	if err == nil {
		return nil
	}
	if errors.Is(err, sql.ErrNoRows) {
		return coremsg.ErrMessageNotFound
	}
	return err
}
