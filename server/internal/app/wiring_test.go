package app

import (
	"context"
	"path/filepath"
	"testing"

	"github.com/kyambuthia/go-chat-site/server/internal/auth"
	corecontacts "github.com/kyambuthia/go-chat-site/server/internal/core/contacts"
	coreid "github.com/kyambuthia/go-chat-site/server/internal/core/identity"
	"github.com/kyambuthia/go-chat-site/server/internal/migrate"
	"github.com/kyambuthia/go-chat-site/server/internal/store"
)

func newTestStore(t *testing.T) *store.SqliteStore {
	t.Helper()
	s, err := store.NewSqliteStore(":memory:")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = s.DB.Close() })
	if err := migrate.RunMigrations(s.DB, filepath.Join("..", "..", "migrations")); err != nil {
		t.Fatal(err)
	}
	return s
}

func TestNewWiring_ComposesWorkingCoreServices(t *testing.T) {
	if err := auth.ConfigureJWT("test-secret-123456"); err != nil {
		t.Fatal(err)
	}

	s := newTestStore(t)
	w := NewWiring(s)
	if w == nil || w.Auth == nil || w.Sessions == nil || w.Tokens == nil || w.Identity == nil || w.Devices == nil || w.Contacts == nil || w.Ledger == nil || w.MessagingThreads == nil {
		t.Fatalf("unexpected nil wiring/services: %+v", w)
	}

	principal, err := w.Auth.RegisterPassword(context.Background(), coreid.PasswordCredential{
		Username: "alice",
		Password: "password123",
	})
	if err != nil {
		t.Fatalf("RegisterPassword error: %v", err)
	}
	if principal.Username != "alice" {
		t.Fatalf("principal username = %q, want alice", principal.Username)
	}

	tokens, err := w.Auth.LoginPassword(context.Background(), coreid.PasswordCredential{
		Username: "alice",
		Password: "password123",
	}, coreid.SessionMetadata{
		DeviceLabel: "Test browser",
		UserAgent:   "test-agent",
		IPAddress:   "127.0.0.1",
	})
	if err != nil {
		t.Fatalf("LoginPassword error: %v", err)
	}
	if tokens.AccessToken == "" || tokens.RefreshToken == "" {
		t.Fatalf("expected access and refresh tokens, got %+v", tokens)
	}

	profile, err := w.Identity.GetProfile(context.Background(), principal.ID)
	if err != nil {
		t.Fatalf("GetProfile error: %v", err)
	}
	if profile.Username != "alice" {
		t.Fatalf("profile username = %q, want alice", profile.Username)
	}

	device, err := w.Devices.RegisterDeviceIdentity(context.Background(), principal.ID, tokens.Session.ID, coreid.RegisterDeviceIdentityRequest{
		Label:                 "Laptop",
		IdentityKey:           "identity-key",
		SignedPrekeyID:        1,
		SignedPrekey:          "signed-prekey",
		SignedPrekeySignature: "signature",
		Prekeys: []coreid.DevicePrekeyUpload{
			{PrekeyID: 1, PublicKey: "prekey-1"},
		},
	})
	if err != nil {
		t.Fatalf("RegisterDeviceIdentity error: %v", err)
	}
	if !device.CurrentSession {
		t.Fatalf("expected device to be associated with current session: %+v", device)
	}

	bobPrincipal, err := w.Auth.RegisterPassword(context.Background(), coreid.PasswordCredential{
		Username: "bob",
		Password: "password123",
	})
	if err != nil {
		t.Fatalf("register bob error: %v", err)
	}

	if err := w.Contacts.SendInviteByUsername(context.Background(), corecontacts.UserID(principal.ID), "bob"); err != nil {
		t.Fatalf("SendInviteByUsername error: %v", err)
	}
	if err := w.Contacts.RespondToInvite(context.Background(), 1, corecontacts.UserID(bobPrincipal.ID), corecontacts.InviteAccepted); err != nil {
		t.Fatalf("RespondToInvite error: %v", err)
	}

	contacts, err := w.Contacts.ListContacts(context.Background(), corecontacts.UserID(principal.ID))
	if err != nil {
		t.Fatalf("ListContacts error: %v", err)
	}
	if len(contacts) != 1 || contacts[0].Username != "bob" {
		t.Fatalf("unexpected contacts: %+v", contacts)
	}

	account, err := w.Ledger.GetAccount(context.Background(), int(principal.ID))
	if err != nil {
		t.Fatalf("GetAccount error: %v", err)
	}
	if account.OwnerUserID != int(principal.ID) {
		t.Fatalf("account owner = %d, want %d", account.OwnerUserID, principal.ID)
	}
}

func TestWSHelpers_ResolveUserAndAuthenticateToken(t *testing.T) {
	if err := auth.ConfigureJWT("test-secret-123456"); err != nil {
		t.Fatal(err)
	}

	s := newTestStore(t)
	userID, err := s.CreateUser("alice", "password123")
	if err != nil {
		t.Fatal(err)
	}
	w := NewWiring(s)
	tokens, err := w.Auth.LoginPassword(context.Background(), coreid.PasswordCredential{
		Username: "alice",
		Password: "password123",
	}, coreid.SessionMetadata{
		DeviceLabel: "CLI",
		UserAgent:   "test-agent",
		IPAddress:   "127.0.0.1",
	})
	if err != nil {
		t.Fatal(err)
	}

	authn := WSAuthenticator(w.Tokens, s)
	resolve := WSResolveUserID(s)

	gotID, gotUsername, gotSessionID, err := authn(tokens.AccessToken)
	if err != nil {
		t.Fatalf("WSAuthenticator error: %v", err)
	}
	if gotID != userID || gotUsername != "alice" {
		t.Fatalf("unexpected authn result id=%d username=%q", gotID, gotUsername)
	}
	if gotSessionID == 0 {
		t.Fatal("expected non-zero session id")
	}

	resolvedID, err := resolve("alice")
	if err != nil {
		t.Fatalf("WSResolveUserID error: %v", err)
	}
	if resolvedID != userID {
		t.Fatalf("resolvedID = %d, want %d", resolvedID, userID)
	}
}
