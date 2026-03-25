package httpapi

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	coreid "github.com/kyambuthia/go-chat-site/server/internal/core/identity"
)

type fakeAuthService struct {
	loginResp  coreid.SessionTokens
	loginErr   error
	loginCalls int
	lastCred   coreid.PasswordCredential
	lastMeta   coreid.SessionMetadata
}

func (f *fakeAuthService) RegisterPassword(ctx context.Context, cred coreid.PasswordCredential) (coreid.Principal, error) {
	_ = ctx
	_ = cred
	return coreid.Principal{}, nil
}

func (f *fakeAuthService) LoginPassword(ctx context.Context, cred coreid.PasswordCredential, meta coreid.SessionMetadata) (coreid.SessionTokens, error) {
	_ = ctx
	f.loginCalls++
	f.lastCred = cred
	f.lastMeta = meta
	return f.loginResp, f.loginErr
}

func loginReq(body string) *http.Request {
	req := httptest.NewRequest(http.MethodPost, "/api/login", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Request-ID", "req-1")
	req.RemoteAddr = "127.0.0.1:3456"
	req.Header.Set("User-Agent", "auth-handler-test")
	return req
}

func TestAuthHandler_Login_RateLimitedIncludesRetryAfter(t *testing.T) {
	security := newAuthSecurityForTest(t, 100, 1, 10)
	identity := &fakeAuthService{loginResp: coreid.SessionTokens{
		AccessToken:           "access-token",
		RefreshToken:          "refresh-token",
		AccessTokenExpiresAt:  time.Now().UTC().Add(15 * time.Minute),
		RefreshTokenExpiresAt: time.Now().UTC().Add(24 * time.Hour),
		Session: coreid.Session{
			ID:                    7,
			UserID:                4,
			DeviceLabel:           "Browser",
			UserAgent:             "auth-handler-test",
			LastSeenIP:            "127.0.0.1",
			CreatedAt:             time.Now().UTC(),
			LastSeenAt:            time.Now().UTC(),
			AccessTokenExpiresAt:  time.Now().UTC().Add(15 * time.Minute),
			RefreshTokenExpiresAt: time.Now().UTC().Add(24 * time.Hour),
		},
	}}
	h := &AuthHandler{Identity: identity, Security: security}

	first := httptest.NewRecorder()
	h.Login(first, loginReq(`{"username":"alice","password":"password123"}`))
	if first.Code != http.StatusOK {
		t.Fatalf("first status = %d, want 200 body=%s", first.Code, first.Body.String())
	}

	second := httptest.NewRecorder()
	h.Login(second, loginReq(`{"username":"alice","password":"password123"}`))

	if second.Code != http.StatusTooManyRequests {
		t.Fatalf("second status = %d, want 429 body=%s", second.Code, second.Body.String())
	}
	if got := second.Header().Get("Retry-After"); got == "" {
		t.Fatal("expected Retry-After header")
	}
	var resp map[string]any
	if err := json.Unmarshal(second.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal response: %v", err)
	}
	if got := resp["error"].(string); got != "rate limit exceeded" {
		t.Fatalf("error = %q, want rate limit exceeded", got)
	}
	if got := resp["scope"].(string); got != "user" {
		t.Fatalf("scope = %q, want user", got)
	}
	if got := int(resp["retry_after_seconds"].(float64)); got < 1 {
		t.Fatalf("retry_after_seconds = %d, want >= 1", got)
	}
	if identity.loginCalls != 1 {
		t.Fatalf("identity login calls = %d, want 1", identity.loginCalls)
	}
}

func TestAuthHandler_Login_LockoutIncludesRetryAfter(t *testing.T) {
	security := newAuthSecurityForTest(t, 100, 100, 2)
	security.recordLoginFailure(context.Background(), "alice", "127.0.0.1", "req-1")
	security.recordLoginFailure(context.Background(), "alice", "127.0.0.1", "req-2")

	identity := &fakeAuthService{}
	h := &AuthHandler{Identity: identity, Security: security}

	rr := httptest.NewRecorder()
	h.Login(rr, loginReq(`{"username":"alice","password":"password123"}`))

	if rr.Code != http.StatusTooManyRequests {
		t.Fatalf("status = %d, want 429 body=%s", rr.Code, rr.Body.String())
	}
	if got := rr.Header().Get("Retry-After"); got == "" {
		t.Fatal("expected Retry-After header")
	}
	var resp map[string]any
	if err := json.Unmarshal(rr.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal response: %v", err)
	}
	if _, ok := resp["locked_until"].(string); !ok {
		t.Fatalf("locked_until missing from response: %v", resp)
	}
	if got := int(resp["retry_after_seconds"].(float64)); got < 1 {
		t.Fatalf("retry_after_seconds = %d, want >= 1", got)
	}
	if identity.loginCalls != 0 {
		t.Fatalf("identity login calls = %d, want 0", identity.loginCalls)
	}
}
