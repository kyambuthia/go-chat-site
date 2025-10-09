// ISSUE: The error handling in the Login function is not descriptive.
// It returns generic HTTP errors, masking the underlying cause (e.g., database errors or token generation failures),
// which makes debugging the 500 error difficult.

package auth

import (
	"encoding/json"
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
			web.JSONError(w, err, http.StatusBadRequest)
			return
		}

		user, err := store.GetUserByUsername(creds.Username)
		if err != nil {
			web.JSONError(w, err, http.StatusUnauthorized)
			return
		}

		if !crypto.CheckPasswordHash(creds.Password, user.PasswordHash) {
			web.JSONError(w, err, http.StatusUnauthorized)
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