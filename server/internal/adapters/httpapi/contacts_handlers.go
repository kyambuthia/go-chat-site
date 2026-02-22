package httpapi

import (
	"encoding/json"
	"errors"
	"net/http"

	"github.com/kyambuthia/go-chat-site/server/internal/auth"
	"github.com/kyambuthia/go-chat-site/server/internal/store"
	"github.com/kyambuthia/go-chat-site/server/internal/web"
)

type ContactsHandler struct {
	Store store.ContactsStore
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

	contacts, err := h.Store.ListContacts(userID)
	if err != nil {
		web.JSONError(w, err, http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(contacts)
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

	user, err := h.Store.GetUserByUsername(req.Username)
	if err != nil {
		web.JSONError(w, errors.New("user not found"), http.StatusNotFound)
		return
	}

	if err := h.Store.AddContact(userID, user.ID); err != nil {
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

	if err := h.Store.RemoveContact(userID, req.ContactID); err != nil {
		web.JSONError(w, err, http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
}
