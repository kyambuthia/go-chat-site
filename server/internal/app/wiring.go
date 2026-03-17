package app

import (
	"context"
	"database/sql"
	"errors"

	"github.com/kyambuthia/go-chat-site/server/internal/adapters/identity/jwttokens"
	"github.com/kyambuthia/go-chat-site/server/internal/adapters/identity/passwordbcrypt"
	"github.com/kyambuthia/go-chat-site/server/internal/adapters/store/sqlitecontacts"
	"github.com/kyambuthia/go-chat-site/server/internal/adapters/store/sqliteidentity"
	"github.com/kyambuthia/go-chat-site/server/internal/adapters/store/sqliteidentityauth"
	"github.com/kyambuthia/go-chat-site/server/internal/adapters/store/sqliteledger"
	"github.com/kyambuthia/go-chat-site/server/internal/adapters/store/sqlitemessaging"
	"github.com/kyambuthia/go-chat-site/server/internal/config"
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
	Sessions             coreid.SessionService
	Tokens               coreid.TokenService
	Identity             coreid.ProfileService
	Devices              coreid.DeviceIdentityService
	Ledger               coreledger.Service
	MessagingPersistence coremsg.PersistenceService
	MessagingThreads     coremsg.ThreadSummaryService
	MessagingCorrelation coremsg.ClientMessageCorrelationRecorder
}

func NewWiring(dataStore store.APIStore) *Wiring {
	contactsAdapter := &sqlitecontacts.Adapter{Store: dataStore}
	authAdapter := &sqliteidentityauth.Adapter{Store: dataStore}
	identityAdapter := &sqliteidentity.Adapter{Store: dataStore}
	ledgerAdapter := &sqliteledger.Adapter{WalletStore: dataStore}
	var messagingPersistence coremsg.PersistenceService
	tokenAdapter := &jwttokens.Adapter{
		AccessTTL:  config.AccessTokenTTL(),
		RefreshTTL: config.RefreshTokenTTL(),
	}
	if dbProvider, ok := dataStore.(interface{ SQLDB() *sql.DB }); ok && dbProvider.SQLDB() != nil {
		tokenAdapter.DB = dbProvider.SQLDB()
		deviceKeysAdapter := &sqliteidentity.DeviceKeysAdapter{DB: dbProvider.SQLDB()}
		messagingAdapter := &sqlitemessaging.Adapter{DB: dbProvider.SQLDB()}
		messagingPersistence = coremsg.NewPersistenceService(messagingAdapter)
		return &Wiring{
			Contacts:             corecontacts.NewService(contactsAdapter, contactsAdapter),
			Auth:                 coreid.NewAuthService(authAdapter, passwordbcrypt.Verifier{}, tokenAdapter),
			Sessions:             coreid.NewSessionService(tokenAdapter),
			Tokens:               tokenAdapter,
			Identity:             coreid.NewProfileService(identityAdapter),
			Devices:              coreid.NewDeviceIdentityService(deviceKeysAdapter),
			Ledger:               coreledger.NewService(ledgerAdapter, ledgerAdapter),
			MessagingPersistence: messagingPersistence,
			MessagingThreads:     coremsg.NewThreadSummaryService(messagingAdapter),
			MessagingCorrelation: messagingAdapter,
		}
	}

	return &Wiring{
		Contacts:             corecontacts.NewService(contactsAdapter, contactsAdapter),
		Auth:                 coreid.NewAuthService(authAdapter, passwordbcrypt.Verifier{}, tokenAdapter),
		Sessions:             coreid.NewSessionService(tokenAdapter),
		Tokens:               tokenAdapter,
		Identity:             coreid.NewProfileService(identityAdapter),
		Devices:              nil,
		Ledger:               coreledger.NewService(ledgerAdapter, ledgerAdapter),
		MessagingPersistence: messagingPersistence,
	}
}

func WSAuthenticator(tokens coreid.TokenService, dataStore store.APIStore) func(token string) (int, string, int64, error) {
	return func(token string) (int, string, int64, error) {
		if tokens == nil {
			return 0, "", 0, errors.New("auth unavailable")
		}
		claims, err := tokens.ValidateToken(context.Background(), token)
		if err != nil {
			return 0, "", 0, err
		}
		user, err := dataStore.GetUserByID(int(claims.SubjectUserID))
		if err != nil {
			return 0, "", 0, err
		}
		return user.ID, user.Username, claims.SessionID, nil
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
