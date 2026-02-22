package httpapi

import (
	"encoding/json"
	"errors"
	"net/http"

	"github.com/kyambuthia/go-chat-site/server/internal/auth"
	corecontacts "github.com/kyambuthia/go-chat-site/server/internal/core/contacts"
	"github.com/kyambuthia/go-chat-site/server/internal/web"
)

type ContactsHandler struct {
	Contacts corecontacts.Service
}

func (h *ContactsHandler) GetContacts(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		web.JSONError(w, errors.New("method not allowed"), http.StatusMethodNotAllowed)
		return
	}

	userID, ok := auth.UserIDFromContext(r.Context())
	if !ok {
		web.JSONError(w, errors.New("unauthorized"), http.StatusUnauthorized)
		return
	}

	contacts, err := h.Contacts.ListContacts(r.Context(), corecontacts.UserID(userID))
	if err != nil {
		web.JSONError(w, err, http.StatusInternalServerError)
		return
	}

	resp := make([]map[string]any, 0, len(contacts))
	for _, c := range contacts {
		resp = append(resp, map[string]any{
			"id":           int(c.UserID),
			"username":     c.Username,
			"display_name": c.DisplayName,
			"avatar_url":   c.AvatarURL,
		})
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(resp)
}

func (h *ContactsHandler) AddContact(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		web.JSONError(w, errors.New("method not allowed"), http.StatusMethodNotAllowed)
		return
	}

	userID, ok := auth.UserIDFromContext(r.Context())
	if !ok {
		web.JSONError(w, errors.New("unauthorized"), http.StatusUnauthorized)
		return
	}

	var req struct {
		Username string `json:"username"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		web.JSONError(w, errors.New("invalid request body"), http.StatusBadRequest)
		return
	}

	if err := h.Contacts.AddContactByUsername(r.Context(), corecontacts.UserID(userID), req.Username); err != nil {
		if errors.Is(err, corecontacts.ErrUserNotFound) {
			web.JSONError(w, errors.New("user not found"), http.StatusNotFound)
			return
		}
		web.JSONError(w, err, http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusCreated)
}

func (h *ContactsHandler) RemoveContact(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodDelete {
		web.JSONError(w, errors.New("method not allowed"), http.StatusMethodNotAllowed)
		return
	}

	userID, ok := auth.UserIDFromContext(r.Context())
	if !ok {
		web.JSONError(w, errors.New("unauthorized"), http.StatusUnauthorized)
		return
	}

	var req struct {
		ContactID int `json:"contact_id"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		web.JSONError(w, errors.New("invalid request body"), http.StatusBadRequest)
		return
	}

	if err := h.Contacts.RemoveContact(r.Context(), corecontacts.UserID(userID), corecontacts.UserID(req.ContactID)); err != nil {
		web.JSONError(w, err, http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
}
