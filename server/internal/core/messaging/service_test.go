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

type fakeCorrelationRecorder struct {
	last ClientMessageCorrelation
	err  error
}

func (f *fakeCorrelationRecorder) RecordClientMessageCorrelation(ctx context.Context, c ClientMessageCorrelation) error {
	_ = ctx
	f.last = c
	return f.err
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

func (f *fakePersistenceService) ListOutbox(ctx context.Context, userID int, limit int) ([]StoredMessage, error) {
	_ = ctx
	_ = userID
	_ = limit
	return nil, nil
}

func (f *fakePersistenceService) ListOutboxBefore(ctx context.Context, userID int, beforeID int64, limit int) ([]StoredMessage, error) {
	_ = ctx
	_ = userID
	_ = beforeID
	_ = limit
	return nil, nil
}

func (f *fakePersistenceService) ListOutboxAfter(ctx context.Context, userID int, afterID int64, limit int) ([]StoredMessage, error) {
	_ = ctx
	_ = userID
	_ = afterID
	_ = limit
	return nil, nil
}

func (f *fakePersistenceService) ListUnreadInbox(ctx context.Context, userID int, limit int) ([]StoredMessage, error) {
	_ = ctx
	_ = userID
	_ = limit
	return nil, nil
}

func (f *fakePersistenceService) ListInboxWithUser(ctx context.Context, userID int, withUserID int, limit int) ([]StoredMessage, error) {
	_ = ctx
	_ = userID
	_ = withUserID
	_ = limit
	return nil, nil
}

func (f *fakePersistenceService) ListUnreadInboxWithUser(ctx context.Context, userID int, withUserID int, limit int) ([]StoredMessage, error) {
	_ = ctx
	_ = userID
	_ = withUserID
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

func (f *fakePersistenceService) ListUnreadInboxBefore(ctx context.Context, userID int, beforeID int64, limit int) ([]StoredMessage, error) {
	_ = ctx
	_ = userID
	_ = beforeID
	_ = limit
	return nil, nil
}

func (f *fakePersistenceService) ListInboxBeforeWithUser(ctx context.Context, userID int, withUserID int, beforeID int64, limit int) ([]StoredMessage, error) {
	_ = ctx
	_ = userID
	_ = withUserID
	_ = beforeID
	_ = limit
	return nil, nil
}

func (f *fakePersistenceService) ListUnreadInboxBeforeWithUser(ctx context.Context, userID int, withUserID int, beforeID int64, limit int) ([]StoredMessage, error) {
	_ = ctx
	_ = userID
	_ = withUserID
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

func (f *fakePersistenceService) ListUnreadInboxAfter(ctx context.Context, userID int, afterID int64, limit int) ([]StoredMessage, error) {
	_ = ctx
	_ = userID
	_ = afterID
	_ = limit
	return nil, nil
}

func (f *fakePersistenceService) ListInboxAfterWithUser(ctx context.Context, userID int, withUserID int, afterID int64, limit int) ([]StoredMessage, error) {
	_ = ctx
	_ = userID
	_ = withUserID
	_ = afterID
	_ = limit
	return nil, nil
}

func (f *fakePersistenceService) ListUnreadInboxAfterWithUser(ctx context.Context, userID int, withUserID int, afterID int64, limit int) ([]StoredMessage, error) {
	_ = ctx
	_ = userID
	_ = withUserID
	_ = afterID
	_ = limit
	return nil, nil
}

func (f *fakePersistenceService) GetMessageForRecipient(ctx context.Context, recipientUserID int, messageID int64) (StoredMessage, error) {
	_ = ctx
	_ = recipientUserID
	return StoredMessage{ID: messageID}, nil
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

func TestDurableRelayService_SendDirect_RecordsClientCorrelation(t *testing.T) {
	tp := &fakeTransport{ok: true}
	ps := &fakePersistenceService{stored: StoredMessage{ID: 777}}
	cr := &fakeCorrelationRecorder{}
	svc := NewDurableRelayServiceWithCorrelation(tp, ps, cr)

	receipt, err := svc.SendDirect(context.Background(), DirectSendRequest{
		FromUserID: 1,
		ToUserID:   2,
		Body:       "hello",
		MessageID:  55,
	})
	if err != nil {
		t.Fatalf("SendDirect returned error: %v", err)
	}
	if !receipt.Delivered {
		t.Fatalf("expected delivered receipt, got %+v", receipt)
	}
	if cr.last.SenderUserID != 1 || cr.last.RecipientUserID != 2 || cr.last.ClientMessageID != 55 || cr.last.StoredMessageID != 777 {
		t.Fatalf("unexpected correlation record: %+v", cr.last)
	}
	if !cr.last.Delivered {
		t.Fatalf("expected delivered=true in correlation: %+v", cr.last)
	}
}

func TestDurableRelayService_SendDirect_RecordsOfflineCorrelation(t *testing.T) {
	tp := &fakeTransport{ok: false}
	ps := &fakePersistenceService{stored: StoredMessage{ID: 778}}
	cr := &fakeCorrelationRecorder{}
	svc := NewDurableRelayServiceWithCorrelation(tp, ps, cr)

	receipt, err := svc.SendDirect(context.Background(), DirectSendRequest{
		FromUserID: 1,
		ToUserID:   2,
		Body:       "hello",
		MessageID:  56,
	})
	if err != nil {
		t.Fatalf("SendDirect returned error: %v", err)
	}
	if receipt.Delivered {
		t.Fatalf("expected offline receipt, got %+v", receipt)
	}
	if cr.last.ClientMessageID != 56 || cr.last.StoredMessageID != 778 {
		t.Fatalf("unexpected correlation record: %+v", cr.last)
	}
	if cr.last.Delivered {
		t.Fatalf("expected delivered=false in correlation: %+v", cr.last)
	}
}
