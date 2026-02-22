package httpapi

import (
	"encoding/json"
	"errors"
	"net/http"
	"strings"

	coreid "github.com/kyambuthia/go-chat-site/server/internal/core/identity"
	"github.com/kyambuthia/go-chat-site/server/internal/web"
)

type AuthHandler struct {
	Identity coreid.AuthService
}

func (h *AuthHandler) Register(w http.ResponseWriter, r *http.Request) {
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

	creds.Username = strings.TrimSpace(creds.Username)
	if creds.Username == "" {
		web.JSONError(w, errors.New("username is required"), http.StatusBadRequest)
		return
	}

	if len(creds.Password) < 8 {
		web.JSONError(w, errors.New("password too short"), http.StatusBadRequest)
		return
	}

	principal, err := h.Identity.RegisterPassword(r.Context(), coreid.PasswordCredential{
		Username: creds.Username,
		Password: creds.Password,
	})
	if err != nil {
		web.JSONError(w, err, http.StatusConflict)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	_ = json.NewEncoder(w).Encode(map[string]interface{}{
		"id":       int(principal.ID),
		"username": creds.Username,
	})
}

func (h *AuthHandler) Login(w http.ResponseWriter, r *http.Request) {
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

	token, err := h.Identity.LoginPassword(r.Context(), coreid.PasswordCredential{
		Username: strings.TrimSpace(creds.Username),
		Password: creds.Password,
	})
	if err != nil {
		if errors.Is(err, coreid.ErrInvalidCredentials) {
			web.JSONError(w, errors.New("invalid username or password"), http.StatusUnauthorized)
			return
		}
		web.JSONError(w, err, http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(struct {
		Token string `json:"token"`
	}{Token: token})
}
