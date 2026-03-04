package store

import (
	"path/filepath"
	"testing"

	"github.com/kyambuthia/go-chat-site/server/internal/migrate"
)

func newStoreForTest(t *testing.T) *SqliteStore {
	t.Helper()
	s, err := NewSqliteStore(":memory:")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = s.DB.Close() })
	if err := migrate.RunMigrations(s.DB, filepath.Join("..", "..", "migrations")); err != nil {
		t.Fatal(err)
	}
	return s
}

func TestCreateUser_RejectsBlankUsername(t *testing.T) {
	s := newStoreForTest(t)
	if _, err := s.CreateUser("   ", "password123"); err == nil {
		t.Fatal("expected username validation error")
	}
}

func TestGetUserByUsername_NotFound(t *testing.T) {
	s := newStoreForTest(t)
	if _, err := s.GetUserByUsername("missing"); err != ErrNotFound {
		t.Fatalf("error = %v, want %v", err, ErrNotFound)
	}
}

func TestContactsAndInvitesFlow(t *testing.T) {
	s := newStoreForTest(t)
	alice, err := s.CreateUser("alice", "password123")
	if err != nil {
		t.Fatal(err)
	}
	bob, err := s.CreateUser("bob", "password123")
	if err != nil {
		t.Fatal(err)
	}

	if err := s.CreateInvite(alice, bob); err != nil {
		t.Fatalf("CreateInvite failed: %v", err)
	}
	if err := s.CreateInvite(alice, bob); err != ErrInviteExists {
		t.Fatalf("duplicate invite error = %v, want %v", err, ErrInviteExists)
	}

	invites, err := s.ListInvites(bob)
	if err != nil {
		t.Fatal(err)
	}
	if len(invites) != 1 {
		t.Fatalf("invites len = %d, want 1", len(invites))
	}

	if err := s.UpdateInviteStatus(invites[0].ID, bob, "accepted"); err != nil {
		t.Fatalf("UpdateInviteStatus failed: %v", err)
	}

	contacts, err := s.ListContacts(alice)
	if err != nil {
		t.Fatal(err)
	}
	if len(contacts) != 1 || contacts[0].Username != "bob" {
		t.Fatalf("unexpected contacts: %#v", contacts)
	}

	if err := s.RemoveContact(alice, bob); err != nil {
		t.Fatal(err)
	}
	contacts, err = s.ListContacts(alice)
	if err != nil {
		t.Fatal(err)
	}
	if len(contacts) != 0 {
		t.Fatalf("contacts len after remove = %d, want 0", len(contacts))
	}
}

func TestWalletHelpers(t *testing.T) {
	s := newStoreForTest(t)
	userID, err := s.CreateUser("wallet-user", "password123")
	if err != nil {
		t.Fatal(err)
	}

	wallet, err := s.GetWallet(userID)
	if err != nil {
		t.Fatal(err)
	}
	if wallet.UserID != userID {
		t.Fatalf("wallet user id = %d, want %d", wallet.UserID, userID)
	}
	if wallet.BalanceFloat() != 0 {
		t.Fatalf("wallet balance float = %f, want 0", wallet.BalanceFloat())
	}

	if _, err := DollarsToCents(0); err == nil {
		t.Fatal("expected cents conversion error for zero")
	}
	cents, err := DollarsToCents(12.34)
	if err != nil {
		t.Fatal(err)
	}
	if cents != 1234 {
		t.Fatalf("cents = %d, want 1234", cents)
	}
}
