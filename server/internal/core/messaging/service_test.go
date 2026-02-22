package messaging

import (
	"context"
	"testing"
)

type fakeTransport struct {
	ok      bool
	toUser  int
	lastMsg Message
}

func (f *fakeTransport) SendDirect(toUserID int, msg Message) bool {
	f.toUser = toUserID
	f.lastMsg = msg
	return f.ok
}

func TestRelayService_SendDirect_ReturnsDeliveredReceipt(t *testing.T) {
	tp := &fakeTransport{ok: true}
	svc := NewRelayService(tp)

	receipt, err := svc.SendDirect(context.Background(), DirectSendRequest{
		FromUserID: 1,
		From:       "alice",
		ToUserID:   2,
		Body:       "hello",
		MessageID:  44,
	})
	if err != nil {
		t.Fatalf("SendDirect returned error: %v", err)
	}
	if !receipt.Delivered || receipt.MessageID != 44 {
		t.Fatalf("unexpected receipt: %+v", receipt)
	}
	if tp.toUser != 2 || tp.lastMsg.Type != KindDirectMessage || tp.lastMsg.From != "alice" || tp.lastMsg.Body != "hello" {
		t.Fatalf("unexpected transport call: to=%d msg=%+v", tp.toUser, tp.lastMsg)
	}
}

func TestRelayService_SendDirect_ReturnsOfflineReceiptWhenTransportFails(t *testing.T) {
	tp := &fakeTransport{ok: false}
	svc := NewRelayService(tp)

	receipt, err := svc.SendDirect(context.Background(), DirectSendRequest{
		From:      "alice",
		ToUserID:  2,
		Body:      "hello",
		MessageID: 45,
	})
	if err != nil {
		t.Fatalf("SendDirect returned error: %v", err)
	}
	if receipt.Delivered {
		t.Fatalf("expected undelivered receipt, got %+v", receipt)
	}
	if receipt.Reason != "recipient_offline" {
		t.Fatalf("unexpected reason %q", receipt.Reason)
	}
}

type fakePersistenceService struct {
	stored       StoredMessage
	storeErr     error
	markDelErr   error
	lastStoreReq PersistDirectMessageRequest
	lastMarkID   int64
}

func (f *fakePersistenceService) StoreDirectMessage(ctx context.Context, req PersistDirectMessageRequest) (StoredMessage, error) {
	_ = ctx
	f.lastStoreReq = req
	if f.storeErr != nil {
		return StoredMessage{}, f.storeErr
	}
	if f.stored.ID == 0 {
		f.stored = StoredMessage{ID: 99, FromUserID: req.FromUserID, ToUserID: req.ToUserID, Body: req.Body}
	}
	return f.stored, nil
}

func (f *fakePersistenceService) MarkDelivered(ctx context.Context, messageID int64) error {
	_ = ctx
	f.lastMarkID = messageID
	return f.markDelErr
}

func (f *fakePersistenceService) MarkDeliveredForRecipient(ctx context.Context, recipientUserID int, messageID int64) error {
	_ = ctx
	_ = recipientUserID
	f.lastMarkID = messageID
	return f.markDelErr
}

func (f *fakePersistenceService) MarkRead(ctx context.Context, messageID int64) error {
	_ = ctx
	_ = messageID
	return nil
}

func (f *fakePersistenceService) MarkReadForRecipient(ctx context.Context, recipientUserID int, messageID int64) error {
	_ = ctx
	_ = recipientUserID
	_ = messageID
	return nil
}

func (f *fakePersistenceService) ListInbox(ctx context.Context, userID int, limit int) ([]StoredMessage, error) {
	_ = ctx
	_ = userID
	_ = limit
	return nil, nil
}

func (f *fakePersistenceService) ListInboxBefore(ctx context.Context, userID int, beforeID int64, limit int) ([]StoredMessage, error) {
	_ = ctx
	_ = userID
	_ = beforeID
	_ = limit
	return nil, nil
}

func (f *fakePersistenceService) ListInboxAfter(ctx context.Context, userID int, afterID int64, limit int) ([]StoredMessage, error) {
	_ = ctx
	_ = userID
	_ = afterID
	_ = limit
	return nil, nil
}

func TestDurableRelayService_SendDirect_PersistsAndMarksDeliveredOnSuccess(t *testing.T) {
	tp := &fakeTransport{ok: true}
	ps := &fakePersistenceService{stored: StoredMessage{ID: 123}}
	svc := NewDurableRelayService(tp, ps)

	receipt, err := svc.SendDirect(context.Background(), DirectSendRequest{
		FromUserID: 1,
		From:       "alice",
		ToUserID:   2,
		Body:       "hello",
		MessageID:  44,
	})
	if err != nil {
		t.Fatalf("SendDirect returned error: %v", err)
	}
	if !receipt.Delivered || receipt.MessageID != 44 {
		t.Fatalf("unexpected receipt: %+v", receipt)
	}
	if receipt.StoredMessageID != 123 {
		t.Fatalf("StoredMessageID = %d, want 123", receipt.StoredMessageID)
	}
	if ps.lastStoreReq.FromUserID != 1 || ps.lastStoreReq.ToUserID != 2 || ps.lastStoreReq.Body != "hello" {
		t.Fatalf("unexpected persistence store req: %+v", ps.lastStoreReq)
	}
	if ps.lastMarkID != 123 {
		t.Fatalf("mark delivered id = %d, want 123", ps.lastMarkID)
	}
}

func TestDurableRelayService_SendDirect_PersistsButDoesNotMarkDeliveredWhenOffline(t *testing.T) {
	tp := &fakeTransport{ok: false}
	ps := &fakePersistenceService{stored: StoredMessage{ID: 321}}
	svc := NewDurableRelayService(tp, ps)

	receipt, err := svc.SendDirect(context.Background(), DirectSendRequest{
		FromUserID: 1,
		ToUserID:   2,
		Body:       "hello",
		MessageID:  45,
	})
	if err != nil {
		t.Fatalf("SendDirect returned error: %v", err)
	}
	if receipt.Delivered {
		t.Fatalf("expected offline receipt, got %+v", receipt)
	}
	if receipt.Reason != "recipient_offline" {
		t.Fatalf("unexpected reason %q", receipt.Reason)
	}
	if receipt.StoredMessageID != 321 {
		t.Fatalf("StoredMessageID = %d, want 321", receipt.StoredMessageID)
	}
	if ps.lastMarkID != 0 {
		t.Fatalf("expected no MarkDelivered call, got %d", ps.lastMarkID)
	}
}
