package httpapi

import (
	"context"
	"database/sql"
	"encoding/json"
	"net"
	"net/http"
	"strings"
	"sync"
	"sync/atomic"
	"time"
)

type requestRateLimiter interface {
	allow(key string) bool
}

type fixedWindowRateLimiter struct {
	limit  int
	window time.Duration

	mu    sync.Mutex
	hits  map[string]int
	until time.Time
}

func newFixedWindowRateLimiter(limit int, window time.Duration) *fixedWindowRateLimiter {
	return &fixedWindowRateLimiter{
		limit:  limit,
		window: window,
		hits:   make(map[string]int),
		until:  time.Now().Add(window),
	}
}

func (l *fixedWindowRateLimiter) allow(key string) bool {
	if l.limit <= 0 {
		return true
	}
	now := time.Now()

	l.mu.Lock()
	defer l.mu.Unlock()

	if now.After(l.until) {
		l.hits = make(map[string]int)
		l.until = now.Add(l.window)
	}

	l.hits[key]++
	return l.hits[key] <= l.limit
}

type sharedWindowRateLimiter struct {
	db     *sql.DB
	limit  int
	window time.Duration

	calls atomic.Uint64
}

func newSharedWindowRateLimiter(db *sql.DB, limit int, window time.Duration) (*sharedWindowRateLimiter, error) {
	l := &sharedWindowRateLimiter{
		db:     db,
		limit:  limit,
		window: window,
	}
	if err := l.init(context.Background()); err != nil {
		return nil, err
	}
	return l, nil
}

func (l *sharedWindowRateLimiter) init(ctx context.Context) error {
	_, err := l.db.ExecContext(ctx, `
		CREATE TABLE IF NOT EXISTS rate_limit_windows (
			rate_key TEXT NOT NULL,
			window_start_unix INTEGER NOT NULL,
			hits INTEGER NOT NULL DEFAULT 0,
			updated_at_unix INTEGER NOT NULL,
			PRIMARY KEY(rate_key, window_start_unix)
		)
	`)
	return err
}

func (l *sharedWindowRateLimiter) allow(key string) bool {
	if l.limit <= 0 {
		return true
	}
	now := time.Now().Unix()
	windowSeconds := int64(l.window / time.Second)
	if windowSeconds <= 0 {
		windowSeconds = 1
	}
	windowStart := now - (now % windowSeconds)

	tx, err := l.db.BeginTx(context.Background(), nil)
	if err != nil {
		return true
	}
	defer tx.Rollback()

	if _, err := tx.Exec(
		`INSERT OR IGNORE INTO rate_limit_windows (rate_key, window_start_unix, hits, updated_at_unix) VALUES (?, ?, 0, ?)`,
		key,
		windowStart,
		now,
	); err != nil {
		return true
	}

	if _, err := tx.Exec(
		`UPDATE rate_limit_windows SET hits = hits + 1, updated_at_unix = ? WHERE rate_key = ? AND window_start_unix = ?`,
		now,
		key,
		windowStart,
	); err != nil {
		return true
	}

	var hits int
	if err := tx.QueryRow(
		`SELECT hits FROM rate_limit_windows WHERE rate_key = ? AND window_start_unix = ?`,
		key,
		windowStart,
	).Scan(&hits); err != nil {
		return true
	}

	if err := tx.Commit(); err != nil {
		return true
	}

	// Probabilistic cleanup to bound growth while keeping hot path cheap.
	if l.calls.Add(1)%256 == 0 {
		cutoff := windowStart - (windowSeconds * 2)
		_, _ = l.db.Exec(`DELETE FROM rate_limit_windows WHERE window_start_unix < ?`, cutoff)
	}
	return hits <= l.limit
}

func rateLimitMiddleware(limiter requestRateLimiter) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if limiter == nil || limiter.allow(clientIP(r)) {
				next.ServeHTTP(w, r)
				return
			}

			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusTooManyRequests)
			_ = json.NewEncoder(w).Encode(map[string]string{"error": "rate limit exceeded"})
		})
	}
}

func clientIP(r *http.Request) string {
	if xff := strings.TrimSpace(r.Header.Get("X-Forwarded-For")); xff != "" {
		parts := strings.Split(xff, ",")
		if len(parts) > 0 {
			if v := strings.TrimSpace(parts[0]); v != "" {
				return v
			}
		}
	}

	host, _, err := net.SplitHostPort(strings.TrimSpace(r.RemoteAddr))
	if err == nil && host != "" {
		return host
	}
	if ra := strings.TrimSpace(r.RemoteAddr); ra != "" {
		return ra
	}
	return "unknown"
}
