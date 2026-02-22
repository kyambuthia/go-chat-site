package httpapi

import (
	"encoding/json"
	"errors"
	"net/http"

	"github.com/kyambuthia/go-chat-site/server/internal/auth"
	coreid "github.com/kyambuthia/go-chat-site/server/internal/core/identity"
	"github.com/kyambuthia/go-chat-site/server/internal/web"
)

type MeHandler struct {
	Identity coreid.ProfileService
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

	profile, err := h.Identity.GetProfile(r.Context(), coreid.UserID(userID))
	if err != nil {
		web.JSONError(w, errors.New("user not found"), http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]any{
		"id":           int(profile.UserID),
		"username":     profile.Username,
		"display_name": profile.DisplayName,
		"avatar_url":   profile.AvatarURL,
	})
}
