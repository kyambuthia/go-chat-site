package jwttokens

import (
	"context"
	"path/filepath"
	"testing"

	"github.com/kyambuthia/go-chat-site/server/internal/auth"
	coreid "github.com/kyambuthia/go-chat-site/server/internal/core/identity"
	"github.com/kyambuthia/go-chat-site/server/internal/migrate"
	"github.com/kyambuthia/go-chat-site/server/internal/store"
)

func newTokenTestStore(t *testing.T) *store.SqliteStore {
	t.Helper()
	s, err := store.NewSqliteStore(":memory:")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = s.DB.Close() })
	if err := migrate.RunMigrations(s.DB, filepath.Join("..", "..", "..", "..", "migrations")); err != nil {
		t.Fatal(err)
	}
	return s
}

func TestAdapter_IssueValidateListAndRevokeSession(t *testing.T) {
	if err := auth.ConfigureJWT("test-secret-123456"); err != nil {
		t.Fatal(err)
	}
	s := newTokenTestStore(t)
	userID, err := s.CreateUser("alice", "password123")
	if err != nil {
		t.Fatal(err)
	}
	a := &Adapter{DB: s.DB}

	tokens, err := a.IssueSession(context.Background(), coreid.Principal{ID: coreid.UserID(userID), Username: "alice"}, coreid.SessionMetadata{
		DeviceLabel: "MacBook",
		UserAgent:   "test-agent",
		IPAddress:   "127.0.0.1",
	})
	if err != nil {
		t.Fatalf("IssueSession error: %v", err)
	}
	if tokens.AccessToken == "" || tokens.RefreshToken == "" || tokens.Session.ID == 0 {
		t.Fatalf("unexpected session tokens: %+v", tokens)
	}

	claims, err := a.ValidateToken(context.Background(), tokens.AccessToken)
	if err != nil {
		t.Fatalf("ValidateToken error: %v", err)
	}
	if claims.SubjectUserID != coreid.UserID(userID) || claims.SessionID != tokens.Session.ID {
		t.Fatalf("unexpected claims: %+v", claims)
	}

	sessions, err := a.ListSessions(context.Background(), coreid.UserID(userID))
	if err != nil {
		t.Fatalf("ListSessions error: %v", err)
	}
	if len(sessions) != 1 {
		t.Fatalf("sessions len = %d, want 1", len(sessions))
	}
	if sessions[0].DeviceLabel != "MacBook" || sessions[0].LastSeenIP != "127.0.0.1" {
		t.Fatalf("unexpected session: %+v", sessions[0])
	}

	if err := a.RevokeSession(context.Background(), coreid.UserID(userID), tokens.Session.ID); err != nil {
		t.Fatalf("RevokeSession error: %v", err)
	}
	if _, err := a.ValidateToken(context.Background(), tokens.AccessToken); err == nil {
		t.Fatal("expected ValidateToken to fail after revoke")
	}
}

func TestAdapter_RefreshSession_RotatesAndRejectsReplay(t *testing.T) {
	if err := auth.ConfigureJWT("test-secret-123456"); err != nil {
		t.Fatal(err)
	}
	s := newTokenTestStore(t)
	userID, err := s.CreateUser("alice", "password123")
	if err != nil {
		t.Fatal(err)
	}
	a := &Adapter{DB: s.DB}

	initial, err := a.IssueSession(context.Background(), coreid.Principal{ID: coreid.UserID(userID), Username: "alice"}, coreid.SessionMetadata{
		DeviceLabel: "This browser",
		UserAgent:   "agent-1",
		IPAddress:   "127.0.0.1",
	})
	if err != nil {
		t.Fatalf("IssueSession error: %v", err)
	}

	rotated, err := a.RefreshSession(context.Background(), initial.RefreshToken, coreid.SessionMetadata{
		DeviceLabel: "This browser",
		UserAgent:   "agent-2",
		IPAddress:   "127.0.0.2",
	})
	if err != nil {
		t.Fatalf("RefreshSession error: %v", err)
	}
	if rotated.RefreshToken == initial.RefreshToken || rotated.AccessToken == initial.AccessToken {
		t.Fatalf("expected token rotation, got initial=%+v rotated=%+v", initial, rotated)
	}

	if _, err := a.RefreshSession(context.Background(), initial.RefreshToken, coreid.SessionMetadata{
		UserAgent: "replay-agent",
		IPAddress: "127.0.0.3",
	}); err != coreid.ErrRefreshTokenReplay {
		t.Fatalf("replay error = %v, want %v", err, coreid.ErrRefreshTokenReplay)
	}

	if _, err := a.ValidateToken(context.Background(), rotated.AccessToken); err == nil {
		t.Fatal("expected rotated access token to fail after replay-driven revoke")
	}
}

func TestAdapter_ValidateToken_FailsForInvalidToken(t *testing.T) {
	if err := auth.ConfigureJWT("test-secret-123456"); err != nil {
		t.Fatal(err)
	}

	a := &Adapter{}
	if _, err := a.ValidateToken(context.Background(), "bad-token"); err == nil {
		t.Fatal("expected invalid token error")
	}
}
