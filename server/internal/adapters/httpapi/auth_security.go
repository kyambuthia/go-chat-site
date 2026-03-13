package httpapi

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/kyambuthia/go-chat-site/server/internal/auth"
	"github.com/kyambuthia/go-chat-site/server/internal/config"
)

var errAuthRateLimited = errors.New("rate limit exceeded")

type loginLockedError struct {
	Until time.Time
}

func (e loginLockedError) Error() string {
	return fmt.Sprintf("too many failed login attempts; try again after %s", e.Until.UTC().Format(time.RFC3339))
}

type authSecurity struct {
	db               *sql.DB
	perIPLimiter     requestRateLimiter
	perUserLimiter   requestRateLimiter
	lockoutThreshold int
	lockoutWindow    time.Duration
	lockoutDuration  time.Duration
}

func newAuthSecurity(db *sql.DB) (*authSecurity, error) {
	if db == nil {
		return nil, nil
	}

	perIPLimiter, err := newSharedWindowRateLimiter(db, config.LoginRateLimitPerMinute(), time.Minute)
	if err != nil {
		return nil, err
	}
	perUserLimiter, err := newSharedWindowRateLimiter(db, config.LoginUserRateLimitPerMinute(), time.Minute)
	if err != nil {
		return nil, err
	}
	security := &authSecurity{
		db:               db,
		perIPLimiter:     perIPLimiter,
		perUserLimiter:   perUserLimiter,
		lockoutThreshold: config.LoginLockoutThreshold(),
		lockoutWindow:    config.LoginLockoutWindow(),
		lockoutDuration:  config.LoginLockoutDuration(),
	}
	if err := security.init(context.Background()); err != nil {
		return nil, err
	}
	return security, nil
}

func (s *authSecurity) init(ctx context.Context) error {
	if s == nil || s.db == nil {
		return nil
	}
	_, err := s.db.ExecContext(ctx, `
		CREATE TABLE IF NOT EXISTS auth_login_throttles (
			scope_key TEXT PRIMARY KEY,
			failure_count INTEGER NOT NULL DEFAULT 0,
			first_failed_at DATETIME,
			locked_until DATETIME,
			updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
		)
	`)
	return err
}

func (s *authSecurity) allowLogin(ctx context.Context, username string, ip string, requestID string) error {
	if s == nil {
		return nil
	}
	normalizedUsername := normalizeThrottleUsername(username)
	if s.perIPLimiter != nil && !s.perIPLimiter.allow("auth-login:ip:"+ip) {
		auth.LogSecurityEvent("auth_rate_limit_exceeded", map[string]any{
			"request_id": requestID,
			"scope":      "ip",
			"ip_address": ip,
		})
		return errAuthRateLimited
	}
	if normalizedUsername != "" && s.perUserLimiter != nil && !s.perUserLimiter.allow("auth-login:user:"+normalizedUsername) {
		auth.LogSecurityEvent("auth_rate_limit_exceeded", map[string]any{
			"request_id": requestID,
			"scope":      "user",
			"username":   normalizedUsername,
			"ip_address": ip,
		})
		return errAuthRateLimited
	}

	lockedUntil, err := s.lockedUntil(ctx, "user:"+normalizedUsername)
	if err != nil {
		return err
	}
	if !lockedUntil.IsZero() && lockedUntil.After(time.Now().UTC()) {
		auth.LogSecurityEvent("auth_login_locked", map[string]any{
			"request_id":   requestID,
			"username":     normalizedUsername,
			"ip_address":   ip,
			"locked_until": lockedUntil.UTC().Format(time.RFC3339),
		})
		return loginLockedError{Until: lockedUntil}
	}
	return nil
}

func (s *authSecurity) recordLoginFailure(ctx context.Context, username string, ip string, requestID string) {
	if s == nil || s.db == nil {
		return
	}
	normalizedUsername := normalizeThrottleUsername(username)
	if normalizedUsername == "" {
		return
	}
	now := time.Now().UTC()
	key := "user:" + normalizedUsername
	count, firstFailedAt, _, err := s.readThrottle(ctx, key)
	if err != nil {
		return
	}
	if firstFailedAt.IsZero() || now.Sub(firstFailedAt) > s.lockoutWindow {
		count = 0
		firstFailedAt = now
	}
	count++
	var lockedUntil *time.Time
	if count >= s.lockoutThreshold {
		until := now.Add(s.lockoutDuration)
		lockedUntil = &until
	}
	_, _ = s.db.ExecContext(ctx, `
		INSERT INTO auth_login_throttles (scope_key, failure_count, first_failed_at, locked_until, updated_at)
		VALUES (?, ?, ?, ?, ?)
		ON CONFLICT(scope_key) DO UPDATE SET
			failure_count = excluded.failure_count,
			first_failed_at = excluded.first_failed_at,
			locked_until = excluded.locked_until,
			updated_at = excluded.updated_at
	`, key, count, firstFailedAt, lockedUntil, now)

	fields := map[string]any{
		"request_id":    requestID,
		"username":      normalizedUsername,
		"ip_address":    ip,
		"failure_count": count,
	}
	if lockedUntil != nil {
		fields["locked_until"] = lockedUntil.UTC().Format(time.RFC3339)
	}
	auth.LogSecurityEvent("auth_login_failed", fields)
	s.maybeCleanup()
}

func (s *authSecurity) recordLoginSuccess(ctx context.Context, userID int, username string, ip string, requestID string) {
	if s == nil || s.db == nil {
		return
	}
	normalizedUsername := normalizeThrottleUsername(username)
	if normalizedUsername != "" {
		_, _ = s.db.ExecContext(ctx, `DELETE FROM auth_login_throttles WHERE scope_key = ?`, "user:"+normalizedUsername)
	}
	auth.LogSecurityEvent("auth_login_succeeded", map[string]any{
		"request_id": requestID,
		"user_id":    userID,
		"username":   normalizedUsername,
		"ip_address": ip,
	})
	s.maybeCleanup()
}

func (s *authSecurity) readThrottle(ctx context.Context, key string) (int, time.Time, time.Time, error) {
	if s == nil || s.db == nil {
		return 0, time.Time{}, time.Time{}, nil
	}
	var count int
	var firstFailedAt sql.NullTime
	var lockedUntil sql.NullTime
	err := s.db.QueryRowContext(ctx, `
		SELECT failure_count, first_failed_at, locked_until
		FROM auth_login_throttles
		WHERE scope_key = ?
	`, key).Scan(&count, &firstFailedAt, &lockedUntil)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return 0, time.Time{}, time.Time{}, nil
		}
		return 0, time.Time{}, time.Time{}, err
	}
	var first time.Time
	if firstFailedAt.Valid {
		first = firstFailedAt.Time
	}
	var locked time.Time
	if lockedUntil.Valid {
		locked = lockedUntil.Time
	}
	return count, first, locked, nil
}

func (s *authSecurity) lockedUntil(ctx context.Context, key string) (time.Time, error) {
	_, _, lockedUntil, err := s.readThrottle(ctx, key)
	return lockedUntil, err
}

func (s *authSecurity) maybeCleanup() {
	if s == nil || s.db == nil {
		return
	}
	cutoff := time.Now().UTC().Add(-24 * time.Hour)
	_, _ = s.db.Exec(`DELETE FROM auth_login_throttles WHERE updated_at < ?`, cutoff)
}

func normalizeThrottleUsername(username string) string {
	return strings.ToLower(strings.TrimSpace(username))
}
