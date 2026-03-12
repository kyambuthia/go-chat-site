package messaging

import (
	"context"
	"errors"
)

// SyncCursor is the initial additive cursor model for durable message sync.
type SyncCursor struct {
	AfterID     int64
	NextAfterID int64
}

// SyncResult carries the merged durable message stream for reconnect polling.
type SyncResult struct {
	Cursor   SyncCursor
	Messages []StoredMessage
	HasMore  bool
}

type SyncService struct {
	persistence PersistenceService
}

func NewSyncService(persistence PersistenceService) *SyncService {
	return &SyncService{persistence: persistence}
}

func (s *SyncService) Sync(ctx context.Context, userID int, afterID int64, limit int) (SyncResult, error) {
	if s == nil || s.persistence == nil {
		return SyncResult{}, errors.New("messaging sync unavailable")
	}
	if limit <= 0 {
		limit = 100
	}

	fetchLimit := limit + 1
	inbox, err := s.persistence.ListInboxAfter(ctx, userID, afterID, fetchLimit)
	if err != nil {
		return SyncResult{}, err
	}
	outbox, err := s.persistence.ListOutboxAfter(ctx, userID, afterID, fetchLimit)
	if err != nil {
		return SyncResult{}, err
	}

	merged := mergeStoredMessagesAscending(inbox, outbox)
	hasMore := len(merged) > limit
	if hasMore {
		merged = merged[:limit]
	}

	nextAfterID := afterID
	if len(merged) > 0 {
		nextAfterID = merged[len(merged)-1].ID
	}

	return SyncResult{
		Cursor: SyncCursor{
			AfterID:     afterID,
			NextAfterID: nextAfterID,
		},
		Messages: merged,
		HasMore:  hasMore,
	}, nil
}

func mergeStoredMessagesAscending(inbox, outbox []StoredMessage) []StoredMessage {
	merged := make([]StoredMessage, 0, len(inbox)+len(outbox))
	inboxIndex := 0
	outboxIndex := 0

	for inboxIndex < len(inbox) && outboxIndex < len(outbox) {
		switch {
		case inbox[inboxIndex].ID < outbox[outboxIndex].ID:
			merged = append(merged, inbox[inboxIndex])
			inboxIndex++
		case outbox[outboxIndex].ID < inbox[inboxIndex].ID:
			merged = append(merged, outbox[outboxIndex])
			outboxIndex++
		default:
			merged = append(merged, inbox[inboxIndex])
			inboxIndex++
			outboxIndex++
		}
	}

	merged = append(merged, inbox[inboxIndex:]...)
	merged = append(merged, outbox[outboxIndex:]...)
	return merged
}
