package httpapi

import (
	"encoding/json"
	"net"
	"net/http"
	"strings"
	"sync"
	"time"
)

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

func rateLimitMiddleware(limiter *fixedWindowRateLimiter) func(http.Handler) http.Handler {
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
