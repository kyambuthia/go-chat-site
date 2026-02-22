package contacts

import (
	"context"
	"errors"
	"strings"
)

type Service interface {
	ListContacts(ctx context.Context, userID UserID) ([]Contact, error)
	AddContactByUsername(ctx context.Context, userID UserID, username string) error
	RemoveContact(ctx context.Context, userID, contactID UserID) error
	SendInvite(ctx context.Context, fromUser, toUser UserID) error
	SendInviteByUsername(ctx context.Context, fromUser UserID, username string) error
	ListInvites(ctx context.Context, userID UserID) ([]Invite, error)
	RespondToInvite(ctx context.Context, inviteID int, userID UserID, status InviteStatus) error
}

type CoreService struct {
	repo  GraphRepository
	users UserDirectory
}

type UserDirectory interface {
	ResolveUserIDByUsername(ctx context.Context, username string) (UserID, error)
}

func NewService(repo GraphRepository, users UserDirectory) *CoreService {
	return &CoreService{repo: repo, users: users}
}

func (s *CoreService) ListContacts(ctx context.Context, userID UserID) ([]Contact, error) {
	return s.repo.ListContacts(ctx, userID)
}

func (s *CoreService) SendInvite(ctx context.Context, fromUser, toUser UserID) error {
	return s.repo.CreateInvite(ctx, fromUser, toUser)
}

func (s *CoreService) AddContactByUsername(ctx context.Context, userID UserID, username string) error {
	if s.users == nil {
		return ErrUserNotFound
	}
	contactID, err := s.users.ResolveUserIDByUsername(ctx, strings.TrimSpace(username))
	if err != nil {
		return ErrUserNotFound
	}
	return s.repo.AddContact(ctx, userID, contactID)
}

func (s *CoreService) RemoveContact(ctx context.Context, userID, contactID UserID) error {
	return s.repo.RemoveContact(ctx, userID, contactID)
}

func (s *CoreService) SendInviteByUsername(ctx context.Context, fromUser UserID, username string) error {
	if s.users == nil {
		return ErrUserNotFound
	}
	toUser, err := s.users.ResolveUserIDByUsername(ctx, strings.TrimSpace(username))
	if err != nil {
		return ErrUserNotFound
	}
	if err := s.repo.CreateInvite(ctx, fromUser, toUser); err != nil {
		return err
	}
	return nil
}

func (s *CoreService) ListInvites(ctx context.Context, userID UserID) ([]Invite, error) {
	return s.repo.ListInvites(ctx, userID)
}

func (s *CoreService) RespondToInvite(ctx context.Context, inviteID int, userID UserID, status InviteStatus) error {
	err := s.repo.UpdateInvite(ctx, inviteID, userID, status)
	if err == nil {
		return nil
	}
	if errors.Is(err, ErrInviteNotFound) {
		return ErrInviteNotFound
	}
	return err
}
