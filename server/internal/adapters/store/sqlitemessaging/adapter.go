package sqlitemessaging

import (
	"context"
	"database/sql"
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

func (a *Adapter) ListInbox(ctx context.Context, userID int, limit int) ([]coremsg.StoredMessage, error) {
	rows, err := a.DB.QueryContext(ctx, `
		SELECT m.id, m.from_user_id, m.to_user_id, m.body, m.created_at,
		       md.delivered_at, md.read_at
		FROM messages m
		LEFT JOIN message_deliveries md ON md.message_id = m.id
		WHERE m.to_user_id = ?
		ORDER BY m.id DESC
		LIMIT ?
	`, userID, limit)
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
