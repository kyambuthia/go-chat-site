package identity

import (
	"context"
	"errors"
	"testing"
)

type fakeAuthRepo struct {
	createID    UserID
	createErr   error
	loginRecord PasswordLoginRecord
	loginErr    error
	lastCreate  PasswordCredential
	lastLookup  string
}

func (f *fakeAuthRepo) CreateUser(ctx context.Context, cred PasswordCredential) (Principal, error) {
	_ = ctx
	f.lastCreate = cred
	if f.createErr != nil {
		return Principal{}, f.createErr
	}
	return Principal{ID: f.createID, Username: cred.Username}, nil
}

func (f *fakeAuthRepo) GetPasswordLoginRecord(ctx context.Context, username string) (PasswordLoginRecord, error) {
	_ = ctx
	f.lastLookup = username
	if f.loginErr != nil {
		return PasswordLoginRecord{}, f.loginErr
	}
	return f.loginRecord, nil
}

type fakePasswordVerifier struct {
	ok           bool
	lastPassword string
	lastHash     string
}

func (f *fakePasswordVerifier) VerifyPassword(password, hash string) bool {
	f.lastPassword = password
	f.lastHash = hash
	return f.ok
}

type fakeTokenService struct {
	token         string
	err           error
	lastPrincipal Principal
}

func (f *fakeTokenService) IssueToken(ctx context.Context, principal Principal) (string, error) {
	_ = ctx
	f.lastPrincipal = principal
	if f.err != nil {
		return "", f.err
	}
	return f.token, nil
}

func (f *fakeTokenService) ValidateToken(ctx context.Context, token string) (TokenClaims, error) {
	_ = ctx
	_ = token
	return TokenClaims{}, nil
}

func TestAuthService_RegisterPassword_DelegatesToRepository(t *testing.T) {
	repo := &fakeAuthRepo{createID: 9}
	svc := NewAuthService(repo, &fakePasswordVerifier{}, &fakeTokenService{})

	principal, err := svc.RegisterPassword(context.Background(), PasswordCredential{Username: "alice", Password: "password123"})
	if err != nil {
		t.Fatalf("RegisterPassword error: %v", err)
	}
	if repo.lastCreate.Username != "alice" {
		t.Fatalf("CreateUser username = %q, want alice", repo.lastCreate.Username)
	}
	if principal.ID != 9 || principal.Username != "alice" {
		t.Fatalf("unexpected principal: %+v", principal)
	}
}

func TestAuthService_LoginPassword_IssuesTokenForValidCredentials(t *testing.T) {
	repo := &fakeAuthRepo{
		loginRecord: PasswordLoginRecord{
			Principal:    Principal{ID: 7, Username: "alice"},
			PasswordHash: "hash",
		},
	}
	verifier := &fakePasswordVerifier{ok: true}
	tokens := &fakeTokenService{token: "jwt-token"}
	svc := NewAuthService(repo, verifier, tokens)

	token, err := svc.LoginPassword(context.Background(), PasswordCredential{Username: "alice", Password: "password123"})
	if err != nil {
		t.Fatalf("LoginPassword error: %v", err)
	}
	if repo.lastLookup != "alice" {
		t.Fatalf("GetPasswordLoginRecord lookup = %q, want alice", repo.lastLookup)
	}
	if verifier.lastPassword != "password123" || verifier.lastHash != "hash" {
		t.Fatalf("unexpected verifier inputs password=%q hash=%q", verifier.lastPassword, verifier.lastHash)
	}
	if tokens.lastPrincipal.ID != 7 || tokens.lastPrincipal.Username != "alice" {
		t.Fatalf("unexpected principal passed to token service: %+v", tokens.lastPrincipal)
	}
	if token != "jwt-token" {
		t.Fatalf("token = %q, want jwt-token", token)
	}
}

func TestAuthService_LoginPassword_ReturnsInvalidCredentials(t *testing.T) {
	t.Run("user lookup fails", func(t *testing.T) {
		repo := &fakeAuthRepo{loginErr: errors.New("not found")}
		svc := NewAuthService(repo, &fakePasswordVerifier{}, &fakeTokenService{})

		_, err := svc.LoginPassword(context.Background(), PasswordCredential{Username: "alice", Password: "pw"})
		if !errors.Is(err, ErrInvalidCredentials) {
			t.Fatalf("expected ErrInvalidCredentials, got %v", err)
		}
	})

	t.Run("password mismatch", func(t *testing.T) {
		repo := &fakeAuthRepo{
			loginRecord: PasswordLoginRecord{Principal: Principal{ID: 7, Username: "alice"}, PasswordHash: "hash"},
		}
		verifier := &fakePasswordVerifier{ok: false}
		svc := NewAuthService(repo, verifier, &fakeTokenService{})

		_, err := svc.LoginPassword(context.Background(), PasswordCredential{Username: "alice", Password: "wrong"})
		if !errors.Is(err, ErrInvalidCredentials) {
			t.Fatalf("expected ErrInvalidCredentials, got %v", err)
		}
	})
}
