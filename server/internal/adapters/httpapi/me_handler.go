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

	writeProfileJSON(w, profile)
}

func (h *MeHandler) UpdateMe(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPatch {
		web.JSONError(w, errors.New("method not allowed"), http.StatusMethodNotAllowed)
		return
	}

	userID, ok := auth.UserIDFromContext(r.Context())
	if !ok {
		web.JSONError(w, errors.New("unauthorized"), http.StatusUnauthorized)
		return
	}

	var req struct {
		DisplayName *string `json:"display_name"`
		AvatarURL   *string `json:"avatar_url"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		web.JSONError(w, errors.New("invalid request body"), http.StatusBadRequest)
		return
	}
	if req.DisplayName == nil && req.AvatarURL == nil {
		web.JSONError(w, errors.New("at least one field is required"), http.StatusBadRequest)
		return
	}

	current, err := h.Identity.GetProfile(r.Context(), coreid.UserID(userID))
	if err != nil {
		web.JSONError(w, errors.New("user not found"), http.StatusNotFound)
		return
	}

	update := coreid.ProfileUpdate{
		DisplayName: current.DisplayName,
		AvatarURL:   current.AvatarURL,
	}
	if req.DisplayName != nil {
		update.DisplayName = *req.DisplayName
	}
	if req.AvatarURL != nil {
		update.AvatarURL = *req.AvatarURL
	}

	profile, err := h.Identity.UpdateProfile(r.Context(), coreid.UserID(userID), update)
	if err != nil {
		web.JSONError(w, errors.New("user not found"), http.StatusNotFound)
		return
	}

	writeProfileJSON(w, profile)
}

func writeProfileJSON(w http.ResponseWriter, profile coreid.Profile) {
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]any{
		"id":           int(profile.UserID),
		"username":     profile.Username,
		"display_name": profile.DisplayName,
		"avatar_url":   profile.AvatarURL,
	})
}
