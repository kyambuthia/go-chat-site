package httpapi

import (
	"encoding/json"
	"errors"
	"net/http"

	"github.com/kyambuthia/go-chat-site/server/internal/auth"
	corecontacts "github.com/kyambuthia/go-chat-site/server/internal/core/contacts"
	"github.com/kyambuthia/go-chat-site/server/internal/store"
	"github.com/kyambuthia/go-chat-site/server/internal/web"
)

type InviteHandler struct {
	Contacts corecontacts.Service
}

func (h *InviteHandler) SendInvite(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		web.JSONError(w, errors.New("method not allowed"), http.StatusMethodNotAllowed)
		return
	}

	var req struct {
		Username string `json:"username"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		web.JSONError(w, errors.New("invalid request body"), http.StatusBadRequest)
		return
	}

	inviterID, ok := auth.UserIDFromContext(r.Context())
	if !ok {
		web.JSONError(w, errors.New("unauthorized"), http.StatusUnauthorized)
		return
	}

	if err := h.Contacts.SendInviteByUsername(r.Context(), corecontacts.UserID(inviterID), req.Username); err != nil {
		if errors.Is(err, corecontacts.ErrUserNotFound) {
			web.JSONError(w, errors.New("user not found"), http.StatusNotFound)
			return
		}
		if errors.Is(err, store.ErrInviteExists) {
			web.JSONError(w, err, http.StatusConflict)
			return
		}
		web.JSONError(w, err, http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusCreated)
}

func (h *InviteHandler) GetInvites(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		web.JSONError(w, errors.New("method not allowed"), http.StatusMethodNotAllowed)
		return
	}

	userID, ok := auth.UserIDFromContext(r.Context())
	if !ok {
		web.JSONError(w, errors.New("unauthorized"), http.StatusUnauthorized)
		return
	}

	invites, err := h.Contacts.ListInvites(r.Context(), corecontacts.UserID(userID))
	if err != nil {
		web.JSONError(w, err, http.StatusInternalServerError)
		return
	}

	resp := make([]map[string]any, 0, len(invites))
	for _, inv := range invites {
		resp = append(resp, map[string]any{
			"id":               inv.ID,
			"inviter_username": inv.InviterUsername,
		})
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(resp)
}

func (h *InviteHandler) AcceptInvite(w http.ResponseWriter, r *http.Request) {
	h.updateInviteStatus(w, r, "accepted")
}

func (h *InviteHandler) RejectInvite(w http.ResponseWriter, r *http.Request) {
	h.updateInviteStatus(w, r, "rejected")
}

func (h *InviteHandler) updateInviteStatus(w http.ResponseWriter, r *http.Request, status string) {
	if r.Method != http.MethodPost {
		web.JSONError(w, errors.New("method not allowed"), http.StatusMethodNotAllowed)
		return
	}

	var req struct {
		InviteID int `json:"invite_id"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		web.JSONError(w, errors.New("invalid request body"), http.StatusBadRequest)
		return
	}

	userID, ok := auth.UserIDFromContext(r.Context())
	if !ok {
		web.JSONError(w, errors.New("unauthorized"), http.StatusUnauthorized)
		return
	}

	if err := h.Contacts.RespondToInvite(r.Context(), req.InviteID, corecontacts.UserID(userID), corecontacts.InviteStatus(status)); err != nil {
		if errors.Is(err, corecontacts.ErrInviteNotFound) || errors.Is(err, store.ErrNotFound) {
			web.JSONError(w, errors.New("invite not found"), http.StatusNotFound)
			return
		}
		web.JSONError(w, err, http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
}
