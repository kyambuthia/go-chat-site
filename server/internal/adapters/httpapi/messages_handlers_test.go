package httpapi

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/kyambuthia/go-chat-site/server/internal/auth"
	coremsg "github.com/kyambuthia/go-chat-site/server/internal/core/messaging"
)

type fakeMessagingPersistence struct {
	listResp         []coremsg.StoredMessage
	listErr          error
	outboxResp       []coremsg.StoredMessage
	outboxErr        error
	lastUserID       int
	lastLimit        int
	lastWithUserID   int
	lastUnreadOnly   bool
	lastBeforeID     int64
	listBeforeErr    error
	lastAfterID      int64
	listAfterErr     error
	listWithUserErr  error
	lastDeliveredID  int64
	deliveredErr     error
	lastReadID       int64
	readErr          error
	getMsgResp       coremsg.StoredMessage
	getMsgErr        error
	lastGetMsgUserID int
	lastGetMsgID     int64
}

func (f *fakeMessagingPersistence) StoreDirectMessage(ctx context.Context, req coremsg.PersistDirectMessageRequest) (coremsg.StoredMessage, error) {
	_ = ctx
	_ = req
	return coremsg.StoredMessage{}, nil
}

func (f *fakeMessagingPersistence) MarkDelivered(ctx context.Context, messageID int64) error {
	_ = ctx
	f.lastDeliveredID = messageID
	return f.deliveredErr
}

func (f *fakeMessagingPersistence) MarkDeliveredForRecipient(ctx context.Context, recipientUserID int, messageID int64) error {
	_ = ctx
	_ = recipientUserID
	f.lastDeliveredID = messageID
	return f.deliveredErr
}

func (f *fakeMessagingPersistence) MarkRead(ctx context.Context, messageID int64) error {
	_ = ctx
	f.lastReadID = messageID
	return f.readErr
}

func (f *fakeMessagingPersistence) MarkReadForRecipient(ctx context.Context, recipientUserID int, messageID int64) error {
	_ = ctx
	_ = recipientUserID
	f.lastReadID = messageID
	return f.readErr
}

func (f *fakeMessagingPersistence) GetMessageForRecipient(ctx context.Context, recipientUserID int, messageID int64) (coremsg.StoredMessage, error) {
	_ = ctx
	f.lastGetMsgUserID = recipientUserID
	f.lastGetMsgID = messageID
	if f.getMsgErr != nil {
		return coremsg.StoredMessage{}, f.getMsgErr
	}
	return f.getMsgResp, nil
}

func (f *fakeMessagingPersistence) ListInbox(ctx context.Context, userID int, limit int) ([]coremsg.StoredMessage, error) {
	_ = ctx
	f.lastUserID = userID
	f.lastLimit = limit
	return f.listResp, f.listErr
}

func (f *fakeMessagingPersistence) ListOutbox(ctx context.Context, userID int, limit int) ([]coremsg.StoredMessage, error) {
	_ = ctx
	f.lastUserID = userID
	f.lastLimit = limit
	if f.outboxResp != nil || f.outboxErr != nil {
		return f.outboxResp, f.outboxErr
	}
	return f.listResp, f.listErr
}

func (f *fakeMessagingPersistence) ListOutboxBefore(ctx context.Context, userID int, beforeID int64, limit int) ([]coremsg.StoredMessage, error) {
	_ = ctx
	f.lastUserID = userID
	f.lastBeforeID = beforeID
	f.lastLimit = limit
	if f.outboxResp != nil || f.outboxErr != nil {
		return f.outboxResp, f.outboxErr
	}
	return f.listResp, f.listBeforeErr
}

func (f *fakeMessagingPersistence) ListOutboxAfter(ctx context.Context, userID int, afterID int64, limit int) ([]coremsg.StoredMessage, error) {
	_ = ctx
	f.lastUserID = userID
	f.lastAfterID = afterID
	f.lastLimit = limit
	if f.outboxResp != nil || f.outboxErr != nil {
		return f.outboxResp, f.outboxErr
	}
	return f.listResp, f.listAfterErr
}

func (f *fakeMessagingPersistence) ListUnreadInbox(ctx context.Context, userID int, limit int) ([]coremsg.StoredMessage, error) {
	_ = ctx
	f.lastUserID = userID
	f.lastLimit = limit
	f.lastUnreadOnly = true
	return f.listResp, f.listErr
}

func (f *fakeMessagingPersistence) ListInboxWithUser(ctx context.Context, userID int, withUserID int, limit int) ([]coremsg.StoredMessage, error) {
	_ = ctx
	f.lastUserID = userID
	f.lastWithUserID = withUserID
	f.lastLimit = limit
	return f.listResp, f.listWithUserErr
}

func (f *fakeMessagingPersistence) ListUnreadInboxWithUser(ctx context.Context, userID int, withUserID int, limit int) ([]coremsg.StoredMessage, error) {
	_ = ctx
	f.lastUserID = userID
	f.lastWithUserID = withUserID
	f.lastLimit = limit
	f.lastUnreadOnly = true
	return f.listResp, f.listWithUserErr
}

func (f *fakeMessagingPersistence) ListInboxBefore(ctx context.Context, userID int, beforeID int64, limit int) ([]coremsg.StoredMessage, error) {
	_ = ctx
	f.lastUserID = userID
	f.lastBeforeID = beforeID
	f.lastLimit = limit
	return f.listResp, f.listBeforeErr
}

func (f *fakeMessagingPersistence) ListUnreadInboxBefore(ctx context.Context, userID int, beforeID int64, limit int) ([]coremsg.StoredMessage, error) {
	_ = ctx
	f.lastUserID = userID
	f.lastBeforeID = beforeID
	f.lastLimit = limit
	f.lastUnreadOnly = true
	return f.listResp, f.listBeforeErr
}

func (f *fakeMessagingPersistence) ListInboxBeforeWithUser(ctx context.Context, userID int, withUserID int, beforeID int64, limit int) ([]coremsg.StoredMessage, error) {
	_ = ctx
	f.lastUserID = userID
	f.lastWithUserID = withUserID
	f.lastBeforeID = beforeID
	f.lastLimit = limit
	return f.listResp, f.listBeforeErr
}

func (f *fakeMessagingPersistence) ListUnreadInboxBeforeWithUser(ctx context.Context, userID int, withUserID int, beforeID int64, limit int) ([]coremsg.StoredMessage, error) {
	_ = ctx
	f.lastUserID = userID
	f.lastWithUserID = withUserID
	f.lastBeforeID = beforeID
	f.lastLimit = limit
	f.lastUnreadOnly = true
	return f.listResp, f.listBeforeErr
}

func (f *fakeMessagingPersistence) ListInboxAfter(ctx context.Context, userID int, afterID int64, limit int) ([]coremsg.StoredMessage, error) {
	_ = ctx
	f.lastUserID = userID
	f.lastAfterID = afterID
	f.lastLimit = limit
	return f.listResp, f.listAfterErr
}

func (f *fakeMessagingPersistence) ListUnreadInboxAfter(ctx context.Context, userID int, afterID int64, limit int) ([]coremsg.StoredMessage, error) {
	_ = ctx
	f.lastUserID = userID
	f.lastAfterID = afterID
	f.lastLimit = limit
	f.lastUnreadOnly = true
	return f.listResp, f.listAfterErr
}

func (f *fakeMessagingPersistence) ListInboxAfterWithUser(ctx context.Context, userID int, withUserID int, afterID int64, limit int) ([]coremsg.StoredMessage, error) {
	_ = ctx
	f.lastUserID = userID
	f.lastWithUserID = withUserID
	f.lastAfterID = afterID
	f.lastLimit = limit
	return f.listResp, f.listAfterErr
}

func (f *fakeMessagingPersistence) ListUnreadInboxAfterWithUser(ctx context.Context, userID int, withUserID int, afterID int64, limit int) ([]coremsg.StoredMessage, error) {
	_ = ctx
	f.lastUserID = userID
	f.lastWithUserID = withUserID
	f.lastAfterID = afterID
	f.lastLimit = limit
	f.lastUnreadOnly = true
	return f.listResp, f.listAfterErr
}

type fakeTransport struct {
	ok      bool
	lastTo  int
	lastMsg coremsg.Message
}

func (f *fakeTransport) SendDirect(toUserID int, msg coremsg.Message) bool {
	f.lastTo = toUserID
	f.lastMsg = msg
	return f.ok
}

func TestMessagesHandler_GetInbox_UsesPersistenceServiceAndSupportsLimit(t *testing.T) {
	now := time.Now().UTC().Truncate(time.Second)
	delivered := now.Add(1 * time.Minute)
	svc := &fakeMessagingPersistence{
		listResp: []coremsg.StoredMessage{{
			ID:          10,
			FromUserID:  1,
			ToUserID:    2,
			Body:        "hello",
			CreatedAt:   now,
			DeliveredAt: &delivered,
		}},
	}
	h := &MessagesHandler{Messaging: svc}

	req := httptest.NewRequest(http.MethodGet, "/api/messages/inbox?limit=25", nil)
	req = req.WithContext(auth.WithUserID(req.Context(), 2))
	rr := httptest.NewRecorder()
	h.GetInbox(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", rr.Code)
	}
	if svc.lastUserID != 2 || svc.lastLimit != 25 {
		t.Fatalf("unexpected ListInbox call user=%d limit=%d", svc.lastUserID, svc.lastLimit)
	}

	var resp []map[string]any
	if err := json.Unmarshal(rr.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal response: %v", err)
	}
	if len(resp) != 1 {
		t.Fatalf("expected 1 message, got %d", len(resp))
	}
	if got := int(resp[0]["id"].(float64)); got != 10 {
		t.Fatalf("id = %d, want 10", got)
	}
	if got := int(resp[0]["from_user_id"].(float64)); got != 1 {
		t.Fatalf("from_user_id = %d, want 1", got)
	}
	if _, ok := resp[0]["created_at"]; !ok {
		t.Fatal("expected created_at field")
	}
}

func TestMessagesHandler_GetOutbox_UsesPersistenceServiceAndSupportsLimit(t *testing.T) {
	now := time.Now().UTC().Truncate(time.Second)
	svc := &fakeMessagingPersistence{
		outboxResp: []coremsg.StoredMessage{{
			ID:         20,
			FromUserID: 2,
			ToUserID:   1,
			Body:       "sent",
			CreatedAt:  now,
		}},
	}
	h := &MessagesHandler{Messaging: svc}

	req := httptest.NewRequest(http.MethodGet, "/api/messages/outbox?limit=5", nil)
	req = req.WithContext(auth.WithUserID(req.Context(), 2))
	rr := httptest.NewRecorder()
	h.GetOutbox(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", rr.Code)
	}
	if svc.lastUserID != 2 || svc.lastLimit != 5 {
		t.Fatalf("unexpected ListOutbox call user=%d limit=%d", svc.lastUserID, svc.lastLimit)
	}

	var resp []map[string]any
	if err := json.Unmarshal(rr.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal response: %v", err)
	}
	if len(resp) != 1 {
		t.Fatalf("expected 1 message, got %d", len(resp))
	}
	if got := int(resp[0]["id"].(float64)); got != 20 {
		t.Fatalf("id = %d, want 20", got)
	}
	if got := int(resp[0]["from_user_id"].(float64)); got != 2 {
		t.Fatalf("from_user_id = %d, want 2", got)
	}
}

func TestMessagesHandler_GetOutbox_SupportsBeforeAndAfterCursors(t *testing.T) {
	t.Run("before_id", func(t *testing.T) {
		svc := &fakeMessagingPersistence{}
		h := &MessagesHandler{Messaging: svc}

		req := httptest.NewRequest(http.MethodGet, "/api/messages/outbox?before_id=90&limit=10", nil)
		req = req.WithContext(auth.WithUserID(req.Context(), 2))
		rr := httptest.NewRecorder()
		h.GetOutbox(rr, req)

		if rr.Code != http.StatusOK {
			t.Fatalf("status = %d, want 200", rr.Code)
		}
		if svc.lastUserID != 2 || svc.lastBeforeID != 90 || svc.lastLimit != 10 {
			t.Fatalf("unexpected ListOutboxBefore call user=%d before=%d limit=%d", svc.lastUserID, svc.lastBeforeID, svc.lastLimit)
		}
	})

	t.Run("after_id", func(t *testing.T) {
		svc := &fakeMessagingPersistence{}
		h := &MessagesHandler{Messaging: svc}

		req := httptest.NewRequest(http.MethodGet, "/api/messages/outbox?after_id=40&limit=10", nil)
		req = req.WithContext(auth.WithUserID(req.Context(), 2))
		rr := httptest.NewRecorder()
		h.GetOutbox(rr, req)

		if rr.Code != http.StatusOK {
			t.Fatalf("status = %d, want 200", rr.Code)
		}
		if svc.lastUserID != 2 || svc.lastAfterID != 40 || svc.lastLimit != 10 {
			t.Fatalf("unexpected ListOutboxAfter call user=%d after=%d limit=%d", svc.lastUserID, svc.lastAfterID, svc.lastLimit)
		}
	})
}

func TestMessagesHandler_GetInbox_SupportsBeforeIDCursor(t *testing.T) {
	svc := &fakeMessagingPersistence{}
	h := &MessagesHandler{Messaging: svc}

	req := httptest.NewRequest(http.MethodGet, "/api/messages/inbox?before_id=99&limit=10", nil)
	req = req.WithContext(auth.WithUserID(req.Context(), 2))
	rr := httptest.NewRecorder()
	h.GetInbox(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", rr.Code)
	}
	if svc.lastUserID != 2 || svc.lastBeforeID != 99 || svc.lastLimit != 10 {
		t.Fatalf("unexpected ListInboxBefore call user=%d before=%d limit=%d", svc.lastUserID, svc.lastBeforeID, svc.lastLimit)
	}
}

func TestMessagesHandler_GetInbox_SupportsAfterIDCursor(t *testing.T) {
	svc := &fakeMessagingPersistence{}
	h := &MessagesHandler{Messaging: svc}

	req := httptest.NewRequest(http.MethodGet, "/api/messages/inbox?after_id=50&limit=10", nil)
	req = req.WithContext(auth.WithUserID(req.Context(), 2))
	rr := httptest.NewRecorder()
	h.GetInbox(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", rr.Code)
	}
	if svc.lastUserID != 2 || svc.lastAfterID != 50 || svc.lastLimit != 10 {
		t.Fatalf("unexpected ListInboxAfter call user=%d after=%d limit=%d", svc.lastUserID, svc.lastAfterID, svc.lastLimit)
	}
}

func TestMessagesHandler_GetInbox_SupportsWithUserFilter(t *testing.T) {
	svc := &fakeMessagingPersistence{}
	h := &MessagesHandler{Messaging: svc}

	req := httptest.NewRequest(http.MethodGet, "/api/messages/inbox?with_user_id=7&limit=10", nil)
	req = req.WithContext(auth.WithUserID(req.Context(), 2))
	rr := httptest.NewRecorder()
	h.GetInbox(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", rr.Code)
	}
	if svc.lastUserID != 2 || svc.lastWithUserID != 7 || svc.lastLimit != 10 {
		t.Fatalf("unexpected ListInboxWithUser call user=%d with_user=%d limit=%d", svc.lastUserID, svc.lastWithUserID, svc.lastLimit)
	}
}

func TestMessagesHandler_GetInbox_SupportsUnreadOnlyFilter(t *testing.T) {
	svc := &fakeMessagingPersistence{}
	h := &MessagesHandler{Messaging: svc}

	req := httptest.NewRequest(http.MethodGet, "/api/messages/inbox?unread_only=true&limit=10", nil)
	req = req.WithContext(auth.WithUserID(req.Context(), 2))
	rr := httptest.NewRecorder()
	h.GetInbox(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", rr.Code)
	}
	if !svc.lastUnreadOnly || svc.lastUserID != 2 || svc.lastLimit != 10 {
		t.Fatalf("unexpected unread ListInbox call unread=%v user=%d limit=%d", svc.lastUnreadOnly, svc.lastUserID, svc.lastLimit)
	}
}

func TestMessagesHandler_GetInbox_MapsErrorsAndInvalidLimit(t *testing.T) {
	t.Run("invalid limit", func(t *testing.T) {
		h := &MessagesHandler{Messaging: &fakeMessagingPersistence{}}
		req := httptest.NewRequest(http.MethodGet, "/api/messages/inbox?limit=abc", nil)
		req = req.WithContext(auth.WithUserID(req.Context(), 2))
		rr := httptest.NewRecorder()

		h.GetInbox(rr, req)
		if rr.Code != http.StatusBadRequest {
			t.Fatalf("status = %d, want 400", rr.Code)
		}
	})

	t.Run("invalid before_id", func(t *testing.T) {
		h := &MessagesHandler{Messaging: &fakeMessagingPersistence{}}
		req := httptest.NewRequest(http.MethodGet, "/api/messages/inbox?before_id=abc", nil)
		req = req.WithContext(auth.WithUserID(req.Context(), 2))
		rr := httptest.NewRecorder()

		h.GetInbox(rr, req)
		if rr.Code != http.StatusBadRequest {
			t.Fatalf("status = %d, want 400", rr.Code)
		}
	})

	t.Run("invalid after_id", func(t *testing.T) {
		h := &MessagesHandler{Messaging: &fakeMessagingPersistence{}}
		req := httptest.NewRequest(http.MethodGet, "/api/messages/inbox?after_id=abc", nil)
		req = req.WithContext(auth.WithUserID(req.Context(), 2))
		rr := httptest.NewRecorder()

		h.GetInbox(rr, req)
		if rr.Code != http.StatusBadRequest {
			t.Fatalf("status = %d, want 400", rr.Code)
		}
	})

	t.Run("invalid with_user_id", func(t *testing.T) {
		h := &MessagesHandler{Messaging: &fakeMessagingPersistence{}}
		req := httptest.NewRequest(http.MethodGet, "/api/messages/inbox?with_user_id=abc", nil)
		req = req.WithContext(auth.WithUserID(req.Context(), 2))
		rr := httptest.NewRecorder()

		h.GetInbox(rr, req)
		if rr.Code != http.StatusBadRequest {
			t.Fatalf("status = %d, want 400", rr.Code)
		}
	})

	t.Run("invalid unread_only", func(t *testing.T) {
		h := &MessagesHandler{Messaging: &fakeMessagingPersistence{}}
		req := httptest.NewRequest(http.MethodGet, "/api/messages/inbox?unread_only=maybe", nil)
		req = req.WithContext(auth.WithUserID(req.Context(), 2))
		rr := httptest.NewRecorder()

		h.GetInbox(rr, req)
		if rr.Code != http.StatusBadRequest {
			t.Fatalf("status = %d, want 400", rr.Code)
		}
	})

	t.Run("before and after are mutually exclusive", func(t *testing.T) {
		h := &MessagesHandler{Messaging: &fakeMessagingPersistence{}}
		req := httptest.NewRequest(http.MethodGet, "/api/messages/inbox?before_id=10&after_id=5", nil)
		req = req.WithContext(auth.WithUserID(req.Context(), 2))
		rr := httptest.NewRecorder()

		h.GetInbox(rr, req)
		if rr.Code != http.StatusBadRequest {
			t.Fatalf("status = %d, want 400", rr.Code)
		}
	})

	t.Run("service error", func(t *testing.T) {
		h := &MessagesHandler{Messaging: &fakeMessagingPersistence{listErr: errors.New("db down")}}
		req := httptest.NewRequest(http.MethodGet, "/api/messages/inbox", nil)
		req = req.WithContext(auth.WithUserID(req.Context(), 2))
		rr := httptest.NewRecorder()

		h.GetInbox(rr, req)
		if rr.Code != http.StatusInternalServerError {
			t.Fatalf("status = %d, want 500", rr.Code)
		}
	})
}

func TestMessagesHandler_MarkRead_ValidatesAndDelegates(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		svc := &fakeMessagingPersistence{getMsgResp: coremsg.StoredMessage{ID: 42, FromUserID: 7, ToUserID: 2}}
		tp := &fakeTransport{ok: true}
		h := &MessagesHandler{Messaging: svc, ReceiptTransport: tp}

		req := httptest.NewRequest(http.MethodPost, "/api/messages/read", bytes.NewReader([]byte(`{"message_id":42}`)))
		req.Header.Set("Content-Type", "application/json")
		req = req.WithContext(auth.WithUserID(req.Context(), 2))
		rr := httptest.NewRecorder()

		h.MarkRead(rr, req)
		if rr.Code != http.StatusOK {
			t.Fatalf("status = %d, want 200", rr.Code)
		}
		if svc.lastReadID != 42 {
			t.Fatalf("lastReadID = %d, want 42", svc.lastReadID)
		}
		if svc.lastGetMsgUserID != 2 || svc.lastGetMsgID != 42 {
			t.Fatalf("unexpected GetMessageForRecipient call user=%d msg=%d", svc.lastGetMsgUserID, svc.lastGetMsgID)
		}
		if tp.lastTo != 7 || tp.lastMsg.Type != coremsg.KindMessageRead || tp.lastMsg.ID != 42 {
			t.Fatalf("unexpected receipt push to=%d msg=%+v", tp.lastTo, tp.lastMsg)
		}
	})

	t.Run("invalid body", func(t *testing.T) {
		h := &MessagesHandler{Messaging: &fakeMessagingPersistence{}}
		req := httptest.NewRequest(http.MethodPost, "/api/messages/read", bytes.NewReader([]byte(`{`)))
		req.Header.Set("Content-Type", "application/json")
		req = req.WithContext(auth.WithUserID(req.Context(), 2))
		rr := httptest.NewRecorder()

		h.MarkRead(rr, req)
		if rr.Code != http.StatusBadRequest {
			t.Fatalf("status = %d, want 400", rr.Code)
		}
	})

	t.Run("invalid message id", func(t *testing.T) {
		h := &MessagesHandler{Messaging: &fakeMessagingPersistence{}}
		req := httptest.NewRequest(http.MethodPost, "/api/messages/read", bytes.NewReader([]byte(`{"message_id":0}`)))
		req.Header.Set("Content-Type", "application/json")
		req = req.WithContext(auth.WithUserID(req.Context(), 2))
		rr := httptest.NewRecorder()

		h.MarkRead(rr, req)
		if rr.Code != http.StatusBadRequest {
			t.Fatalf("status = %d, want 400", rr.Code)
		}
	})

	t.Run("service error", func(t *testing.T) {
		h := &MessagesHandler{Messaging: &fakeMessagingPersistence{readErr: errors.New("db down")}}
		req := httptest.NewRequest(http.MethodPost, "/api/messages/read", bytes.NewReader([]byte(`{"message_id":42}`)))
		req.Header.Set("Content-Type", "application/json")
		req = req.WithContext(auth.WithUserID(req.Context(), 2))
		rr := httptest.NewRecorder()

		h.MarkRead(rr, req)
		if rr.Code != http.StatusInternalServerError {
			t.Fatalf("status = %d, want 500", rr.Code)
		}
	})

	t.Run("message not found", func(t *testing.T) {
		h := &MessagesHandler{Messaging: &fakeMessagingPersistence{readErr: coremsg.ErrMessageNotFound}}
		req := httptest.NewRequest(http.MethodPost, "/api/messages/read", bytes.NewReader([]byte(`{"message_id":42}`)))
		req.Header.Set("Content-Type", "application/json")
		req = req.WithContext(auth.WithUserID(req.Context(), 2))
		rr := httptest.NewRecorder()

		h.MarkRead(rr, req)
		if rr.Code != http.StatusNotFound {
			t.Fatalf("status = %d, want 404", rr.Code)
		}
	})

	t.Run("offline sender does not fail request", func(t *testing.T) {
		svc := &fakeMessagingPersistence{getMsgResp: coremsg.StoredMessage{ID: 42, FromUserID: 7, ToUserID: 2}}
		h := &MessagesHandler{Messaging: svc, ReceiptTransport: &fakeTransport{ok: false}}
		req := httptest.NewRequest(http.MethodPost, "/api/messages/read", bytes.NewReader([]byte(`{"message_id":42}`)))
		req.Header.Set("Content-Type", "application/json")
		req = req.WithContext(auth.WithUserID(req.Context(), 2))
		rr := httptest.NewRecorder()

		h.MarkRead(rr, req)
		if rr.Code != http.StatusOK {
			t.Fatalf("status = %d, want 200", rr.Code)
		}
	})
}

func TestMessagesHandler_MarkDelivered_ValidatesAndDelegates(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		svc := &fakeMessagingPersistence{getMsgResp: coremsg.StoredMessage{ID: 43, FromUserID: 7, ToUserID: 2}}
		tp := &fakeTransport{ok: true}
		h := &MessagesHandler{Messaging: svc, ReceiptTransport: tp}

		req := httptest.NewRequest(http.MethodPost, "/api/messages/delivered", bytes.NewReader([]byte(`{"message_id":43}`)))
		req.Header.Set("Content-Type", "application/json")
		req = req.WithContext(auth.WithUserID(req.Context(), 2))
		rr := httptest.NewRecorder()

		h.MarkDelivered(rr, req)
		if rr.Code != http.StatusOK {
			t.Fatalf("status = %d, want 200", rr.Code)
		}
		if svc.lastDeliveredID != 43 {
			t.Fatalf("lastDeliveredID = %d, want 43", svc.lastDeliveredID)
		}
		if svc.lastGetMsgUserID != 2 || svc.lastGetMsgID != 43 {
			t.Fatalf("unexpected GetMessageForRecipient call user=%d msg=%d", svc.lastGetMsgUserID, svc.lastGetMsgID)
		}
		if tp.lastTo != 7 || tp.lastMsg.Type != coremsg.KindMessageDelivered || tp.lastMsg.ID != 43 {
			t.Fatalf("unexpected receipt push to=%d msg=%+v", tp.lastTo, tp.lastMsg)
		}
	})

	t.Run("invalid body", func(t *testing.T) {
		h := &MessagesHandler{Messaging: &fakeMessagingPersistence{}}
		req := httptest.NewRequest(http.MethodPost, "/api/messages/delivered", bytes.NewReader([]byte(`{`)))
		req.Header.Set("Content-Type", "application/json")
		req = req.WithContext(auth.WithUserID(req.Context(), 2))
		rr := httptest.NewRecorder()

		h.MarkDelivered(rr, req)
		if rr.Code != http.StatusBadRequest {
			t.Fatalf("status = %d, want 400", rr.Code)
		}
	})

	t.Run("service error", func(t *testing.T) {
		h := &MessagesHandler{Messaging: &fakeMessagingPersistence{deliveredErr: errors.New("db down")}}
		req := httptest.NewRequest(http.MethodPost, "/api/messages/delivered", bytes.NewReader([]byte(`{"message_id":43}`)))
		req.Header.Set("Content-Type", "application/json")
		req = req.WithContext(auth.WithUserID(req.Context(), 2))
		rr := httptest.NewRecorder()

		h.MarkDelivered(rr, req)
		if rr.Code != http.StatusInternalServerError {
			t.Fatalf("status = %d, want 500", rr.Code)
		}
	})

	t.Run("message not found", func(t *testing.T) {
		h := &MessagesHandler{Messaging: &fakeMessagingPersistence{deliveredErr: coremsg.ErrMessageNotFound}}
		req := httptest.NewRequest(http.MethodPost, "/api/messages/delivered", bytes.NewReader([]byte(`{"message_id":43}`)))
		req.Header.Set("Content-Type", "application/json")
		req = req.WithContext(auth.WithUserID(req.Context(), 2))
		rr := httptest.NewRecorder()

		h.MarkDelivered(rr, req)
		if rr.Code != http.StatusNotFound {
			t.Fatalf("status = %d, want 404", rr.Code)
		}
	})
}
