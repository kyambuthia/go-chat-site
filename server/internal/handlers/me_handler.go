package handlers

import (
	"encoding/json"
	"net/http"

	"github.com/kyambuthia/go-chat-site/server/internal/store"
	"github.com/kyambuthia/go-chat-site/server/internal/web"
)

type MeHandler struct {
	Store *store.SqliteStore
}

func (h *MeHandler) GetMe(w http.ResponseWriter, r *http.Request) {
	userID := r.Context().Value("userID").(int)

	user, err := h.Store.GetUserByID(userID)
	if err != nil {
		web.JSONError(w, err, http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(user)
}
