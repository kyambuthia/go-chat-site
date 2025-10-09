package auth

import (
	"encoding/json"
	"net/http"

	"github.com/kyambuthia/go-chat-site/server/internal/api"
	"github.com/kyambuthia/go-chat-site/server/internal/crypto"
	"github.com/kyambuthia/go-chat-site/server/internal/store"
)

func Login(store *store.SqliteStore) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var creds struct {
			Username string `json:"username"`
			Password string `json:"password"`
		}

		if err := json.NewDecoder(r.Body).Decode(&creds); err != nil {
			api.JSONError(w, err, http.StatusBadRequest)
			return
		}

		user, err := store.GetUserByUsername(creds.Username)
		if err != nil {
			api.JSONError(w, err, http.StatusUnauthorized)
			return
		}

		if !crypto.CheckPasswordHash(creds.Password, user.PasswordHash) {
			api.JSONError(w, err, http.StatusUnauthorized)
			return
		}

		token, err := GenerateToken(user.ID)
		if err != nil {
			api.JSONError(w, err, http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(struct {
			Token string `json:"token"`
		}{
			Token: token,
		})
	}
}
