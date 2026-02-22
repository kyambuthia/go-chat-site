package store

import (
	"path/filepath"
	"testing"

	"github.com/kyambuthia/go-chat-site/server/internal/migrate"
)

func newLedgerTestStore(t *testing.T) *SqliteStore {
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

func seedUser(t *testing.T, s *SqliteStore, username string) int {
	t.Helper()
	id, err := s.CreateUser(username, "password123")
	if err != nil {
		t.Fatal(err)
	}
	return id
}

func setWalletBalance(t *testing.T, s *SqliteStore, userID int, cents int64) {
	t.Helper()
	if _, err := s.DB.Exec(`INSERT OR IGNORE INTO wallet_accounts (user_id) VALUES (?)`, userID); err != nil {
		t.Fatal(err)
	}
	if _, err := s.DB.Exec(`UPDATE wallet_accounts SET balance_cents = ? WHERE user_id = ?`, cents, userID); err != nil {
		t.Fatal(err)
	}
}

func walletBalance(t *testing.T, s *SqliteStore, userID int) int64 {
	t.Helper()
	var cents int64
	if err := s.DB.QueryRow(`SELECT balance_cents FROM wallet_accounts WHERE user_id = ?`, userID).Scan(&cents); err != nil {
		t.Fatal(err)
	}
	return cents
}

func TestSendMoney_SucceedsWithSufficientBalance(t *testing.T) {
	s := newLedgerTestStore(t)
	aliceID := seedUser(t, s, "alice")
	bobID := seedUser(t, s, "bob")

	setWalletBalance(t, s, aliceID, 1_000)
	setWalletBalance(t, s, bobID, 250)

	if err := s.SendMoney(aliceID, bobID, 300); err != nil {
		t.Fatalf("SendMoney failed: %v", err)
	}

	if got := walletBalance(t, s, aliceID); got != 700 {
		t.Fatalf("sender balance = %d, want 700", got)
	}
	if got := walletBalance(t, s, bobID); got != 550 {
		t.Fatalf("recipient balance = %d, want 550", got)
	}
}

func TestSendMoney_FailsWithInsufficientBalanceAndIsAtomic(t *testing.T) {
	s := newLedgerTestStore(t)
	aliceID := seedUser(t, s, "alice")
	bobID := seedUser(t, s, "bob")

	setWalletBalance(t, s, aliceID, 200)
	setWalletBalance(t, s, bobID, 500)

	beforeAlice := walletBalance(t, s, aliceID)
	beforeBob := walletBalance(t, s, bobID)

	err := s.SendMoney(aliceID, bobID, 300)
	if err == nil {
		t.Fatal("expected insufficient funds error")
	}
	if err != ErrInsufficientFund {
		t.Fatalf("unexpected error: %v", err)
	}

	if got := walletBalance(t, s, aliceID); got != beforeAlice {
		t.Fatalf("sender balance changed on failed transfer: got %d want %d", got, beforeAlice)
	}
	if got := walletBalance(t, s, bobID); got != beforeBob {
		t.Fatalf("recipient balance changed on failed transfer: got %d want %d", got, beforeBob)
	}
}

func TestSendMoney_TransferIsAtomicOnSuccess(t *testing.T) {
	s := newLedgerTestStore(t)
	aliceID := seedUser(t, s, "alice")
	bobID := seedUser(t, s, "bob")

	setWalletBalance(t, s, aliceID, 10_000)
	setWalletBalance(t, s, bobID, 1_000)

	if err := s.SendMoney(aliceID, bobID, 2_500); err != nil {
		t.Fatalf("SendMoney failed: %v", err)
	}

	if got := walletBalance(t, s, aliceID); got != 7_500 {
		t.Fatalf("sender balance = %d, want 7500", got)
	}
	if got := walletBalance(t, s, bobID); got != 3_500 {
		t.Fatalf("recipient balance = %d, want 3500", got)
	}

	var transferCount int
	if err := s.DB.QueryRow(`SELECT COUNT(*) FROM wallet_transfers WHERE sender_user_id = ? AND recipient_user_id = ? AND amount_cents = ?`, aliceID, bobID, 2_500).Scan(&transferCount); err != nil {
		t.Fatal(err)
	}
	if transferCount != 1 {
		t.Fatalf("transfer record count = %d, want 1", transferCount)
	}
}
