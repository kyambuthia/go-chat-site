package auth

import (
	"encoding/json"
	"log"
	"net/http"

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
			http.Error(w, "Invalid request body", http.StatusBadRequest)
			return
		}

		user, err := store.GetUserByUsername(creds.Username)
		if err != nil {
			http.Error(w, "Invalid username or password", http.StatusUnauthorized)
			return
		}

		if !crypto.CheckPasswordHash(creds.Password, user.PasswordHash) {
			http.Error(w, "Invalid username or password", http.StatusUnauthorized)
			return
		}

		token, err := GenerateToken(user.ID)
		if err != nil {
			log.Printf("Failed to generate token: %v", err)
			http.Error(w, "Failed to generate token", http.StatusInternalServerError)
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
