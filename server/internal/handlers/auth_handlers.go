// ISSUE: The error handling in the Register function is not descriptive.
// It returns generic HTTP errors, masking the underlying cause (e.g., database errors),
// which makes debugging difficult.

package handlers

import (
	"encoding/json"
	"errors"
	"net/http"

	"github.com/kyambuthia/go-chat-site/server/internal/store"
	"github.com/kyambuthia/go-chat-site/server/internal/web"
)

type AuthHandler struct {
	Store *store.SqliteStore
}

func (h *AuthHandler) Register(w http.ResponseWriter, r *http.Request) {
	var creds struct {
		Username string `json:"username"`
		Password string `json:"password"`
	}

	if err := json.NewDecoder(r.Body).Decode(&creds); err != nil {
		web.JSONError(w, err, http.StatusBadRequest)
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

	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(map[string]interface{}{
		"id":       id,
		"username": creds.Username,
	})
}