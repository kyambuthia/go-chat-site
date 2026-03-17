package httpapi

import (
	"database/sql"
	"errors"
	"log"
	"net/http"
	"time"

	"github.com/kyambuthia/go-chat-site/server/internal/adapters/transport/wsrelay"
	"github.com/kyambuthia/go-chat-site/server/internal/app"
	"github.com/kyambuthia/go-chat-site/server/internal/auth"
	"github.com/kyambuthia/go-chat-site/server/internal/config"
	coremsg "github.com/kyambuthia/go-chat-site/server/internal/core/messaging"
	"github.com/kyambuthia/go-chat-site/server/internal/store"
	"github.com/kyambuthia/go-chat-site/server/internal/web"
)

// NewRouter builds the mux-based HTTP+WS adapter while preserving current route paths.
func NewRouter(dataStore store.APIStore, hub *wsrelay.Hub) http.Handler {
	mux := http.NewServeMux()
	wsLimiterImpl := requestRateLimiter(newFixedWindowRateLimiter(config.WSHandshakeRateLimitPerMinute(), time.Minute))
	var authSecurity *authSecurity
	if dbProvider, ok := dataStore.(interface{ SQLDB() *sql.DB }); ok && dbProvider.SQLDB() != nil {
		wsShared, err := newSharedWindowRateLimiter(dbProvider.SQLDB(), config.WSHandshakeRateLimitPerMinute(), time.Minute)
		if err != nil {
			log.Printf("warn: shared websocket rate limiter disabled: %v", err)
		} else {
			wsLimiterImpl = wsShared
		}
		authSecurity, err = newAuthSecurity(dbProvider.SQLDB())
		if err != nil {
			log.Printf("warn: auth security controls disabled: %v", err)
		}
	}
	wsHandshakeLimiter := rateLimitMiddleware(wsLimiterImpl)
	wiring := app.NewWiring(dataStore)
	hub.SetDeliveryService(coremsg.NewDurableRelayServiceWithCorrelation(hub, wiring.MessagingPersistence, wiring.MessagingCorrelation))
	authMiddleware := auth.Middleware(wiring.Tokens)

	authHandler := &AuthHandler{Identity: wiring.Auth, Sessions: wiring.Sessions, Security: authSecurity, SessionHub: hub}
	contactsHandler := &ContactsHandler{Contacts: wiring.Contacts}
	inviteHandler := &InviteHandler{Contacts: wiring.Contacts}
	walletHandler := &WalletHandler{Ledger: wiring.Ledger}
	messagesHandler := &MessagesHandler{Messaging: wiring.MessagingPersistence, Threads: wiring.MessagingThreads, ReceiptTransport: hub}
	meHandler := &MeHandler{Identity: wiring.Identity}
	deviceKeysHandler := &DeviceKeysHandler{Devices: wiring.Devices}

	mux.HandleFunc("/healthz", healthzHandler)
	mux.HandleFunc("/readyz", readyzHandler(readinessCheck(dataStore)))

	mux.HandleFunc("/api/register", authHandler.Register)
	mux.HandleFunc("/api/login", authHandler.Login)
	mux.HandleFunc("/api/auth/refresh", authHandler.Refresh)

	mux.Handle("/api/logout", authMiddleware(http.HandlerFunc(authHandler.Logout)))
	mux.Handle("/api/sessions", authMiddleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodGet:
			authHandler.GetSessions(w, r)
		case http.MethodDelete:
			authHandler.RevokeSession(w, r)
		default:
			web.JSONError(w, errors.New("method not allowed"), http.StatusMethodNotAllowed)
		}
	})))

	mux.Handle("/api/contacts", authMiddleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodGet:
			contactsHandler.GetContacts(w, r)
		case http.MethodPost:
			contactsHandler.AddContact(w, r)
		case http.MethodDelete:
			contactsHandler.RemoveContact(w, r)
		default:
			web.JSONError(w, errors.New("method not allowed"), http.StatusMethodNotAllowed)
		}
	})))

	mux.Handle("/api/invites", authMiddleware(http.HandlerFunc(inviteHandler.GetInvites)))
	mux.Handle("/api/invites/send", authMiddleware(http.HandlerFunc(inviteHandler.SendInvite)))
	mux.Handle("/api/invites/accept", authMiddleware(http.HandlerFunc(inviteHandler.AcceptInvite)))
	mux.Handle("/api/invites/reject", authMiddleware(http.HandlerFunc(inviteHandler.RejectInvite)))

	mux.Handle("/api/me", authMiddleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodGet:
			meHandler.GetMe(w, r)
		case http.MethodPatch:
			meHandler.UpdateMe(w, r)
		default:
			web.JSONError(w, errors.New("method not allowed"), http.StatusMethodNotAllowed)
		}
	})))
	mux.Handle("/api/wallet", authMiddleware(http.HandlerFunc(walletHandler.GetWallet)))
	mux.Handle("/api/wallet/transfers", authMiddleware(http.HandlerFunc(walletHandler.GetTransfers)))
	mux.Handle("/api/wallet/send", authMiddleware(http.HandlerFunc(walletHandler.SendMoney)))
	mux.Handle("/api/messages/inbox", authMiddleware(http.HandlerFunc(messagesHandler.GetInbox)))
	mux.Handle("/api/messages/outbox", authMiddleware(http.HandlerFunc(messagesHandler.GetOutbox)))
	mux.Handle("/api/messages/threads", authMiddleware(http.HandlerFunc(messagesHandler.GetThreads)))
	mux.Handle("/api/messaging/threads", authMiddleware(http.HandlerFunc(messagesHandler.GetThreads)))
	mux.Handle("/api/messages/sync", authMiddleware(http.HandlerFunc(messagesHandler.GetSync)))
	mux.Handle("/api/messaging/sync", authMiddleware(http.HandlerFunc(messagesHandler.GetSync)))
	mux.Handle("/api/messages/read", authMiddleware(http.HandlerFunc(messagesHandler.MarkRead)))
	mux.Handle("/api/messages/delivered", authMiddleware(http.HandlerFunc(messagesHandler.MarkDelivered)))
	mux.Handle("/api/devices", authMiddleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodGet:
			deviceKeysHandler.GetDevices(w, r)
		case http.MethodPost:
			deviceKeysHandler.RegisterDevice(w, r)
		case http.MethodDelete:
			deviceKeysHandler.RevokeDevice(w, r)
		default:
			web.JSONError(w, errors.New("method not allowed"), http.StatusMethodNotAllowed)
		}
	})))
	mux.Handle("/api/devices/rotate", authMiddleware(http.HandlerFunc(deviceKeysHandler.RotateDevice)))
	mux.Handle("/api/messaging/prekeys", authMiddleware(http.HandlerFunc(deviceKeysHandler.PublishPrekeys)))
	mux.Handle("/api/devices/directory", authMiddleware(http.HandlerFunc(deviceKeysHandler.GetDirectory)))

	mux.Handle("/ws", wsHandshakeLimiter(wsrelay.WebSocketHandler(hub, app.WSAuthenticator(wiring.Tokens, dataStore), app.WSResolveUserID(dataStore))))

	return mux
}
