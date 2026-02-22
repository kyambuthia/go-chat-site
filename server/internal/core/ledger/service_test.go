package ledger

import (
	"context"
	"errors"
	"testing"
)

type fakeRepo struct {
	accountResp  Account
	accountErr   error
	transferResp Transfer
	transferErr  error
	lastTransfer Transfer
}

func (f *fakeRepo) GetAccount(ctx context.Context, userID int) (Account, error) {
	_ = ctx
	_ = userID
	return f.accountResp, f.accountErr
}

func (f *fakeRepo) Transfer(ctx context.Context, transfer Transfer) (Transfer, error) {
	_ = ctx
	f.lastTransfer = transfer
	if f.transferErr != nil {
		return Transfer{}, f.transferErr
	}
	if f.transferResp.AmountCents == 0 {
		f.transferResp = transfer
	}
	return f.transferResp, nil
}

type fakeDirectory struct {
	userID int
	err    error
	last   string
}

func (f *fakeDirectory) ResolveUserIDByUsername(ctx context.Context, username string) (int, error) {
	_ = ctx
	f.last = username
	return f.userID, f.err
}

func TestService_GetAccount_DelegatesToRepository(t *testing.T) {
	repo := &fakeRepo{accountResp: Account{ID: "acct-1", OwnerUserID: 11, BalanceCents: 2500, CurrencyCode: "USD"}}
	svc := NewService(repo, &fakeDirectory{})

	got, err := svc.GetAccount(context.Background(), 11)
	if err != nil {
		t.Fatalf("GetAccount returned error: %v", err)
	}
	if got.OwnerUserID != 11 || got.BalanceCents != 2500 {
		t.Fatalf("unexpected account: %+v", got)
	}
}

func TestService_SendTransferByUsername_ResolvesRecipientAndCallsRepository(t *testing.T) {
	repo := &fakeRepo{}
	dir := &fakeDirectory{userID: 22}
	svc := NewService(repo, dir)

	transfer, err := svc.SendTransferByUsername(context.Background(), 11, "bob", 1250)
	if err != nil {
		t.Fatalf("SendTransferByUsername returned error: %v", err)
	}

	if dir.last != "bob" {
		t.Fatalf("directory lookup username = %q, want bob", dir.last)
	}
	if repo.lastTransfer.FromUserID != 11 || repo.lastTransfer.ToUserID != 22 || repo.lastTransfer.AmountCents != 1250 {
		t.Fatalf("unexpected transfer passed to repo: %+v", repo.lastTransfer)
	}
	if transfer.ToUserID != 22 || transfer.AmountCents != 1250 {
		t.Fatalf("unexpected transfer response: %+v", transfer)
	}
}

func TestService_SendTransferByUsername_ReturnsRecipientNotFound(t *testing.T) {
	repo := &fakeRepo{}
	dir := &fakeDirectory{err: errors.New("not found")}
	svc := NewService(repo, dir)

	_, err := svc.SendTransferByUsername(context.Background(), 11, "missing", 100)
	if !errors.Is(err, ErrRecipientNotFound) {
		t.Fatalf("expected ErrRecipientNotFound, got %v", err)
	}
}
