package messaging

import (
	"context"
	"time"
)

type PersistDirectMessageRequest struct {
	FromUserID int
	ToUserID   int
	Body       string
}

type MessageRepository interface {
	SaveDirectMessage(ctx context.Context, msg StoredMessage) (StoredMessage, error)
	MarkDelivered(ctx context.Context, messageID int64, deliveredAt time.Time) error
	MarkDeliveredForRecipient(ctx context.Context, recipientUserID int, messageID int64, deliveredAt time.Time) error
	MarkRead(ctx context.Context, messageID int64, readAt time.Time) error
	MarkReadForRecipient(ctx context.Context, recipientUserID int, messageID int64, readAt time.Time) error
	GetMessageForRecipient(ctx context.Context, recipientUserID int, messageID int64) (StoredMessage, error)
	ListInbox(ctx context.Context, userID int, limit int) ([]StoredMessage, error)
	ListOutbox(ctx context.Context, userID int, limit int) ([]StoredMessage, error)
	ListUnreadInbox(ctx context.Context, userID int, limit int) ([]StoredMessage, error)
	ListInboxWithUser(ctx context.Context, userID int, withUserID int, limit int) ([]StoredMessage, error)
	ListUnreadInboxWithUser(ctx context.Context, userID int, withUserID int, limit int) ([]StoredMessage, error)
	ListInboxBefore(ctx context.Context, userID int, beforeID int64, limit int) ([]StoredMessage, error)
	ListUnreadInboxBefore(ctx context.Context, userID int, beforeID int64, limit int) ([]StoredMessage, error)
	ListInboxBeforeWithUser(ctx context.Context, userID int, withUserID int, beforeID int64, limit int) ([]StoredMessage, error)
	ListUnreadInboxBeforeWithUser(ctx context.Context, userID int, withUserID int, beforeID int64, limit int) ([]StoredMessage, error)
	ListInboxAfter(ctx context.Context, userID int, afterID int64, limit int) ([]StoredMessage, error)
	ListUnreadInboxAfter(ctx context.Context, userID int, afterID int64, limit int) ([]StoredMessage, error)
	ListInboxAfterWithUser(ctx context.Context, userID int, withUserID int, afterID int64, limit int) ([]StoredMessage, error)
	ListUnreadInboxAfterWithUser(ctx context.Context, userID int, withUserID int, afterID int64, limit int) ([]StoredMessage, error)
}

type PersistenceService interface {
	StoreDirectMessage(ctx context.Context, req PersistDirectMessageRequest) (StoredMessage, error)
	MarkDelivered(ctx context.Context, messageID int64) error
	MarkDeliveredForRecipient(ctx context.Context, recipientUserID int, messageID int64) error
	MarkRead(ctx context.Context, messageID int64) error
	MarkReadForRecipient(ctx context.Context, recipientUserID int, messageID int64) error
	GetMessageForRecipient(ctx context.Context, recipientUserID int, messageID int64) (StoredMessage, error)
	ListInbox(ctx context.Context, userID int, limit int) ([]StoredMessage, error)
	ListOutbox(ctx context.Context, userID int, limit int) ([]StoredMessage, error)
	ListUnreadInbox(ctx context.Context, userID int, limit int) ([]StoredMessage, error)
	ListInboxWithUser(ctx context.Context, userID int, withUserID int, limit int) ([]StoredMessage, error)
	ListUnreadInboxWithUser(ctx context.Context, userID int, withUserID int, limit int) ([]StoredMessage, error)
	ListInboxBefore(ctx context.Context, userID int, beforeID int64, limit int) ([]StoredMessage, error)
	ListUnreadInboxBefore(ctx context.Context, userID int, beforeID int64, limit int) ([]StoredMessage, error)
	ListInboxBeforeWithUser(ctx context.Context, userID int, withUserID int, beforeID int64, limit int) ([]StoredMessage, error)
	ListUnreadInboxBeforeWithUser(ctx context.Context, userID int, withUserID int, beforeID int64, limit int) ([]StoredMessage, error)
	ListInboxAfter(ctx context.Context, userID int, afterID int64, limit int) ([]StoredMessage, error)
	ListUnreadInboxAfter(ctx context.Context, userID int, afterID int64, limit int) ([]StoredMessage, error)
	ListInboxAfterWithUser(ctx context.Context, userID int, withUserID int, afterID int64, limit int) ([]StoredMessage, error)
	ListUnreadInboxAfterWithUser(ctx context.Context, userID int, withUserID int, afterID int64, limit int) ([]StoredMessage, error)
}

type persistenceService struct {
	repo MessageRepository
	now  func() time.Time
}

func NewPersistenceService(repo MessageRepository) PersistenceService {
	return &persistenceService{
		repo: repo,
		now:  time.Now,
	}
}

func (s *persistenceService) StoreDirectMessage(ctx context.Context, req PersistDirectMessageRequest) (StoredMessage, error) {
	return s.repo.SaveDirectMessage(ctx, StoredMessage{
		FromUserID: req.FromUserID,
		ToUserID:   req.ToUserID,
		Body:       req.Body,
	})
}

func (s *persistenceService) MarkDelivered(ctx context.Context, messageID int64) error {
	return s.repo.MarkDelivered(ctx, messageID, s.now().UTC())
}

func (s *persistenceService) MarkRead(ctx context.Context, messageID int64) error {
	return s.repo.MarkRead(ctx, messageID, s.now().UTC())
}

func (s *persistenceService) MarkDeliveredForRecipient(ctx context.Context, recipientUserID int, messageID int64) error {
	return s.repo.MarkDeliveredForRecipient(ctx, recipientUserID, messageID, s.now().UTC())
}

func (s *persistenceService) MarkReadForRecipient(ctx context.Context, recipientUserID int, messageID int64) error {
	return s.repo.MarkReadForRecipient(ctx, recipientUserID, messageID, s.now().UTC())
}

func (s *persistenceService) GetMessageForRecipient(ctx context.Context, recipientUserID int, messageID int64) (StoredMessage, error) {
	return s.repo.GetMessageForRecipient(ctx, recipientUserID, messageID)
}

func (s *persistenceService) ListInbox(ctx context.Context, userID int, limit int) ([]StoredMessage, error) {
	if limit <= 0 {
		limit = 100
	}
	return s.repo.ListInbox(ctx, userID, limit)
}

func (s *persistenceService) ListOutbox(ctx context.Context, userID int, limit int) ([]StoredMessage, error) {
	if limit <= 0 {
		limit = 100
	}
	return s.repo.ListOutbox(ctx, userID, limit)
}

func (s *persistenceService) ListUnreadInbox(ctx context.Context, userID int, limit int) ([]StoredMessage, error) {
	if limit <= 0 {
		limit = 100
	}
	return s.repo.ListUnreadInbox(ctx, userID, limit)
}

func (s *persistenceService) ListInboxBefore(ctx context.Context, userID int, beforeID int64, limit int) ([]StoredMessage, error) {
	if limit <= 0 {
		limit = 100
	}
	return s.repo.ListInboxBefore(ctx, userID, beforeID, limit)
}

func (s *persistenceService) ListInboxWithUser(ctx context.Context, userID int, withUserID int, limit int) ([]StoredMessage, error) {
	if limit <= 0 {
		limit = 100
	}
	return s.repo.ListInboxWithUser(ctx, userID, withUserID, limit)
}

func (s *persistenceService) ListUnreadInboxWithUser(ctx context.Context, userID int, withUserID int, limit int) ([]StoredMessage, error) {
	if limit <= 0 {
		limit = 100
	}
	return s.repo.ListUnreadInboxWithUser(ctx, userID, withUserID, limit)
}

func (s *persistenceService) ListUnreadInboxBefore(ctx context.Context, userID int, beforeID int64, limit int) ([]StoredMessage, error) {
	if limit <= 0 {
		limit = 100
	}
	return s.repo.ListUnreadInboxBefore(ctx, userID, beforeID, limit)
}

func (s *persistenceService) ListInboxBeforeWithUser(ctx context.Context, userID int, withUserID int, beforeID int64, limit int) ([]StoredMessage, error) {
	if limit <= 0 {
		limit = 100
	}
	return s.repo.ListInboxBeforeWithUser(ctx, userID, withUserID, beforeID, limit)
}

func (s *persistenceService) ListUnreadInboxBeforeWithUser(ctx context.Context, userID int, withUserID int, beforeID int64, limit int) ([]StoredMessage, error) {
	if limit <= 0 {
		limit = 100
	}
	return s.repo.ListUnreadInboxBeforeWithUser(ctx, userID, withUserID, beforeID, limit)
}

func (s *persistenceService) ListInboxAfter(ctx context.Context, userID int, afterID int64, limit int) ([]StoredMessage, error) {
	if limit <= 0 {
		limit = 100
	}
	return s.repo.ListInboxAfter(ctx, userID, afterID, limit)
}

func (s *persistenceService) ListUnreadInboxAfter(ctx context.Context, userID int, afterID int64, limit int) ([]StoredMessage, error) {
	if limit <= 0 {
		limit = 100
	}
	return s.repo.ListUnreadInboxAfter(ctx, userID, afterID, limit)
}

func (s *persistenceService) ListInboxAfterWithUser(ctx context.Context, userID int, withUserID int, afterID int64, limit int) ([]StoredMessage, error) {
	if limit <= 0 {
		limit = 100
	}
	return s.repo.ListInboxAfterWithUser(ctx, userID, withUserID, afterID, limit)
}

func (s *persistenceService) ListUnreadInboxAfterWithUser(ctx context.Context, userID int, withUserID int, afterID int64, limit int) ([]StoredMessage, error) {
	if limit <= 0 {
		limit = 100
	}
	return s.repo.ListUnreadInboxAfterWithUser(ctx, userID, withUserID, afterID, limit)
}
