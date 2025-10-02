package api

import (
	"net/http"

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

	return mux
}
