package messaging

import (
	"context"
	"time"
)

// ThreadSummary is the durable thread-list projection used for reconnect/bootstrap UI.
type ThreadSummary struct {
	CounterpartyUserID      int
	CounterpartyUsername    string
	CounterpartyDisplayName string
	CounterpartyAvatarURL   string
	LastMessageID           int64
	LastMessageFromUserID   int
	LastMessageToUserID     int
	LastMessageBody         string
	LastMessageCreatedAt    time.Time
	LastDeliveredAt         *time.Time
	LastReadAt              *time.Time
	UnreadCount             int
}

type ThreadSummaryRepository interface {
	ListThreadSummaries(ctx context.Context, userID int, limit int) ([]ThreadSummary, error)
}

type ThreadSummaryService interface {
	ListThreadSummaries(ctx context.Context, userID int, limit int) ([]ThreadSummary, error)
}

type threadSummaryService struct {
	repo ThreadSummaryRepository
}

func NewThreadSummaryService(repo ThreadSummaryRepository) ThreadSummaryService {
	return &threadSummaryService{repo: repo}
}

func (s *threadSummaryService) ListThreadSummaries(ctx context.Context, userID int, limit int) ([]ThreadSummary, error) {
	if limit <= 0 {
		limit = 100
	}
	return s.repo.ListThreadSummaries(ctx, userID, limit)
}
