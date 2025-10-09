package api

import (
	"fmt"
	"net/http"

	"github.com/kyambuthia/go-chat-site/server/internal/auth"
	"github.com/kyambuthia/go-chat-site/server/internal/handlers"
	"github.com/kyambuthia/go-chat-site/server/internal/store"
	"github.com/kyambuthia/go-chat-site/server/internal/ws"
)

func NewAPI(store *store.SqliteStore, hub *ws.Hub) http.Handler {
	mux := http.NewServeMux()

	authHandler := &handlers.AuthHandler{Store: store}
	contactsHandler := &handlers.ContactsHandler{Store: store}
	inviteHandler := &handlers.InviteHandler{Store: store}
	walletHandler := &handlers.WalletHandler{Store: store}

	// Auth
	mux.HandleFunc("/api/register", authHandler.Register)
	mux.HandleFunc("/api/login", auth.Login(store))

	// Contacts
	mux.Handle("/api/contacts", auth.Middleware(http.HandlerFunc(contactsHandler.GetContacts)))

	// Invites
	mux.Handle("/api/invites", auth.Middleware(http.HandlerFunc(inviteHandler.GetInvites)))
	mux.Handle("/api/invites/send", auth.Middleware(http.HandlerFunc(inviteHandler.SendInvite)))
	mux.Handle("/api/invites/accept", auth.Middleware(http.HandlerFunc(inviteHandler.AcceptInvite)))
	mux.Handle("/api/invites/reject", auth.Middleware(http.HandlerFunc(inviteHandler.RejectInvite)))

	// Wallet
	mux.Handle("/api/wallet", auth.Middleware(http.HandlerFunc(walletHandler.GetWallet)))
	mux.Handle("/api/wallet/send", auth.Middleware(http.HandlerFunc(walletHandler.SendMoney)))

	// Websocket
	authenticator := func(token string) (int, string, error) {
		claims, err := auth.ValidateToken(token)
		if err != nil {
			return 0, "", err
		}
		user, err := store.GetUserByID(claims.UserID)
		if err != nil {
			return 0, "", err
		}
		return user.ID, user.Username, nil
	}

	resolve := func(username string) (int, error) {
		user, err := store.GetUserByUsername(username)
		if err != nil {
			return 0, err
		}
		return user.ID, nil
	}

	mux.HandleFunc("/ws", ws.WebSocketHandler(hub, authenticator, resolve))

	return loggingMiddleware(mux)
}

func loggingMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Println("Request received")
		next.ServeHTTP(w, r)
	})
}