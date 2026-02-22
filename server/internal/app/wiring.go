package app

import (
	"github.com/kyambuthia/go-chat-site/server/internal/adapters/store/sqlitecontacts"
	"github.com/kyambuthia/go-chat-site/server/internal/adapters/store/sqliteledger"
	"github.com/kyambuthia/go-chat-site/server/internal/auth"
	corecontacts "github.com/kyambuthia/go-chat-site/server/internal/core/contacts"
	coreledger "github.com/kyambuthia/go-chat-site/server/internal/core/ledger"
	"github.com/kyambuthia/go-chat-site/server/internal/store"
)

// Wiring assembles core services and adapter-backed helper functions for HTTP/WS composition.
type Wiring struct {
	Contacts corecontacts.Service
	Ledger   coreledger.Service
}

func NewWiring(dataStore store.APIStore) *Wiring {
	contactsAdapter := &sqlitecontacts.Adapter{Store: dataStore}
	ledgerAdapter := &sqliteledger.Adapter{WalletStore: dataStore}

	return &Wiring{
		Contacts: corecontacts.NewService(contactsAdapter, contactsAdapter),
		Ledger:   coreledger.NewService(ledgerAdapter, ledgerAdapter),
	}
}

func WSAuthenticator(dataStore store.APIStore) func(token string) (int, string, error) {
	return func(token string) (int, string, error) {
		claims, err := auth.ValidateToken(token)
		if err != nil {
			return 0, "", err
		}
		user, err := dataStore.GetUserByID(claims.UserID)
		if err != nil {
			return 0, "", err
		}
		return user.ID, user.Username, nil
	}
}

func WSResolveUserID(dataStore store.APIStore) func(username string) (int, error) {
	return func(username string) (int, error) {
		user, err := dataStore.GetUserByUsername(username)
		if err != nil {
			return 0, err
		}
		return user.ID, nil
	}
}
