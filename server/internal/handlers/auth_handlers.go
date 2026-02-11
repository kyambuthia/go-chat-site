package handlers

import (
	"encoding/json"
	"errors"
	"net/http"
	"strings"

	"github.com/kyambuthia/go-chat-site/server/internal/store"
	"github.com/kyambuthia/go-chat-site/server/internal/web"
)

type AuthHandler struct {
	Store store.AuthStore
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

	id, err := h.Store.CreateUser(creds.Username, creds.Password)
	if err != nil {
		web.JSONError(w, err, http.StatusConflict)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	_ = json.NewEncoder(w).Encode(map[string]interface{}{
		"id":       id,
		"username": creds.Username,
	})
}
