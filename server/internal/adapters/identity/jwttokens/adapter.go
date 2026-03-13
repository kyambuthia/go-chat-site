package jwttokens

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"database/sql"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"fmt"
	"sync/atomic"
	"time"

	"github.com/kyambuthia/go-chat-site/server/internal/auth"
	coreid "github.com/kyambuthia/go-chat-site/server/internal/core/identity"
)

const (
	defaultAccessTTL  = 15 * time.Minute
	defaultRefreshTTL = 30 * 24 * time.Hour
)

type Adapter struct {
	DB         *sql.DB
	AccessTTL  time.Duration
	RefreshTTL time.Duration

	cleanupCalls atomic.Uint64
}

var _ coreid.TokenService = (*Adapter)(nil)

func (a *Adapter) IssueSession(ctx context.Context, principal coreid.Principal, meta coreid.SessionMetadata) (coreid.SessionTokens, error) {
	if a.DB == nil {
		token, err := auth.GenerateToken(int(principal.ID))
		if err != nil {
			return coreid.SessionTokens{}, err
		}
		now := time.Now().UTC()
		return coreid.SessionTokens{
			AccessToken:           token,
			AccessTokenExpiresAt:  now.Add(24 * time.Hour),
			RefreshTokenExpiresAt: now.Add(24 * time.Hour),
			Session: coreid.Session{
				UserID:                principal.ID,
				DeviceLabel:           normalizeDeviceLabel(meta.DeviceLabel),
				UserAgent:             meta.UserAgent,
				LastSeenIP:            meta.IPAddress,
				CreatedAt:             now,
				LastSeenAt:            now,
				AccessTokenExpiresAt:  now.Add(24 * time.Hour),
				RefreshTokenExpiresAt: now.Add(24 * time.Hour),
			},
		}, nil
	}

	now := time.Now().UTC()
	tx, err := a.DB.BeginTx(ctx, nil)
	if err != nil {
		return coreid.SessionTokens{}, err
	}
	defer tx.Rollback()

	tokens, err := a.issueSessionTx(ctx, tx, principal, meta, now)
	if err != nil {
		return coreid.SessionTokens{}, err
	}
	if err := tx.Commit(); err != nil {
		return coreid.SessionTokens{}, err
	}
	a.maybeCleanup()
	return tokens, nil
}

func (a *Adapter) RefreshSession(ctx context.Context, refreshToken string, meta coreid.SessionMetadata) (coreid.SessionTokens, error) {
	if a.DB == nil {
		return coreid.SessionTokens{}, coreid.ErrInvalidRefreshToken
	}
	tokenHash := hashRefreshToken(refreshToken)
	if tokenHash == "" {
		return coreid.SessionTokens{}, coreid.ErrInvalidRefreshToken
	}

	now := time.Now().UTC()
	tx, err := a.DB.BeginTx(ctx, nil)
	if err != nil {
		return coreid.SessionTokens{}, err
	}
	defer tx.Rollback()

	record, err := a.getSessionByRefreshHashTx(ctx, tx, tokenHash)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return coreid.SessionTokens{}, coreid.ErrInvalidRefreshToken
		}
		return coreid.SessionTokens{}, err
	}
	if record.RevokedAt != nil {
		return coreid.SessionTokens{}, coreid.ErrInvalidRefreshToken
	}
	if record.RefreshTokenExpiresAt.Before(now) {
		if err := a.revokeSessionTx(ctx, tx, int(record.UserID), record.ID, "refresh_token_expired", now); err != nil {
			return coreid.SessionTokens{}, err
		}
		if err := tx.Commit(); err != nil {
			return coreid.SessionTokens{}, err
		}
		return coreid.SessionTokens{}, coreid.ErrInvalidRefreshToken
	}
	if record.PreviousRefreshHash == tokenHash {
		if err := a.revokeSessionTx(ctx, tx, int(record.UserID), record.ID, "refresh_token_replay", now); err != nil {
			return coreid.SessionTokens{}, err
		}
		if err := tx.Commit(); err != nil {
			return coreid.SessionTokens{}, err
		}
		return coreid.SessionTokens{}, coreid.ErrRefreshTokenReplay
	}
	if record.CurrentRefreshHash != tokenHash {
		return coreid.SessionTokens{}, coreid.ErrInvalidRefreshToken
	}

	nextRefreshToken, err := newRefreshToken()
	if err != nil {
		return coreid.SessionTokens{}, err
	}
	nextRefreshHash := hashRefreshToken(nextRefreshToken)
	accessExpiresAt := now.Add(a.accessTTL())
	refreshExpiresAt := now.Add(a.refreshTTL())
	deviceLabel := record.DeviceLabel
	if meta.DeviceLabel != "" {
		deviceLabel = normalizeDeviceLabel(meta.DeviceLabel)
	}
	userAgent := record.UserAgent
	if meta.UserAgent != "" {
		userAgent = meta.UserAgent
	}
	lastSeenIP := record.LastSeenIP
	if meta.IPAddress != "" {
		lastSeenIP = meta.IPAddress
	}

	if _, err := tx.ExecContext(ctx, `
		UPDATE auth_sessions
		SET previous_refresh_hash = current_refresh_hash,
			current_refresh_hash = ?,
			device_label = ?,
			user_agent = ?,
			last_seen_ip = ?,
			last_seen_at = ?,
			refreshed_at = ?,
			access_token_expires_at = ?,
			refresh_token_expires_at = ?,
			updated_at = ?
		WHERE id = ? AND user_id = ? AND revoked_at IS NULL
	`, nextRefreshHash, deviceLabel, userAgent, lastSeenIP, now, now, accessExpiresAt, refreshExpiresAt, now, record.ID, record.UserID); err != nil {
		return coreid.SessionTokens{}, err
	}

	accessToken, err := auth.GenerateAccessToken(int(record.UserID), record.ID, a.accessTTL())
	if err != nil {
		return coreid.SessionTokens{}, err
	}
	if err := tx.Commit(); err != nil {
		return coreid.SessionTokens{}, err
	}
	a.maybeCleanup()

	return coreid.SessionTokens{
		AccessToken:           accessToken,
		RefreshToken:          nextRefreshToken,
		AccessTokenExpiresAt:  accessExpiresAt,
		RefreshTokenExpiresAt: refreshExpiresAt,
		Session: coreid.Session{
			ID:                    record.ID,
			UserID:                record.UserID,
			DeviceLabel:           deviceLabel,
			UserAgent:             userAgent,
			LastSeenIP:            lastSeenIP,
			CreatedAt:             record.CreatedAt,
			LastSeenAt:            now,
			AccessTokenExpiresAt:  accessExpiresAt,
			RefreshTokenExpiresAt: refreshExpiresAt,
		},
	}, nil
}

func (a *Adapter) ValidateToken(ctx context.Context, token string) (coreid.TokenClaims, error) {
	claims, err := auth.ValidateToken(token)
	if err != nil {
		return coreid.TokenClaims{}, err
	}
	if a.DB == nil || claims.SessionID == 0 {
		return coreid.TokenClaims{
			SubjectUserID: coreid.UserID(claims.UserID),
			SessionID:     claims.SessionID,
		}, nil
	}

	var userID int
	var revokedAt sql.NullTime
	if err := a.DB.QueryRowContext(ctx, `
		SELECT user_id, revoked_at
		FROM auth_sessions
		WHERE id = ?
	`, claims.SessionID).Scan(&userID, &revokedAt); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return coreid.TokenClaims{}, coreid.ErrSessionNotFound
		}
		return coreid.TokenClaims{}, err
	}
	if revokedAt.Valid || userID != claims.UserID {
		return coreid.TokenClaims{}, coreid.ErrSessionNotFound
	}

	return coreid.TokenClaims{
		SubjectUserID: coreid.UserID(claims.UserID),
		SessionID:     claims.SessionID,
	}, nil
}

func (a *Adapter) ListSessions(ctx context.Context, userID coreid.UserID) ([]coreid.Session, error) {
	if a.DB == nil {
		return nil, nil
	}

	rows, err := a.DB.QueryContext(ctx, `
		SELECT id, user_id, device_label, user_agent, last_seen_ip,
		       created_at, last_seen_at, access_token_expires_at, refresh_token_expires_at, revoked_at
		FROM auth_sessions
		WHERE user_id = ?
		ORDER BY created_at DESC, id DESC
	`, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	sessions := make([]coreid.Session, 0)
	for rows.Next() {
		session, err := scanSession(rows)
		if err != nil {
			return nil, err
		}
		sessions = append(sessions, session)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	a.maybeCleanup()
	return sessions, nil
}

func (a *Adapter) RevokeSession(ctx context.Context, actorUserID coreid.UserID, sessionID int64) error {
	if a.DB == nil {
		return coreid.ErrSessionNotFound
	}
	now := time.Now().UTC()
	tx, err := a.DB.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	if err := a.revokeSessionTx(ctx, tx, int(actorUserID), sessionID, "user_revoked", now); err != nil {
		return err
	}
	if err := tx.Commit(); err != nil {
		return err
	}
	a.maybeCleanup()
	return nil
}

func (a *Adapter) TouchSession(ctx context.Context, sessionID int64, meta coreid.SessionMetadata) error {
	if a.DB == nil || sessionID <= 0 {
		return nil
	}
	now := time.Now().UTC()
	_, err := a.DB.ExecContext(ctx, `
		UPDATE auth_sessions
		SET user_agent = CASE WHEN ? <> '' THEN ? ELSE user_agent END,
			last_seen_ip = CASE WHEN ? <> '' THEN ? ELSE last_seen_ip END,
			last_seen_at = ?,
			updated_at = ?
		WHERE id = ? AND revoked_at IS NULL
	`, meta.UserAgent, meta.UserAgent, meta.IPAddress, meta.IPAddress, now, now, sessionID)
	if err == nil {
		a.maybeCleanup()
	}
	return err
}

type sessionRecord struct {
	coreid.Session
	CurrentRefreshHash  string
	PreviousRefreshHash string
}

func (a *Adapter) issueSessionTx(ctx context.Context, tx *sql.Tx, principal coreid.Principal, meta coreid.SessionMetadata, now time.Time) (coreid.SessionTokens, error) {
	refreshToken, err := newRefreshToken()
	if err != nil {
		return coreid.SessionTokens{}, err
	}
	refreshHash := hashRefreshToken(refreshToken)
	accessExpiresAt := now.Add(a.accessTTL())
	refreshExpiresAt := now.Add(a.refreshTTL())
	deviceLabel := normalizeDeviceLabel(meta.DeviceLabel)

	result, err := tx.ExecContext(ctx, `
		INSERT INTO auth_sessions (
			user_id, device_label, user_agent, last_seen_ip, current_refresh_hash,
			access_token_expires_at, refresh_token_expires_at, created_at, last_seen_at, refreshed_at, updated_at
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`, principal.ID, deviceLabel, meta.UserAgent, meta.IPAddress, refreshHash, accessExpiresAt, refreshExpiresAt, now, now, now, now)
	if err != nil {
		return coreid.SessionTokens{}, err
	}
	sessionID, err := result.LastInsertId()
	if err != nil {
		return coreid.SessionTokens{}, err
	}
	accessToken, err := auth.GenerateAccessToken(int(principal.ID), sessionID, a.accessTTL())
	if err != nil {
		return coreid.SessionTokens{}, err
	}
	return coreid.SessionTokens{
		AccessToken:           accessToken,
		RefreshToken:          refreshToken,
		AccessTokenExpiresAt:  accessExpiresAt,
		RefreshTokenExpiresAt: refreshExpiresAt,
		Session: coreid.Session{
			ID:                    sessionID,
			UserID:                principal.ID,
			DeviceLabel:           deviceLabel,
			UserAgent:             meta.UserAgent,
			LastSeenIP:            meta.IPAddress,
			CreatedAt:             now,
			LastSeenAt:            now,
			AccessTokenExpiresAt:  accessExpiresAt,
			RefreshTokenExpiresAt: refreshExpiresAt,
		},
	}, nil
}

func (a *Adapter) getSessionByRefreshHashTx(ctx context.Context, tx *sql.Tx, refreshHash string) (sessionRecord, error) {
	row := tx.QueryRowContext(ctx, `
		SELECT id, user_id, device_label, user_agent, last_seen_ip,
		       created_at, last_seen_at, access_token_expires_at, refresh_token_expires_at, revoked_at,
		       current_refresh_hash, previous_refresh_hash
		FROM auth_sessions
		WHERE current_refresh_hash = ? OR previous_refresh_hash = ?
		LIMIT 1
	`, refreshHash, refreshHash)

	session, err := scanSessionRecord(row)
	if err != nil {
		return sessionRecord{}, err
	}
	return session, nil
}

func (a *Adapter) revokeSessionTx(ctx context.Context, tx *sql.Tx, actorUserID int, sessionID int64, reason string, now time.Time) error {
	result, err := tx.ExecContext(ctx, `
		UPDATE auth_sessions
		SET revoked_at = COALESCE(revoked_at, ?),
			revoke_reason = CASE WHEN revoke_reason = '' THEN ? ELSE revoke_reason END,
			updated_at = ?
		WHERE id = ? AND user_id = ?
	`, now, reason, now, sessionID, actorUserID)
	if err != nil {
		return err
	}
	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if rowsAffected == 0 {
		return coreid.ErrSessionNotFound
	}
	return nil
}

func (a *Adapter) accessTTL() time.Duration {
	if a.AccessTTL > 0 {
		return a.AccessTTL
	}
	return defaultAccessTTL
}

func (a *Adapter) refreshTTL() time.Duration {
	if a.RefreshTTL > 0 {
		return a.RefreshTTL
	}
	return defaultRefreshTTL
}

func (a *Adapter) maybeCleanup() {
	if a.DB == nil {
		return
	}
	if a.cleanupCalls.Add(1)%256 != 0 {
		return
	}
	cutoff := time.Now().UTC().Add(-30 * 24 * time.Hour)
	_, _ = a.DB.Exec(`
		DELETE FROM auth_sessions
		WHERE (revoked_at IS NOT NULL AND updated_at < ?)
		   OR refresh_token_expires_at < ?
	`, cutoff, cutoff)
}

type scanner interface {
	Scan(dest ...any) error
}

func scanSession(s scanner) (coreid.Session, error) {
	var session coreid.Session
	var revokedAt sql.NullTime
	if err := s.Scan(
		&session.ID,
		&session.UserID,
		&session.DeviceLabel,
		&session.UserAgent,
		&session.LastSeenIP,
		&session.CreatedAt,
		&session.LastSeenAt,
		&session.AccessTokenExpiresAt,
		&session.RefreshTokenExpiresAt,
		&revokedAt,
	); err != nil {
		return coreid.Session{}, err
	}
	if revokedAt.Valid {
		t := revokedAt.Time
		session.RevokedAt = &t
	}
	return session, nil
}

func scanSessionRecord(s scanner) (sessionRecord, error) {
	var record sessionRecord
	var revokedAt sql.NullTime
	if err := s.Scan(
		&record.ID,
		&record.UserID,
		&record.DeviceLabel,
		&record.UserAgent,
		&record.LastSeenIP,
		&record.CreatedAt,
		&record.LastSeenAt,
		&record.AccessTokenExpiresAt,
		&record.RefreshTokenExpiresAt,
		&revokedAt,
		&record.CurrentRefreshHash,
		&record.PreviousRefreshHash,
	); err != nil {
		return sessionRecord{}, err
	}
	if revokedAt.Valid {
		t := revokedAt.Time
		record.RevokedAt = &t
	}
	return record, nil
}

func newRefreshToken() (string, error) {
	var raw [32]byte
	if _, err := rand.Read(raw[:]); err != nil {
		return "", fmt.Errorf("generate refresh token: %w", err)
	}
	return base64.RawURLEncoding.EncodeToString(raw[:]), nil
}

func hashRefreshToken(token string) string {
	if token == "" {
		return ""
	}
	sum := sha256.Sum256([]byte(token))
	return hex.EncodeToString(sum[:])
}

func normalizeDeviceLabel(label string) string {
	if label == "" {
		return "This device"
	}
	return label
}
