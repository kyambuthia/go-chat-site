package auth

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	coreid "github.com/kyambuthia/go-chat-site/server/internal/core/identity"
)

type fakeAccessTokenService struct {
	claims        coreid.TokenClaims
	validateErr   error
	lastToken     string
	lastTouchID   int64
	lastTouchMeta coreid.SessionMetadata
}

func (f *fakeAccessTokenService) ValidateToken(ctx context.Context, token string) (coreid.TokenClaims, error) {
	_ = ctx
	f.lastToken = token
	if f.validateErr != nil {
		return coreid.TokenClaims{}, f.validateErr
	}
	return f.claims, nil
}

func (f *fakeAccessTokenService) TouchSession(ctx context.Context, sessionID int64, meta coreid.SessionMetadata) error {
	_ = ctx
	f.lastTouchID = sessionID
	f.lastTouchMeta = meta
	return nil
}

func TestMiddleware_RequiresAuthorizationHeader(t *testing.T) {
	h := Middleware(&fakeAccessTokenService{})(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Fatal("next handler should not be called")
	}))

	req := httptest.NewRequest(http.MethodGet, "/api/me", nil)
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)

	if rr.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, want 401", rr.Code)
	}
}

func TestMiddleware_RejectsMalformedBearerHeader(t *testing.T) {
	h := Middleware(&fakeAccessTokenService{})(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Fatal("next handler should not be called")
	}))

	req := httptest.NewRequest(http.MethodGet, "/api/me", nil)
	req.Header.Set("Authorization", "invalid")
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)

	if rr.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, want 401", rr.Code)
	}
}

func TestMiddleware_RejectsInvalidToken(t *testing.T) {
	h := Middleware(&fakeAccessTokenService{validateErr: errors.New("invalid token")})(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Fatal("next handler should not be called")
	}))

	req := httptest.NewRequest(http.MethodGet, "/api/me", nil)
	req.Header.Set("Authorization", "Bearer invalid-token")
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)

	if rr.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, want 401", rr.Code)
	}
}

func TestMiddleware_SetsUserIDOnValidToken(t *testing.T) {
	svc := &fakeAccessTokenService{claims: coreid.TokenClaims{SubjectUserID: 99, SessionID: 444}}

	h := Middleware(svc)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		userID, ok := UserIDFromContext(r.Context())
		if !ok {
			t.Fatal("expected user id in context")
		}
		if userID != 99 {
			t.Fatalf("userID = %d, want 99", userID)
		}
		sessionID, ok := SessionIDFromContext(r.Context())
		if !ok || sessionID != 444 {
			t.Fatalf("sessionID = %d, want 444", sessionID)
		}
		w.WriteHeader(http.StatusNoContent)
	}))

	req := httptest.NewRequest(http.MethodGet, "/api/me", nil)
	req.Header.Set("Authorization", "Bearer valid-token")
	req.RemoteAddr = "127.0.0.1:1234"
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)

	if rr.Code != http.StatusNoContent {
		t.Fatalf("status = %d, want 204", rr.Code)
	}
	if svc.lastTouchID != 444 {
		t.Fatalf("lastTouchID = %d, want 444", svc.lastTouchID)
	}
	if svc.lastTouchMeta.IPAddress != "127.0.0.1" {
		t.Fatalf("touch ip = %q, want 127.0.0.1", svc.lastTouchMeta.IPAddress)
	}
}

func TestMiddleware_PropagatesNextErrorPath(t *testing.T) {
	svc := &fakeAccessTokenService{claims: coreid.TokenClaims{SubjectUserID: 100}}

	nextCalled := false
	h := Middleware(svc)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		nextCalled = true
		http.Error(w, errors.New("boom").Error(), http.StatusTeapot)
	}))

	req := httptest.NewRequest(http.MethodGet, "/api/me", nil)
	req.Header.Set("Authorization", "Bearer valid-token")
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)

	if !nextCalled {
		t.Fatal("expected next handler call")
	}
	if rr.Code != http.StatusTeapot {
		t.Fatalf("status = %d, want 418", rr.Code)
	}
}
