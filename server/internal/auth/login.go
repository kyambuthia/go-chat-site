package auth

import (
	"encoding/json"
	"errors"
	"net/http"
	"strings"

	"github.com/kyambuthia/go-chat-site/server/internal/crypto"
	"github.com/kyambuthia/go-chat-site/server/internal/store"
	"github.com/kyambuthia/go-chat-site/server/internal/web"
)

// Login is a legacy compatibility handler retained during the adapter refactor.
// Deprecated: prefer server/internal/adapters/httpapi.AuthHandler with core identity services.
func Login(authStore store.LoginStore) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			web.JSONError(w, errors.New("method not allowed"), http.StatusMethodNotAllowed)
			return
		}

		var creds struct {
			Username string `json:"username"`
			Password string `json:"password"`
		}

		if err := json.NewDecoder(r.Body).Decode(&creds); err != nil {
			web.JSONError(w, errors.New("invalid request body"), http.StatusBadRequest)
			return
		}

		user, err := authStore.GetUserByUsername(strings.TrimSpace(creds.Username))
		if err != nil {
			web.JSONError(w, errors.New("invalid username or password"), http.StatusUnauthorized)
			return
		}

		if !crypto.CheckPasswordHash(creds.Password, user.PasswordHash) {
			web.JSONError(w, errors.New("invalid username or password"), http.StatusUnauthorized)
			return
		}

		token, err := GenerateToken(user.ID)
		if err != nil {
			web.JSONError(w, err, http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(struct {
			Token string `json:"token"`
		}{
			Token: token,
		})
	}
}
