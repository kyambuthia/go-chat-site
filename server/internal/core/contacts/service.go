package contacts

import "context"

type Service interface {
	ListContacts(ctx context.Context, userID UserID) ([]Contact, error)
	SendInvite(ctx context.Context, fromUser, toUser UserID) error
	ListInvites(ctx context.Context, userID UserID) ([]Invite, error)
	RespondToInvite(ctx context.Context, inviteID int, userID UserID, status InviteStatus) error
}

type CoreService struct {
	repo GraphRepository
}

func NewService(repo GraphRepository) *CoreService {
	return &CoreService{repo: repo}
}

func (s *CoreService) ListContacts(ctx context.Context, userID UserID) ([]Contact, error) {
	return s.repo.ListContacts(ctx, userID)
}

func (s *CoreService) SendInvite(ctx context.Context, fromUser, toUser UserID) error {
	return s.repo.CreateInvite(ctx, fromUser, toUser)
}

func (s *CoreService) ListInvites(ctx context.Context, userID UserID) ([]Invite, error) {
	return s.repo.ListInvites(ctx, userID)
}

func (s *CoreService) RespondToInvite(ctx context.Context, inviteID int, userID UserID, status InviteStatus) error {
	return s.repo.UpdateInvite(ctx, inviteID, userID, status)
}
