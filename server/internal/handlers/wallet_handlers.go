package handlers

import (
	"encoding/json"
	"errors"
	"net/http"

	"github.com/kyambuthia/go-chat-site/server/internal/store"
	"github.com/kyambuthia/go-chat-site/server/internal/web"
)

type WalletHandler struct {
	Store *store.SqliteStore
}

func (h *WalletHandler) GetWallet(w http.ResponseWriter, r *http.Request) {
	userID := r.Context().Value("userID").(int)

	wallet, err := h.Store.GetWallet(userID)
	if err != nil {
		web.JSONError(w, err, http.StatusInternalServerError)
		return
	}

	json.NewEncoder(w).Encode(wallet)
}

func (h *WalletHandler) SendMoney(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Username string  `json:"username"`
		Amount   float64 `json:"amount"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		web.JSONError(w, errors.New("invalid request body"), http.StatusBadRequest)
		return
	}

	senderID := r.Context().Value("userID").(int)

	user, err := h.Store.GetUserByUsername(req.Username)
	if err != nil {
		web.JSONError(w, errors.New("user not found"), http.StatusNotFound)
		return
	}

	if err := h.Store.SendMoney(senderID, user.ID, req.Amount); err != nil {
		web.JSONError(w, err, http.StatusInternalServerError)
		return	}

	w.WriteHeader(http.StatusOK)
}