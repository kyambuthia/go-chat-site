package api

import (
	"net/http"

	"github.com/kyambuthia/go-chat-site/server/internal/handlers"
	"github.com/kyambuthia/go-chat-site/server/internal/store"
)

func NewAPI(store *store.SqliteStore) http.Handler {
	mux := http.NewServeMux()

	authHandler := &handlers.AuthHandler{Store: store}
	mux.HandleFunc("/api/register", authHandler.Register)
	mux.HandleFunc("/api/login", authHandler.Login)

	return mux
}
