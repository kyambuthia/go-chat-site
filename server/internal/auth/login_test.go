package auth

import (
	"bytes"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/kyambuthia/go-chat-site/server/internal/crypto"
	"github.com/kyambuthia/go-chat-site/server/internal/store"
)

type fakeLoginStore struct {
	user *store.User
	err  error
}

func (f *fakeLoginStore) GetUserByUsername(username string) (*store.User, error) {
	if f.err != nil {
		return nil, f.err
	}
	if f.user == nil || f.user.Username != username {
		return nil, store.ErrNotFound
	}
	return f.user, nil
}

func TestLogin_MethodNotAllowed(t *testing.T) {
	h := Login(&fakeLoginStore{})
	req := httptest.NewRequest(http.MethodGet, "/api/login", nil)
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)
	if rr.Code != http.StatusMethodNotAllowed {
		t.Fatalf("status = %d, want 405", rr.Code)
	}
}

func TestLogin_InvalidBody(t *testing.T) {
	h := Login(&fakeLoginStore{})
	req := httptest.NewRequest(http.MethodPost, "/api/login", bytes.NewBufferString("{"))
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)
	if rr.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400", rr.Code)
	}
}

func TestLogin_InvalidCredentials(t *testing.T) {
	h := Login(&fakeLoginStore{err: errors.New("db down")})
	body, _ := json.Marshal(map[string]string{"username": "alice", "password": "password123"})
	req := httptest.NewRequest(http.MethodPost, "/api/login", bytes.NewReader(body))
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)
	if rr.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, want 401", rr.Code)
	}
}

func TestLogin_Succeeds(t *testing.T) {
	if err := ConfigureJWT("login-test-secret"); err != nil {
		t.Fatal(err)
	}
	hash, err := crypto.HashPassword("password123")
	if err != nil {
		t.Fatal(err)
	}

	h := Login(&fakeLoginStore{user: &store.User{ID: 7, Username: "alice", PasswordHash: hash}})
	body, _ := json.Marshal(map[string]string{"username": "alice", "password": "password123"})
	req := httptest.NewRequest(http.MethodPost, "/api/login", bytes.NewReader(body))
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200 body=%s", rr.Code, rr.Body.String())
	}
	var resp map[string]string
	if err := json.Unmarshal(rr.Body.Bytes(), &resp); err != nil {
		t.Fatal(err)
	}
	if resp["token"] == "" {
		t.Fatal("expected token")
	}
}
