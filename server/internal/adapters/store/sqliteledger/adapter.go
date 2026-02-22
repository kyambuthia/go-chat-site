package sqliteledger

import (
	"context"
	"fmt"

	coreledger "github.com/kyambuthia/go-chat-site/server/internal/core/ledger"
	"github.com/kyambuthia/go-chat-site/server/internal/store"
)

// Adapter bridges the existing SQLite wallet store semantics to the core ledger service.
type Adapter struct {
	WalletStore store.WalletStore
}

var _ coreledger.Repository = (*Adapter)(nil)
var _ coreledger.UserDirectory = (*Adapter)(nil)

func (a *Adapter) GetAccount(ctx context.Context, userID int) (coreledger.Account, error) {
	_ = ctx
	wallet, err := a.WalletStore.GetWallet(userID)
	if err != nil {
		return coreledger.Account{}, err
	}
	return coreledger.Account{
		ID:           coreledger.AccountID(fmt.Sprintf("%d", wallet.ID)),
		OwnerUserID:  wallet.UserID,
		BalanceCents: wallet.BalanceCents,
		CurrencyCode: "USD",
	}, nil
}

func (a *Adapter) Transfer(ctx context.Context, transfer coreledger.Transfer) (coreledger.Transfer, error) {
	_ = ctx
	if err := a.WalletStore.SendMoney(transfer.FromUserID, transfer.ToUserID, transfer.AmountCents); err != nil {
		return coreledger.Transfer{}, err
	}
	return transfer, nil
}

func (a *Adapter) ResolveUserIDByUsername(ctx context.Context, username string) (int, error) {
	_ = ctx
	user, err := a.WalletStore.GetUserByUsername(username)
	if err != nil {
		return 0, err
	}
	return user.ID, nil
}
