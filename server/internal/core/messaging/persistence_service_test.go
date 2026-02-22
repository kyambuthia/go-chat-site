package messaging

import (
	"context"
	"errors"
	"testing"
	"time"
)

type fakePersistenceRepo struct {
	saveResp                    StoredMessage
	saveErr                     error
	lastSave                    StoredMessage
	lastDelivered               int64
	deliveredErr                error
	lastRead                    int64
	lastReadUserID              int
	readErr                     error
	lastDeliveredRecipientID    int
	lastDeliveredRecipientMsgID int64
	deliveredRecipientErr       error
	lastReadRecipientID         int
	lastReadRecipientMsgID      int64
	readRecipientErr            error
	listResp                    []StoredMessage
	listErr                     error
	lastListUserID              int
	lastListLimit               int
}

func (f *fakePersistenceRepo) SaveDirectMessage(ctx context.Context, msg StoredMessage) (StoredMessage, error) {
	_ = ctx
	f.lastSave = msg
	if f.saveErr != nil {
		return StoredMessage{}, f.saveErr
	}
	if f.saveResp.ID == 0 {
		f.saveResp = msg
		f.saveResp.ID = 1
	}
	return f.saveResp, nil
}

func (f *fakePersistenceRepo) MarkDelivered(ctx context.Context, messageID int64, deliveredAt time.Time) error {
	_ = ctx
	_ = deliveredAt
	f.lastDelivered = messageID
	return f.deliveredErr
}

func (f *fakePersistenceRepo) MarkDeliveredForRecipient(ctx context.Context, recipientUserID int, messageID int64, deliveredAt time.Time) error {
	_ = ctx
	_ = deliveredAt
	f.lastDeliveredRecipientID = recipientUserID
	f.lastDeliveredRecipientMsgID = messageID
	return f.deliveredRecipientErr
}

func (f *fakePersistenceRepo) MarkRead(ctx context.Context, messageID int64, readAt time.Time) error {
	_ = ctx
	_ = readAt
	f.lastRead = messageID
	return f.readErr
}

func (f *fakePersistenceRepo) MarkReadForRecipient(ctx context.Context, recipientUserID int, messageID int64, readAt time.Time) error {
	_ = ctx
	_ = readAt
	f.lastReadRecipientID = recipientUserID
	f.lastReadRecipientMsgID = messageID
	return f.readRecipientErr
}

func (f *fakePersistenceRepo) ListInbox(ctx context.Context, userID int, limit int) ([]StoredMessage, error) {
	_ = ctx
	f.lastListUserID = userID
	f.lastListLimit = limit
	return f.listResp, f.listErr
}

func (f *fakePersistenceRepo) ListInboxBefore(ctx context.Context, userID int, beforeID int64, limit int) ([]StoredMessage, error) {
	_ = ctx
	_ = beforeID
	f.lastListUserID = userID
	f.lastListLimit = limit
	return f.listResp, f.listErr
}

func TestPersistenceService_StoreDirectMessage_DelegatesAndSetsDefaults(t *testing.T) {
	repo := &fakePersistenceRepo{}
	svc := NewPersistenceService(repo)

	got, err := svc.StoreDirectMessage(context.Background(), PersistDirectMessageRequest{
		FromUserID: 1,
		ToUserID:   2,
		Body:       "hello",
	})
	if err != nil {
		t.Fatalf("StoreDirectMessage error: %v", err)
	}
	if repo.lastSave.FromUserID != 1 || repo.lastSave.ToUserID != 2 || repo.lastSave.Body != "hello" {
		t.Fatalf("unexpected save payload: %+v", repo.lastSave)
	}
	if got.ID == 0 {
		t.Fatal("expected persisted ID")
	}
}

func TestPersistenceService_MarkDeliveredAndRead_Delegates(t *testing.T) {
	repo := &fakePersistenceRepo{}
	svc := NewPersistenceService(repo)

	if err := svc.MarkDelivered(context.Background(), 10); err != nil {
		t.Fatalf("MarkDelivered error: %v", err)
	}
	if repo.lastDelivered != 10 {
		t.Fatalf("lastDelivered = %d, want 10", repo.lastDelivered)
	}

	if err := svc.MarkRead(context.Background(), 10); err != nil {
		t.Fatalf("MarkRead error: %v", err)
	}
	if repo.lastRead != 10 {
		t.Fatalf("lastRead = %d, want 10", repo.lastRead)
	}
}

func TestPersistenceService_MarkDeliveredAndReadForRecipient_Delegates(t *testing.T) {
	repo := &fakePersistenceRepo{}
	svc := NewPersistenceService(repo)

	if err := svc.MarkDeliveredForRecipient(context.Background(), 2, 10); err != nil {
		t.Fatalf("MarkDeliveredForRecipient error: %v", err)
	}
	if repo.lastDeliveredRecipientID != 2 || repo.lastDeliveredRecipientMsgID != 10 {
		t.Fatalf("unexpected delivered recipient call user=%d msg=%d", repo.lastDeliveredRecipientID, repo.lastDeliveredRecipientMsgID)
	}

	if err := svc.MarkReadForRecipient(context.Background(), 2, 10); err != nil {
		t.Fatalf("MarkReadForRecipient error: %v", err)
	}
	if repo.lastReadRecipientID != 2 || repo.lastReadRecipientMsgID != 10 {
		t.Fatalf("unexpected read recipient call user=%d msg=%d", repo.lastReadRecipientID, repo.lastReadRecipientMsgID)
	}
}

func TestPersistenceService_ListInbox_UsesDefaultLimitAndPropagatesErrors(t *testing.T) {
	repo := &fakePersistenceRepo{listErr: errors.New("db down")}
	svc := NewPersistenceService(repo)

	_, err := svc.ListInbox(context.Background(), 2, 0)
	if err == nil {
		t.Fatal("expected error")
	}
	if repo.lastListUserID != 2 {
		t.Fatalf("lastListUserID = %d, want 2", repo.lastListUserID)
	}
	if repo.lastListLimit != 100 {
		t.Fatalf("default limit = %d, want 100", repo.lastListLimit)
	}
}
