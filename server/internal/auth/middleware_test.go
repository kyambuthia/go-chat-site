package auth

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestMiddleware_RequiresAuthorizationHeader(t *testing.T) {
	h := Middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
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
	h := Middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
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
	if err := ConfigureJWT("middleware-test-secret"); err != nil {
		t.Fatal(err)
	}

	h := Middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
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
	if err := ConfigureJWT("middleware-test-secret-valid"); err != nil {
		t.Fatal(err)
	}
	jwtToken, err := GenerateToken(99)
	if err != nil {
		t.Fatal(err)
	}

	h := Middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		userID, ok := UserIDFromContext(r.Context())
		if !ok {
			t.Fatal("expected user id in context")
		}
		if userID != 99 {
			t.Fatalf("userID = %d, want 99", userID)
		}
		w.WriteHeader(http.StatusNoContent)
	}))

	req := httptest.NewRequest(http.MethodGet, "/api/me", nil)
	req.Header.Set("Authorization", "Bearer "+jwtToken)
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)

	if rr.Code != http.StatusNoContent {
		t.Fatalf("status = %d, want 204", rr.Code)
	}
}

func TestMiddleware_PropagatesNextErrorPath(t *testing.T) {
	if err := ConfigureJWT("middleware-test-secret-pass"); err != nil {
		t.Fatal(err)
	}
	jwtToken, err := GenerateToken(100)
	if err != nil {
		t.Fatal(err)
	}

	nextCalled := false
	h := Middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		nextCalled = true
		http.Error(w, errors.New("boom").Error(), http.StatusTeapot)
	}))

	req := httptest.NewRequest(http.MethodGet, "/api/me", nil)
	req.Header.Set("Authorization", "Bearer "+jwtToken)
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)

	if !nextCalled {
		t.Fatal("expected next handler call")
	}
	if rr.Code != http.StatusTeapot {
		t.Fatalf("status = %d, want 418", rr.Code)
	}
}
