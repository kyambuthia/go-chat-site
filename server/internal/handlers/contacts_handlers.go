package handlers

import (
	"encoding/json"
	"net/http"

	"github.com/kyambuthia/go-chat-site/server/internal/store"
)

type ContactsHandler struct {
	Store *store.SqliteStore
}

func (h *ContactsHandler) GetContacts(w http.ResponseWriter, r *http.Request) {
	userID, ok := r.Context().Value("userID").(int)
	if !ok {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	rows, err := h.Store.GetContacts(userID)
	if err != nil {
		http.Error(w, "Failed to get contacts", http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	var contacts []store.User
	for rows.Next() {
		var contact store.User
		if err := rows.Scan(&contact.ID, &contact.Username, &contact.DisplayName, &contact.AvatarURL); err != nil {
			http.Error(w, "Failed to scan contact", http.StatusInternalServerError)
			return
		}
		contacts = append(contacts, contact)
	}

	json.NewEncoder(w).Encode(contacts)
}

func (h *ContactsHandler) AddContact(w http.ResponseWriter, r *http.Request) {
	userID := r.Context().Value("userID").(int)

	var req struct {
		Username string `json:"username"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	user, err := h.Store.GetUserByUsername(req.Username)
	if err != nil {
		http.Error(w, "User not found", http.StatusNotFound)
		return
	}

	if err := h.Store.AddContact(userID, user.ID); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusCreated)
}

func (h *ContactsHandler) RemoveContact(w http.ResponseWriter, r *http.Request) {
	userID := r.Context().Value("userID").(int)

	var req struct {
		ContactID int `json:"contact_id"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	if err := h.Store.RemoveContact(userID, req.ContactID); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
}
