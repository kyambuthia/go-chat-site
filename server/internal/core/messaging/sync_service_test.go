package messaging

import (
	"context"
	"errors"
	"testing"
	"time"
)

type fakeSyncPersistence struct {
	fakePersistenceService
	inboxAfter      []StoredMessage
	outboxAfter     []StoredMessage
	inboxAfterErr   error
	outboxAfterErr  error
	lastUserID      int
	lastAfterID     int64
	lastInboxLimit  int
	lastOutboxLimit int
}

func (f *fakeSyncPersistence) ListInboxAfter(ctx context.Context, userID int, afterID int64, limit int) ([]StoredMessage, error) {
	_ = ctx
	f.lastUserID = userID
	f.lastAfterID = afterID
	f.lastInboxLimit = limit
	return f.inboxAfter, f.inboxAfterErr
}

func (f *fakeSyncPersistence) ListOutboxAfter(ctx context.Context, userID int, afterID int64, limit int) ([]StoredMessage, error) {
	_ = ctx
	f.lastUserID = userID
	f.lastAfterID = afterID
	f.lastOutboxLimit = limit
	return f.outboxAfter, f.outboxAfterErr
}

func TestSyncService_Sync_MergesInboxAndOutboxByAscendingID(t *testing.T) {
	persistence := &fakeSyncPersistence{
		inboxAfter: []StoredMessage{
			{ID: 11, FromUserID: 1, ToUserID: 2, Body: "in-11", CreatedAt: time.Unix(11, 0).UTC()},
			{ID: 14, FromUserID: 1, ToUserID: 2, Body: "in-14", CreatedAt: time.Unix(14, 0).UTC()},
		},
		outboxAfter: []StoredMessage{
			{ID: 12, FromUserID: 2, ToUserID: 1, Body: "out-12", CreatedAt: time.Unix(12, 0).UTC()},
			{ID: 13, FromUserID: 2, ToUserID: 1, Body: "out-13", CreatedAt: time.Unix(13, 0).UTC()},
		},
	}

	result, err := NewSyncService(persistence).Sync(context.Background(), 2, 10, 3)
	if err != nil {
		t.Fatalf("Sync returned error: %v", err)
	}

	if persistence.lastUserID != 2 || persistence.lastAfterID != 10 {
		t.Fatalf("unexpected sync query user=%d after=%d", persistence.lastUserID, persistence.lastAfterID)
	}
	if persistence.lastInboxLimit != 4 || persistence.lastOutboxLimit != 4 {
		t.Fatalf("expected fetch limit 4, got inbox=%d outbox=%d", persistence.lastInboxLimit, persistence.lastOutboxLimit)
	}
	if len(result.Messages) != 3 {
		t.Fatalf("messages len = %d, want 3", len(result.Messages))
	}
	if result.Messages[0].ID != 11 || result.Messages[1].ID != 12 || result.Messages[2].ID != 13 {
		t.Fatalf("unexpected merged order: %+v", result.Messages)
	}
	if !result.HasMore {
		t.Fatal("expected has_more to be true")
	}
	if result.Cursor.AfterID != 10 || result.Cursor.NextAfterID != 13 {
		t.Fatalf("unexpected cursor: %+v", result.Cursor)
	}
}

func TestSyncService_Sync_PropagatesPersistenceErrors(t *testing.T) {
	persistence := &fakeSyncPersistence{inboxAfterErr: errors.New("db down")}

	if _, err := NewSyncService(persistence).Sync(context.Background(), 2, 10, 10); err == nil {
		t.Fatal("expected sync error")
	}
}
