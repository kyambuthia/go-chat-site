package handlers

import (
	"encoding/json"
	"net/http"

	"github.com/kyambuthia/go-chat-site/server/internal/store"
)

type InviteHandler struct {
	Store *store.SqliteStore
}

func (h *InviteHandler) SendInvite(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Username string `json:"username"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	inviterID := r.Context().Value("userID").(int)

	user, err := h.Store.GetUserByUsername(req.Username)
	if err != nil {
		http.Error(w, "User not found", http.StatusNotFound)
		return
	}

	if err := h.Store.CreateInvite(inviterID, user.ID); err != nil {
		http.Error(w, "Failed to send invite", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusCreated)
}

func (h *InviteHandler) GetInvites(w http.ResponseWriter, r *http.Request) {
	userID := r.Context().Value("userID").(int)

	rows, err := h.Store.GetInvites(userID)
	if err != nil {
		http.Error(w, "Failed to get invites", http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	var invites []struct {
		ID               int    `json:"id"`
		InviterUsername string `json:"inviter_username"`
	}

	for rows.Next() {
		var invite struct {
			ID               int    `json:"id"`
			InviterUsername string `json:"inviter_username"`
		}
		if err := rows.Scan(&invite.ID, &invite.InviterUsername); err != nil {
			http.Error(w, "Failed to scan invite", http.StatusInternalServerError)
			return
		}
		invites = append(invites, invite)
	}

	json.NewEncoder(w).Encode(invites)
}

func (h *InviteHandler) AcceptInvite(w http.ResponseWriter, r *http.Request) {
	var req struct {
		InviteID int `json:"invite_id"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	userID := r.Context().Value("userID").(int)

	if err := h.Store.UpdateInviteStatus(req.InviteID, userID, "accepted"); err != nil {
		http.Error(w, "Failed to accept invite", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
}

func (h *InviteHandler) RejectInvite(w http.ResponseWriter, r *http.Request) {
	var req struct {
		InviteID int `json:"invite_id"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	userID := r.Context().Value("userID").(int)

	if err := h.Store.UpdateInviteStatus(req.InviteID, userID, "rejected"); err != nil {
		http.Error(w, "Failed to reject invite", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
}
