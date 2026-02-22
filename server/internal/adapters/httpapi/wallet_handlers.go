package httpapi

import (
	"encoding/json"
	"errors"
	"net/http"
	"strconv"

	"github.com/kyambuthia/go-chat-site/server/internal/auth"
	coreledger "github.com/kyambuthia/go-chat-site/server/internal/core/ledger"
	"github.com/kyambuthia/go-chat-site/server/internal/store"
	"github.com/kyambuthia/go-chat-site/server/internal/web"
)

type WalletHandler struct {
	Ledger coreledger.Service
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

	account, err := h.Ledger.GetAccount(r.Context(), userID)
	if err != nil {
		web.JSONError(w, err, http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	idValue := any(account.ID)
	if legacyID, err := strconv.Atoi(string(account.ID)); err == nil {
		idValue = legacyID
	}
	_ = json.NewEncoder(w).Encode(map[string]interface{}{
		"id":            idValue,
		"user_id":       account.OwnerUserID,
		"balance":       float64(account.BalanceCents) / 100.0,
		"balance_cents": account.BalanceCents,
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

	amountCents, err := store.DollarsToCents(req.Amount)
	if err != nil {
		web.JSONError(w, err, http.StatusBadRequest)
		return
	}

	if _, err := h.Ledger.SendTransferByUsername(r.Context(), senderID, req.Username, amountCents); err != nil {
		if errors.Is(err, coreledger.ErrRecipientNotFound) {
			web.JSONError(w, errors.New("user not found"), http.StatusNotFound)
			return
		}
		if errors.Is(err, store.ErrInsufficientFund) {
			web.JSONError(w, err, http.StatusBadRequest)
			return
		}
		web.JSONError(w, err, http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
}
