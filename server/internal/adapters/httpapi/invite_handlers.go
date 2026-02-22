package httpapi

import (
	"encoding/json"
	"errors"
	"net/http"

	"github.com/kyambuthia/go-chat-site/server/internal/auth"
	"github.com/kyambuthia/go-chat-site/server/internal/store"
	"github.com/kyambuthia/go-chat-site/server/internal/web"
)

type InviteHandler struct {
	Store store.InviteStore
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

	user, err := h.Store.GetUserByUsername(req.Username)
	if err != nil {
		web.JSONError(w, errors.New("user not found"), http.StatusNotFound)
		return
	}

	if err := h.Store.CreateInvite(inviterID, user.ID); err != nil {
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

	invites, err := h.Store.ListInvites(userID)
	if err != nil {
		web.JSONError(w, err, http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(invites)
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

	if err := h.Store.UpdateInviteStatus(req.InviteID, userID, status); err != nil {
		if errors.Is(err, store.ErrNotFound) {
			web.JSONError(w, errors.New("invite not found"), http.StatusNotFound)
			return
		}
		web.JSONError(w, err, http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
}
