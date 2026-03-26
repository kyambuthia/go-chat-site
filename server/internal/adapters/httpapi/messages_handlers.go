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
	Messaging        coremsg.PersistenceService
	Threads          coremsg.ThreadSummaryService
	ReceiptTransport coremsg.Transport
}

func (h *MessagesHandler) GetOutbox(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
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

	var outbox []coremsg.StoredMessage
	var err error
	if beforeID > 0 {
		outbox, err = h.Messaging.ListOutboxBefore(r.Context(), userID, beforeID, limit)
	} else if afterID > 0 {
		outbox, err = h.Messaging.ListOutboxAfter(r.Context(), userID, afterID, limit)
	} else {
		outbox, err = h.Messaging.ListOutbox(r.Context(), userID, limit)
	}
	if err != nil {
		web.JSONError(w, err, http.StatusInternalServerError)
		return
	}
	writeStoredMessagesJSON(w, outbox)
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
	withUserID := 0
	unreadOnly := false
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
	if raw := r.URL.Query().Get("with_user_id"); raw != "" {
		n, err := strconv.Atoi(raw)
		if err != nil || n <= 0 {
			web.JSONError(w, errors.New("invalid with_user_id"), http.StatusBadRequest)
			return
		}
		withUserID = n
	}
	if raw := r.URL.Query().Get("unread_only"); raw != "" {
		v, err := strconv.ParseBool(raw)
		if err != nil {
			web.JSONError(w, errors.New("invalid unread_only"), http.StatusBadRequest)
			return
		}
		unreadOnly = v
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
	if beforeID > 0 && withUserID > 0 && unreadOnly {
		inbox, err = h.Messaging.ListUnreadInboxBeforeWithUser(r.Context(), userID, withUserID, beforeID, limit)
	} else if beforeID > 0 && withUserID > 0 {
		inbox, err = h.Messaging.ListInboxBeforeWithUser(r.Context(), userID, withUserID, beforeID, limit)
	} else if beforeID > 0 && unreadOnly {
		inbox, err = h.Messaging.ListUnreadInboxBefore(r.Context(), userID, beforeID, limit)
	} else if beforeID > 0 {
		inbox, err = h.Messaging.ListInboxBefore(r.Context(), userID, beforeID, limit)
	} else if afterID > 0 && withUserID > 0 && unreadOnly {
		inbox, err = h.Messaging.ListUnreadInboxAfterWithUser(r.Context(), userID, withUserID, afterID, limit)
	} else if afterID > 0 && withUserID > 0 {
		inbox, err = h.Messaging.ListInboxAfterWithUser(r.Context(), userID, withUserID, afterID, limit)
	} else if afterID > 0 && unreadOnly {
		inbox, err = h.Messaging.ListUnreadInboxAfter(r.Context(), userID, afterID, limit)
	} else if afterID > 0 {
		inbox, err = h.Messaging.ListInboxAfter(r.Context(), userID, afterID, limit)
	} else if withUserID > 0 && unreadOnly {
		inbox, err = h.Messaging.ListUnreadInboxWithUser(r.Context(), userID, withUserID, limit)
	} else if withUserID > 0 {
		inbox, err = h.Messaging.ListInboxWithUser(r.Context(), userID, withUserID, limit)
	} else if unreadOnly {
		inbox, err = h.Messaging.ListUnreadInbox(r.Context(), userID, limit)
	} else {
		inbox, err = h.Messaging.ListInbox(r.Context(), userID, limit)
	}
	if err != nil {
		web.JSONError(w, err, http.StatusInternalServerError)
		return
	}

	writeStoredMessagesJSON(w, inbox)
}

func (h *MessagesHandler) GetSync(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
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

	limit := 0
	afterID := int64(0)
	if raw := r.URL.Query().Get("limit"); raw != "" {
		n, err := strconv.Atoi(raw)
		if err != nil || n <= 0 {
			web.JSONError(w, errors.New("invalid limit"), http.StatusBadRequest)
			return
		}
		limit = n
	}
	if raw := r.URL.Query().Get("after_id"); raw != "" {
		n, err := strconv.ParseInt(raw, 10, 64)
		if err != nil || n < 0 {
			web.JSONError(w, errors.New("invalid after_id"), http.StatusBadRequest)
			return
		}
		afterID = n
	}

	result, err := coremsg.NewSyncService(h.Messaging).Sync(r.Context(), userID, afterID, limit)
	if err != nil {
		web.JSONError(w, err, http.StatusInternalServerError)
		return
	}

	resp := map[string]any{
		"cursor": map[string]any{
			"after_id":      result.Cursor.AfterID,
			"next_after_id": result.Cursor.NextAfterID,
		},
		"messages": storedMessagesToJSON(result.Messages),
		"has_more": result.HasMore,
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(resp)
}

func (h *MessagesHandler) GetThreads(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		web.JSONError(w, errors.New("method not allowed"), http.StatusMethodNotAllowed)
		return
	}

	userID, ok := auth.UserIDFromContext(r.Context())
	if !ok {
		web.JSONError(w, errors.New("unauthorized"), http.StatusUnauthorized)
		return
	}
	if h.Threads == nil {
		web.JSONError(w, errors.New("messaging threads unavailable"), http.StatusServiceUnavailable)
		return
	}

	limit := 0
	if raw := r.URL.Query().Get("limit"); raw != "" {
		n, err := strconv.Atoi(raw)
		if err != nil || n <= 0 {
			web.JSONError(w, errors.New("invalid limit"), http.StatusBadRequest)
			return
		}
		limit = n
	}

	summaries, err := h.Threads.ListThreadSummaries(r.Context(), userID, limit)
	if err != nil {
		web.JSONError(w, err, http.StatusInternalServerError)
		return
	}

	resp := make([]map[string]any, 0, len(summaries))
	for _, summary := range summaries {
		item := map[string]any{
			"user_id":      summary.CounterpartyUserID,
			"username":     summary.CounterpartyUsername,
			"display_name": summary.CounterpartyDisplayName,
			"avatar_url":   summary.CounterpartyAvatarURL,
			"unread_count": summary.UnreadCount,
			"last_message": map[string]any{
				"id":           summary.LastMessageID,
				"from_user_id": summary.LastMessageFromUserID,
				"to_user_id":   summary.LastMessageToUserID,
				"body":         summary.LastMessageBody,
				"created_at":   summary.LastMessageCreatedAt,
			},
		}
		if summary.LastDeliveredAt != nil {
			item["last_message"].(map[string]any)["delivered_at"] = *summary.LastDeliveredAt
		}
		if summary.LastReadAt != nil {
			item["last_message"].(map[string]any)["read_at"] = *summary.LastReadAt
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

	msg, err := h.Messaging.GetMessageForRecipient(r.Context(), userID, req.MessageID)
	if err != nil {
		if errors.Is(err, coremsg.ErrMessageNotFound) {
			web.JSONError(w, errors.New("message not found"), http.StatusNotFound)
			return
		}
		web.JSONError(w, err, http.StatusInternalServerError)
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
	h.tryPushReceipt(msg.FromUserID, coremsg.KindMessageRead, req.MessageID)
	w.WriteHeader(http.StatusOK)
}

func (h *MessagesHandler) MarkThreadRead(w http.ResponseWriter, r *http.Request) {
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
		WithUserID int `json:"with_user_id"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		web.JSONError(w, errors.New("invalid request body"), http.StatusBadRequest)
		return
	}
	if req.WithUserID <= 0 {
		web.JSONError(w, errors.New("invalid request body"), http.StatusBadRequest)
		return
	}

	const pageLimit = 200
	beforeID := int64(0)
	unreadMessages := make([]coremsg.StoredMessage, 0)

	for {
		var page []coremsg.StoredMessage
		var err error
		if beforeID > 0 {
			page, err = h.Messaging.ListInboxBeforeWithUser(r.Context(), userID, req.WithUserID, beforeID, pageLimit)
		} else {
			page, err = h.Messaging.ListInboxWithUser(r.Context(), userID, req.WithUserID, pageLimit)
		}
		if err != nil {
			web.JSONError(w, err, http.StatusInternalServerError)
			return
		}
		if len(page) == 0 {
			break
		}

		for _, msg := range page {
			if msg.ReadAt == nil {
				unreadMessages = append(unreadMessages, msg)
			}
		}

		if len(page) < pageLimit {
			break
		}
		beforeID = page[len(page)-1].ID
	}

	for _, msg := range unreadMessages {
		if err := h.Messaging.MarkReadForRecipient(r.Context(), userID, msg.ID); err != nil {
			if errors.Is(err, coremsg.ErrMessageNotFound) {
				web.JSONError(w, errors.New("message not found"), http.StatusNotFound)
				return
			}
			web.JSONError(w, err, http.StatusInternalServerError)
			return
		}
		h.tryPushReceipt(msg.FromUserID, coremsg.KindMessageRead, msg.ID)
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

	msg, err := h.Messaging.GetMessageForRecipient(r.Context(), userID, req.MessageID)
	if err != nil {
		if errors.Is(err, coremsg.ErrMessageNotFound) {
			web.JSONError(w, errors.New("message not found"), http.StatusNotFound)
			return
		}
		web.JSONError(w, err, http.StatusInternalServerError)
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
	h.tryPushReceipt(msg.FromUserID, coremsg.KindMessageDelivered, req.MessageID)
	w.WriteHeader(http.StatusOK)
}

func (h *MessagesHandler) tryPushReceipt(senderUserID int, kind coremsg.MessageKind, messageID int64) {
	if h.ReceiptTransport == nil || senderUserID <= 0 {
		return
	}
	_ = h.ReceiptTransport.SendDirect(senderUserID, coremsg.Message{
		Type: kind,
		ID:   messageID,
	})
}

func writeStoredMessagesJSON(w http.ResponseWriter, msgs []coremsg.StoredMessage) {
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(storedMessagesToJSON(msgs))
}

func storedMessagesToJSON(msgs []coremsg.StoredMessage) []map[string]any {
	resp := make([]map[string]any, 0, len(msgs))
	for _, msg := range msgs {
		item := map[string]any{
			"id":           msg.ID,
			"from_user_id": msg.FromUserID,
			"to_user_id":   msg.ToUserID,
			"body":         msg.Body,
			"created_at":   msg.CreatedAt,
		}
		if msg.Ciphertext != "" {
			item["ciphertext"] = msg.Ciphertext
		}
		if msg.EnvelopeVersion != "" {
			item["encryption_version"] = msg.EnvelopeVersion
		}
		if msg.SenderDeviceID > 0 {
			item["sender_device_id"] = msg.SenderDeviceID
		}
		if msg.RecipientDeviceID > 0 {
			item["recipient_device_id"] = msg.RecipientDeviceID
		}
		if msg.ClientMessageID > 0 {
			item["client_message_id"] = msg.ClientMessageID
		}
		if msg.DeliveryFailed {
			item["delivery_failed"] = true
		}
		if msg.DeliveredAt != nil {
			item["delivered_at"] = *msg.DeliveredAt
		}
		if msg.ReadAt != nil {
			item["read_at"] = *msg.ReadAt
		}
		resp = append(resp, item)
	}
	return resp
}
