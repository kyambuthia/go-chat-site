package auth

import (
	"encoding/json"
	"errors"
	"log"
	"net/http"

	"github.com/kyambuthia/go-chat-site/server/internal/crypto"
	"github.com/kyambuthia/go-chat-site/server/internal/store"
	"github.com/kyambuthia/go-chat-site/server/internal/web"
)

func Login(store *store.SqliteStore) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var creds struct {
			Username string `json:"username"`
			Password string `json:"password"`
		}

		if err := json.NewDecoder(r.Body).Decode(&creds); err != nil {
			web.JSONError(w, errors.New("invalid request body"), http.StatusBadRequest)
			return
		}

		user, err := store.GetUserByUsername(creds.Username)
		if err != nil {
			web.JSONError(w, errors.New("invalid username or password"), http.StatusUnauthorized)
			return
		}

		log.Printf("Attempting login for user '%s'. Hash from DB: '%s'", user.Username, user.PasswordHash)

		if !crypto.CheckPasswordHash(creds.Password, user.PasswordHash) {
			log.Printf("Password check failed for user '%s'", user.Username)
			web.JSONError(w, errors.New("invalid username or password"), http.StatusUnauthorized)
			return
		}

		token, err := GenerateToken(user.ID)
		if err != nil {
			web.JSONError(w, err, http.StatusInternalServerError)
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