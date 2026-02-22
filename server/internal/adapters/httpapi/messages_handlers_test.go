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
	listResp        []coremsg.StoredMessage
	listErr         error
	lastUserID      int
	lastLimit       int
	lastDeliveredID int64
	deliveredErr    error
	lastReadID      int64
	readErr         error
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

func (f *fakeMessagingPersistence) ListInbox(ctx context.Context, userID int, limit int) ([]coremsg.StoredMessage, error) {
	_ = ctx
	f.lastUserID = userID
	f.lastLimit = limit
	return f.listResp, f.listErr
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
		svc := &fakeMessagingPersistence{}
		h := &MessagesHandler{Messaging: svc}

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
}

func TestMessagesHandler_MarkDelivered_ValidatesAndDelegates(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		svc := &fakeMessagingPersistence{}
		h := &MessagesHandler{Messaging: svc}

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
