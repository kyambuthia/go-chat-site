package httpapi

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/kyambuthia/go-chat-site/server/internal/store"
)

func newAuthSecurityForTest(t *testing.T, perIPLimit int, perUserLimit int, threshold int) *authSecurity {
	t.Helper()

	s, err := store.NewSqliteStore(":memory:")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = s.DB.Close() })

	security := &authSecurity{
		db:               s.DB,
		perIPLimiter:     newFixedWindowRateLimiter(perIPLimit, time.Minute),
		perUserLimiter:   newFixedWindowRateLimiter(perUserLimit, time.Minute),
		lockoutThreshold: threshold,
		lockoutWindow:    time.Hour,
		lockoutDuration:  time.Hour,
	}
	if err := security.init(context.Background()); err != nil {
		t.Fatal(err)
	}
	return security
}

func TestAuthSecurity_LocksOutAfterThreshold(t *testing.T) {
	security := newAuthSecurityForTest(t, 100, 100, 3)
	ctx := context.Background()

	for range 3 {
		security.recordLoginFailure(ctx, "alice", "127.0.0.1", "req-1")
	}

	err := security.allowLogin(ctx, "alice", "127.0.0.1", "req-2")
	var locked loginLockedError
	if !errors.As(err, &locked) {
		t.Fatalf("allowLogin error = %v, want loginLockedError", err)
	}
	if locked.Until.IsZero() {
		t.Fatal("expected non-zero lockout deadline")
	}
}

func TestAuthSecurity_RecordLoginSuccessClearsLockout(t *testing.T) {
	security := newAuthSecurityForTest(t, 100, 100, 2)
	ctx := context.Background()

	for range 2 {
		security.recordLoginFailure(ctx, "alice", "127.0.0.1", "req-1")
	}
	if err := security.allowLogin(ctx, "alice", "127.0.0.1", "req-2"); err == nil {
		t.Fatal("expected lockout after repeated failures")
	}

	security.recordLoginSuccess(ctx, 7, "alice", "127.0.0.1", "req-3")

	if err := security.allowLogin(ctx, "alice", "127.0.0.1", "req-4"); err != nil {
		t.Fatalf("allowLogin after success error = %v, want nil", err)
	}
}

func TestAuthSecurity_EnforcesPerUserRateLimit(t *testing.T) {
	security := newAuthSecurityForTest(t, 100, 1, 10)
	ctx := context.Background()

	if err := security.allowLogin(ctx, "alice", "127.0.0.1", "req-1"); err != nil {
		t.Fatalf("first allowLogin error = %v", err)
	}
	err := security.allowLogin(ctx, "alice", "127.0.0.1", "req-2")
	if !errors.Is(err, errAuthRateLimited) {
		t.Fatalf("second allowLogin error = %v, want %v", err, errAuthRateLimited)
	}
	var limited rateLimitedError
	if !errors.As(err, &limited) {
		t.Fatalf("second allowLogin error = %v, want rateLimitedError", err)
	}
	if limited.Scope != "user" {
		t.Fatalf("rate limit scope = %q, want user", limited.Scope)
	}
	if limited.RetryAfter <= 0 {
		t.Fatalf("retry_after = %s, want > 0", limited.RetryAfter)
	}
}

func TestAuthSecurity_EnforcesPerIPRateLimit(t *testing.T) {
	security := newAuthSecurityForTest(t, 1, 100, 10)
	ctx := context.Background()

	if err := security.allowLogin(ctx, "alice", "127.0.0.1", "req-1"); err != nil {
		t.Fatalf("first allowLogin error = %v", err)
	}
	err := security.allowLogin(ctx, "bob", "127.0.0.1", "req-2")
	if !errors.Is(err, errAuthRateLimited) {
		t.Fatalf("second allowLogin error = %v, want %v", err, errAuthRateLimited)
	}
	var limited rateLimitedError
	if !errors.As(err, &limited) {
		t.Fatalf("second allowLogin error = %v, want rateLimitedError", err)
	}
	if limited.Scope != "ip" {
		t.Fatalf("rate limit scope = %q, want ip", limited.Scope)
	}
	if limited.RetryAfter <= 0 {
		t.Fatalf("retry_after = %s, want > 0", limited.RetryAfter)
	}
}
