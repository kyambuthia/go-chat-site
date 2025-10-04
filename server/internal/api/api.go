package api

import (
	"context"
	"net/http"
	"strings"

	"github.com/kyambuthia/go-chat-site/server/internal/auth"
	"github.com/kyambuthia/go-chat-site/server/internal/handlers"
	"github.com/kyambuthia/go-chat-site/server/internal/store"
	"github.com/kyambuthia/go-chat-site/server/internal/ws"
)

func NewAPI(store *store.SqliteStore, hub *ws.Hub) http.Handler {
	mux := http.NewServeMux()

	authHandler := &handlers.AuthHandler{Store: store}
	mux.HandleFunc("/api/register", authHandler.Register)
	mux.HandleFunc("/api/login", authHandler.Login)

	contactsHandler := &handlers.ContactsHandler{Store: store}
	mux.HandleFunc("/api/contacts", authMiddleware(contactsHandler.GetContacts))
	mux.HandleFunc("/api/contacts/add", authMiddleware(contactsHandler.AddContact))
	mux.HandleFunc("/api/contacts/remove", authMiddleware(contactsHandler.RemoveContact))

	authenticator := func(token string) (int, string, error) {
		claims, err := auth.ValidateToken(token)
		if err != nil {
			return 0, "", err
		}
		// This is a placeholder for getting username from DB
		return claims.UserID, "", nil
	}
	resolve := func(username string) (int, error) {
		row, err := store.GetUserByUsername(username)
		if err != nil {
			return 0, err
		}
		var id int
		var passwordHash string
		if err := row.Scan(&id, &passwordHash); err != nil {
			return 0, err
		}
		return id, nil
	}
	mux.HandleFunc("/ws", ws.WebSocketHandler(hub, authenticator, resolve))

	// Serve static files
	mux.Handle("/", http.FileServer(http.Dir("../client/public")))

	return mux
}

func authMiddleware(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		authHeader := r.Header.Get("Authorization")
		if authHeader == "" {
			http.Error(w, "Authorization header required", http.StatusUnauthorized)
			return
		}

		tokenString := strings.TrimPrefix(authHeader, "Bearer ")
		claims, err := auth.ValidateToken(tokenString)
		if err != nil {
			http.Error(w, "Invalid token", http.StatusUnauthorized)
			return
		}

		ctx := context.WithValue(r.Context(), "userID", claims.UserID)
		next.ServeHTTP(w, r.WithContext(ctx))
	}
}