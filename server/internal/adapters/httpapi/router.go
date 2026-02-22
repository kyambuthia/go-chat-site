package httpapi

import (
	"net/http"
	"time"

	"github.com/kyambuthia/go-chat-site/server/internal/adapters/transport/wsrelay"
	"github.com/kyambuthia/go-chat-site/server/internal/auth"
	"github.com/kyambuthia/go-chat-site/server/internal/config"
	"github.com/kyambuthia/go-chat-site/server/internal/store"
)

// NewRouter builds the mux-based HTTP+WS adapter while preserving current route paths.
func NewRouter(dataStore store.APIStore, hub *wsrelay.Hub) http.Handler {
	mux := http.NewServeMux()
	loginLimiter := rateLimitMiddleware(newFixedWindowRateLimiter(config.LoginRateLimitPerMinute(), time.Minute))
	wsHandshakeLimiter := rateLimitMiddleware(newFixedWindowRateLimiter(config.WSHandshakeRateLimitPerMinute(), time.Minute))

	authHandler := &AuthHandler{Store: dataStore}
	contactsHandler := &ContactsHandler{Store: dataStore}
	inviteHandler := &InviteHandler{Store: dataStore}
	walletHandler := &WalletHandler{Store: dataStore}
	meHandler := &MeHandler{Store: dataStore}

	mux.HandleFunc("/api/register", authHandler.Register)
	mux.Handle("/api/login", loginLimiter(auth.Login(dataStore)))

	mux.Handle("/api/contacts", auth.Middleware(http.HandlerFunc(contactsHandler.GetContacts)))

	mux.Handle("/api/invites", auth.Middleware(http.HandlerFunc(inviteHandler.GetInvites)))
	mux.Handle("/api/invites/send", auth.Middleware(http.HandlerFunc(inviteHandler.SendInvite)))
	mux.Handle("/api/invites/accept", auth.Middleware(http.HandlerFunc(inviteHandler.AcceptInvite)))
	mux.Handle("/api/invites/reject", auth.Middleware(http.HandlerFunc(inviteHandler.RejectInvite)))

	mux.Handle("/api/me", auth.Middleware(http.HandlerFunc(meHandler.GetMe)))
	mux.Handle("/api/wallet", auth.Middleware(http.HandlerFunc(walletHandler.GetWallet)))
	mux.Handle("/api/wallet/send", auth.Middleware(http.HandlerFunc(walletHandler.SendMoney)))

	authenticator := func(token string) (int, string, error) {
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

	resolve := func(username string) (int, error) {
		user, err := dataStore.GetUserByUsername(username)
		if err != nil {
			return 0, err
		}
		return user.ID, nil
	}

	mux.Handle("/ws", wsHandshakeLimiter(wsrelay.WebSocketHandler(hub, authenticator, resolve)))

	return mux
}
