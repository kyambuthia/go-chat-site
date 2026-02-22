package httpapi

import (
	"encoding/json"
	"errors"
	"net/http"
	"strconv"

	"github.com/kyambuthia/go-chat-site/server/internal/auth"
	coremsg "github.com/kyambuthia/go-chat-site/server/internal/core/messaging"
	"github.com/kyambuthia/go-chat-site/server/internal/web"
)

type MessagesHandler struct {
	Messaging coremsg.PersistenceService
}

func (h *MessagesHandler) GetInbox(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		web.JSONError(w, errors.New("method not allowed"), http.StatusMethodNotAllowed)
		return
	}

	userID, ok := auth.UserIDFromContext(r.Context())
	if !ok {
		web.JSONError(w, errors.New("unauthorized"), http.StatusUnauthorized)
		return
	}

	limit := 0
	beforeID := int64(0)
	afterID := int64(0)
	if raw := r.URL.Query().Get("limit"); raw != "" {
		n, err := strconv.Atoi(raw)
		if err != nil || n <= 0 {
			web.JSONError(w, errors.New("invalid limit"), http.StatusBadRequest)
			return
		}
		limit = n
	}
	if raw := r.URL.Query().Get("before_id"); raw != "" {
		n, err := strconv.ParseInt(raw, 10, 64)
		if err != nil || n <= 0 {
			web.JSONError(w, errors.New("invalid before_id"), http.StatusBadRequest)
			return
		}
		beforeID = n
	}
	if raw := r.URL.Query().Get("after_id"); raw != "" {
		n, err := strconv.ParseInt(raw, 10, 64)
		if err != nil || n <= 0 {
			web.JSONError(w, errors.New("invalid after_id"), http.StatusBadRequest)
			return
		}
		afterID = n
	}
	if beforeID > 0 && afterID > 0 {
		web.JSONError(w, errors.New("before_id and after_id cannot be combined"), http.StatusBadRequest)
		return
	}

	if h.Messaging == nil {
		web.JSONError(w, errors.New("messaging sync unavailable"), http.StatusServiceUnavailable)
		return
	}

	var inbox []coremsg.StoredMessage
	var err error
	if beforeID > 0 {
		inbox, err = h.Messaging.ListInboxBefore(r.Context(), userID, beforeID, limit)
	} else if afterID > 0 {
		inbox, err = h.Messaging.ListInboxAfter(r.Context(), userID, afterID, limit)
	} else {
		inbox, err = h.Messaging.ListInbox(r.Context(), userID, limit)
	}
	if err != nil {
		web.JSONError(w, err, http.StatusInternalServerError)
		return
	}

	resp := make([]map[string]any, 0, len(inbox))
	for _, msg := range inbox {
		item := map[string]any{
			"id":           msg.ID,
			"from_user_id": msg.FromUserID,
			"to_user_id":   msg.ToUserID,
			"body":         msg.Body,
			"created_at":   msg.CreatedAt,
		}
		if msg.DeliveredAt != nil {
			item["delivered_at"] = *msg.DeliveredAt
		}
		if msg.ReadAt != nil {
			item["read_at"] = *msg.ReadAt
		}
		resp = append(resp, item)
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(resp)
}

func (h *MessagesHandler) MarkRead(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		web.JSONError(w, errors.New("method not allowed"), http.StatusMethodNotAllowed)
		return
	}

	userID, ok := auth.UserIDFromContext(r.Context())
	if !ok {
		web.JSONError(w, errors.New("unauthorized"), http.StatusUnauthorized)
		return
	}

	if h.Messaging == nil {
		web.JSONError(w, errors.New("messaging sync unavailable"), http.StatusServiceUnavailable)
		return
	}

	var req struct {
		MessageID int64 `json:"message_id"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		web.JSONError(w, errors.New("invalid request body"), http.StatusBadRequest)
		return
	}
	if req.MessageID <= 0 {
		web.JSONError(w, errors.New("invalid request body"), http.StatusBadRequest)
		return
	}

	if err := h.Messaging.MarkReadForRecipient(r.Context(), userID, req.MessageID); err != nil {
		if errors.Is(err, coremsg.ErrMessageNotFound) {
			web.JSONError(w, errors.New("message not found"), http.StatusNotFound)
			return
		}
		web.JSONError(w, err, http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusOK)
}

func (h *MessagesHandler) MarkDelivered(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		web.JSONError(w, errors.New("method not allowed"), http.StatusMethodNotAllowed)
		return
	}

	userID, ok := auth.UserIDFromContext(r.Context())
	if !ok {
		web.JSONError(w, errors.New("unauthorized"), http.StatusUnauthorized)
		return
	}

	if h.Messaging == nil {
		web.JSONError(w, errors.New("messaging sync unavailable"), http.StatusServiceUnavailable)
		return
	}

	var req struct {
		MessageID int64 `json:"message_id"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		web.JSONError(w, errors.New("invalid request body"), http.StatusBadRequest)
		return
	}
	if req.MessageID <= 0 {
		web.JSONError(w, errors.New("invalid request body"), http.StatusBadRequest)
		return
	}

	if err := h.Messaging.MarkDeliveredForRecipient(r.Context(), userID, req.MessageID); err != nil {
		if errors.Is(err, coremsg.ErrMessageNotFound) {
			web.JSONError(w, errors.New("message not found"), http.StatusNotFound)
			return
		}
		web.JSONError(w, err, http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusOK)
}
