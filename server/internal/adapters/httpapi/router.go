package httpapi

import (
	"database/sql"
	"log"
	"net/http"
	"time"

	"github.com/kyambuthia/go-chat-site/server/internal/adapters/transport/wsrelay"
	"github.com/kyambuthia/go-chat-site/server/internal/app"
	"github.com/kyambuthia/go-chat-site/server/internal/auth"
	"github.com/kyambuthia/go-chat-site/server/internal/config"
	coremsg "github.com/kyambuthia/go-chat-site/server/internal/core/messaging"
	"github.com/kyambuthia/go-chat-site/server/internal/store"
)

// NewRouter builds the mux-based HTTP+WS adapter while preserving current route paths.
func NewRouter(dataStore store.APIStore, hub *wsrelay.Hub) http.Handler {
	mux := http.NewServeMux()
	loginLimiterImpl := requestRateLimiter(newFixedWindowRateLimiter(config.LoginRateLimitPerMinute(), time.Minute))
	wsLimiterImpl := requestRateLimiter(newFixedWindowRateLimiter(config.WSHandshakeRateLimitPerMinute(), time.Minute))
	if dbProvider, ok := dataStore.(interface{ SQLDB() *sql.DB }); ok && dbProvider.SQLDB() != nil {
		loginShared, err := newSharedWindowRateLimiter(dbProvider.SQLDB(), config.LoginRateLimitPerMinute(), time.Minute)
		if err != nil {
			log.Printf("warn: shared login rate limiter disabled: %v", err)
		} else {
			loginLimiterImpl = loginShared
		}

		wsShared, err := newSharedWindowRateLimiter(dbProvider.SQLDB(), config.WSHandshakeRateLimitPerMinute(), time.Minute)
		if err != nil {
			log.Printf("warn: shared websocket rate limiter disabled: %v", err)
		} else {
			wsLimiterImpl = wsShared
		}
	}
	loginLimiter := rateLimitMiddleware(loginLimiterImpl)
	wsHandshakeLimiter := rateLimitMiddleware(wsLimiterImpl)
	wiring := app.NewWiring(dataStore)
	hub.SetDeliveryService(coremsg.NewDurableRelayServiceWithCorrelation(hub, wiring.MessagingPersistence, wiring.MessagingCorrelation))

	authHandler := &AuthHandler{Identity: wiring.Auth}
	contactsHandler := &ContactsHandler{Contacts: wiring.Contacts}
	inviteHandler := &InviteHandler{Contacts: wiring.Contacts}
	walletHandler := &WalletHandler{Ledger: wiring.Ledger}
	messagesHandler := &MessagesHandler{Messaging: wiring.MessagingPersistence, ReceiptTransport: hub}
	meHandler := &MeHandler{Identity: wiring.Identity}

	mux.HandleFunc("/api/register", authHandler.Register)
	mux.Handle("/api/login", loginLimiter(http.HandlerFunc(authHandler.Login)))

	mux.Handle("/api/contacts", auth.Middleware(http.HandlerFunc(contactsHandler.GetContacts)))

	mux.Handle("/api/invites", auth.Middleware(http.HandlerFunc(inviteHandler.GetInvites)))
	mux.Handle("/api/invites/send", auth.Middleware(http.HandlerFunc(inviteHandler.SendInvite)))
	mux.Handle("/api/invites/accept", auth.Middleware(http.HandlerFunc(inviteHandler.AcceptInvite)))
	mux.Handle("/api/invites/reject", auth.Middleware(http.HandlerFunc(inviteHandler.RejectInvite)))

	mux.Handle("/api/me", auth.Middleware(http.HandlerFunc(meHandler.GetMe)))
	mux.Handle("/api/wallet", auth.Middleware(http.HandlerFunc(walletHandler.GetWallet)))
	mux.Handle("/api/wallet/send", auth.Middleware(http.HandlerFunc(walletHandler.SendMoney)))
	mux.Handle("/api/messages/inbox", auth.Middleware(http.HandlerFunc(messagesHandler.GetInbox)))
	mux.Handle("/api/messages/outbox", auth.Middleware(http.HandlerFunc(messagesHandler.GetOutbox)))
	mux.Handle("/api/messages/read", auth.Middleware(http.HandlerFunc(messagesHandler.MarkRead)))
	mux.Handle("/api/messages/delivered", auth.Middleware(http.HandlerFunc(messagesHandler.MarkDelivered)))

	mux.Handle("/ws", wsHandshakeLimiter(wsrelay.WebSocketHandler(hub, app.WSAuthenticator(dataStore), app.WSResolveUserID(dataStore))))

	return mux
}
