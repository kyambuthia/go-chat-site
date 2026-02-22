package httpapi

import (
	"encoding/json"
	"errors"
	"net/http"

	"github.com/kyambuthia/go-chat-site/server/internal/auth"
	"github.com/kyambuthia/go-chat-site/server/internal/store"
	"github.com/kyambuthia/go-chat-site/server/internal/web"
)

type MeHandler struct {
	Store store.MeStore
}

func (h *MeHandler) GetMe(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		web.JSONError(w, errors.New("method not allowed"), http.StatusMethodNotAllowed)
		return
	}

	userID, ok := auth.UserIDFromContext(r.Context())
	if !ok {
		web.JSONError(w, errors.New("unauthorized"), http.StatusUnauthorized)
		return
	}

	user, err := h.Store.GetUserByID(userID)
	if err != nil {
		web.JSONError(w, errors.New("user not found"), http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(user)
}
