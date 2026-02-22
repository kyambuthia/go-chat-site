package httpapi

import (
	"encoding/json"
	"errors"
	"net/http"

	"github.com/kyambuthia/go-chat-site/server/internal/auth"
	"github.com/kyambuthia/go-chat-site/server/internal/store"
	"github.com/kyambuthia/go-chat-site/server/internal/web"
)

type WalletHandler struct {
	Store store.WalletStore
}

func (h *WalletHandler) GetWallet(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		web.JSONError(w, errors.New("method not allowed"), http.StatusMethodNotAllowed)
		return
	}

	userID, ok := auth.UserIDFromContext(r.Context())
	if !ok {
		web.JSONError(w, errors.New("unauthorized"), http.StatusUnauthorized)
		return
	}

	wallet, err := h.Store.GetWallet(userID)
	if err != nil {
		web.JSONError(w, err, http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]interface{}{
		"id":            wallet.ID,
		"user_id":       wallet.UserID,
		"balance":       wallet.BalanceFloat(),
		"balance_cents": wallet.BalanceCents,
	})
}

func (h *WalletHandler) SendMoney(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		web.JSONError(w, errors.New("method not allowed"), http.StatusMethodNotAllowed)
		return
	}

	var req struct {
		Username string  `json:"username"`
		Amount   float64 `json:"amount"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		web.JSONError(w, errors.New("invalid request body"), http.StatusBadRequest)
		return
	}

	senderID, ok := auth.UserIDFromContext(r.Context())
	if !ok {
		web.JSONError(w, errors.New("unauthorized"), http.StatusUnauthorized)
		return
	}

	user, err := h.Store.GetUserByUsername(req.Username)
	if err != nil {
		web.JSONError(w, errors.New("user not found"), http.StatusNotFound)
		return
	}

	amountCents, err := store.DollarsToCents(req.Amount)
	if err != nil {
		web.JSONError(w, err, http.StatusBadRequest)
		return
	}

	if err := h.Store.SendMoney(senderID, user.ID, amountCents); err != nil {
		if errors.Is(err, store.ErrInsufficientFund) {
			web.JSONError(w, err, http.StatusBadRequest)
			return
		}
		web.JSONError(w, err, http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
}
