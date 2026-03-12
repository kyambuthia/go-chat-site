package sqliteledger

import (
	"context"
	"errors"
	"testing"
	"time"

	coreledger "github.com/kyambuthia/go-chat-site/server/internal/core/ledger"
	"github.com/kyambuthia/go-chat-site/server/internal/store"
)

type fakeWalletStore struct {
	wallet     *store.Wallet
	history    []store.WalletTransfer
	user       *store.User
	getErr     error
	historyErr error
	userErr    error
	sendErr    error
	lastSend   struct {
		from, to int
		cents    int64
	}
}

func (f *fakeWalletStore) GetWallet(userID int) (*store.Wallet, error) {
	_ = userID
	if f.getErr != nil {
		return nil, f.getErr
	}
	return f.wallet, nil
}

func (f *fakeWalletStore) ListTransfers(userID, limit int) ([]store.WalletTransfer, error) {
	_, _ = userID, limit
	if f.historyErr != nil {
		return nil, f.historyErr
	}
	return f.history, nil
}

func (f *fakeWalletStore) GetUserByUsername(username string) (*store.User, error) {
	_ = username
	if f.userErr != nil {
		return nil, f.userErr
	}
	return f.user, nil
}

func (f *fakeWalletStore) SendMoney(senderID, recipientID int, amountCents int64) error {
	f.lastSend = struct {
		from, to int
		cents    int64
	}{from: senderID, to: recipientID, cents: amountCents}
	return f.sendErr
}

func TestAdapter_GetAccount_MapsWalletToLedgerAccount(t *testing.T) {
	a := &Adapter{WalletStore: &fakeWalletStore{
		wallet: &store.Wallet{ID: 5, UserID: 11, BalanceCents: 1234},
	}}

	got, err := a.GetAccount(context.Background(), 11)
	if err != nil {
		t.Fatalf("GetAccount error: %v", err)
	}
	if got.ID != "5" || got.OwnerUserID != 11 || got.BalanceCents != 1234 || got.CurrencyCode != "USD" {
		t.Fatalf("unexpected mapped account: %+v", got)
	}
}

func TestAdapter_Transfer_DelegatesToWalletStore(t *testing.T) {
	fs := &fakeWalletStore{}
	a := &Adapter{WalletStore: fs}
	in := coreledger.Transfer{FromUserID: 1, ToUserID: 2, AmountCents: 450}

	got, err := a.Transfer(context.Background(), in)
	if err != nil {
		t.Fatalf("Transfer error: %v", err)
	}
	if got.AmountCents != 450 {
		t.Fatalf("unexpected transfer result: %+v", got)
	}
	if fs.lastSend.from != 1 || fs.lastSend.to != 2 || fs.lastSend.cents != 450 {
		t.Fatalf("unexpected SendMoney call: %+v", fs.lastSend)
	}
}

func TestAdapter_Transfer_PropagatesStoreError(t *testing.T) {
	wantErr := errors.New("db down")
	a := &Adapter{WalletStore: &fakeWalletStore{sendErr: wantErr}}

	_, err := a.Transfer(context.Background(), coreledger.Transfer{FromUserID: 1, ToUserID: 2, AmountCents: 1})
	if !errors.Is(err, wantErr) {
		t.Fatalf("expected %v, got %v", wantErr, err)
	}
}

func TestAdapter_ListTransfers_MapsWalletHistoryToLedgerRecords(t *testing.T) {
	createdAt := time.Date(2026, time.March, 12, 10, 0, 0, 0, time.UTC)
	a := &Adapter{WalletStore: &fakeWalletStore{
		history: []store.WalletTransfer{{
			ID:                      5,
			Direction:               "received",
			CounterpartyUserID:      22,
			CounterpartyUsername:    "bob",
			CounterpartyDisplayName: "Bob",
			CounterpartyAvatarURL:   "https://example.com/bob.png",
			AmountCents:             450,
			CreatedAt:               createdAt,
		}},
	}}

	got, err := a.ListTransfers(context.Background(), 11, 10)
	if err != nil {
		t.Fatalf("ListTransfers error: %v", err)
	}
	if len(got) != 1 {
		t.Fatalf("history len = %d, want 1", len(got))
	}
	if got[0].ID != "5" || got[0].CounterpartyUsername != "bob" || got[0].CurrencyCode != "USD" {
		t.Fatalf("unexpected mapped transfer: %+v", got[0])
	}
	if !got[0].CreatedAt.Equal(createdAt) {
		t.Fatalf("created_at = %v, want %v", got[0].CreatedAt, createdAt)
	}
}
