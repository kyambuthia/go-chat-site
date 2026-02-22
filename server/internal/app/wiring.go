package app

import (
	"database/sql"

	"github.com/kyambuthia/go-chat-site/server/internal/adapters/identity/jwttokens"
	"github.com/kyambuthia/go-chat-site/server/internal/adapters/identity/passwordbcrypt"
	"github.com/kyambuthia/go-chat-site/server/internal/adapters/store/sqlitecontacts"
	"github.com/kyambuthia/go-chat-site/server/internal/adapters/store/sqliteidentity"
	"github.com/kyambuthia/go-chat-site/server/internal/adapters/store/sqliteidentityauth"
	"github.com/kyambuthia/go-chat-site/server/internal/adapters/store/sqliteledger"
	"github.com/kyambuthia/go-chat-site/server/internal/adapters/store/sqlitemessaging"
	"github.com/kyambuthia/go-chat-site/server/internal/auth"
	corecontacts "github.com/kyambuthia/go-chat-site/server/internal/core/contacts"
	coreid "github.com/kyambuthia/go-chat-site/server/internal/core/identity"
	coreledger "github.com/kyambuthia/go-chat-site/server/internal/core/ledger"
	coremsg "github.com/kyambuthia/go-chat-site/server/internal/core/messaging"
	"github.com/kyambuthia/go-chat-site/server/internal/store"
)

// Wiring assembles core services and adapter-backed helper functions for HTTP/WS composition.
type Wiring struct {
	Contacts             corecontacts.Service
	Auth                 coreid.AuthService
	Identity             coreid.ProfileService
	Ledger               coreledger.Service
	MessagingPersistence coremsg.PersistenceService
	MessagingCorrelation coremsg.ClientMessageCorrelationRecorder
}

func NewWiring(dataStore store.APIStore) *Wiring {
	contactsAdapter := &sqlitecontacts.Adapter{Store: dataStore}
	authAdapter := &sqliteidentityauth.Adapter{Store: dataStore}
	identityAdapter := &sqliteidentity.Adapter{Store: dataStore}
	ledgerAdapter := &sqliteledger.Adapter{WalletStore: dataStore}
	var messagingPersistence coremsg.PersistenceService
	if dbProvider, ok := dataStore.(interface{ SQLDB() *sql.DB }); ok && dbProvider.SQLDB() != nil {
		messagingAdapter := &sqlitemessaging.Adapter{DB: dbProvider.SQLDB()}
		messagingPersistence = coremsg.NewPersistenceService(messagingAdapter)
		return &Wiring{
			Contacts:             corecontacts.NewService(contactsAdapter, contactsAdapter),
			Auth:                 coreid.NewAuthService(authAdapter, passwordbcrypt.Verifier{}, &jwttokens.Adapter{}),
			Identity:             coreid.NewProfileService(identityAdapter),
			Ledger:               coreledger.NewService(ledgerAdapter, ledgerAdapter),
			MessagingPersistence: messagingPersistence,
			MessagingCorrelation: messagingAdapter,
		}
	}

	return &Wiring{
		Contacts:             corecontacts.NewService(contactsAdapter, contactsAdapter),
		Auth:                 coreid.NewAuthService(authAdapter, passwordbcrypt.Verifier{}, &jwttokens.Adapter{}),
		Identity:             coreid.NewProfileService(identityAdapter),
		Ledger:               coreledger.NewService(ledgerAdapter, ledgerAdapter),
		MessagingPersistence: messagingPersistence,
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
