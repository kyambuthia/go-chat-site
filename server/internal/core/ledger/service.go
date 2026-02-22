package ledger

import (
	"context"
	"errors"
	"strings"
	"time"
)

var ErrRecipientNotFound = errors.New("recipient not found")

type UserDirectory interface {
	ResolveUserIDByUsername(ctx context.Context, username string) (int, error)
}

type Service interface {
	GetAccount(ctx context.Context, userID int) (Account, error)
	SendTransferByUsername(ctx context.Context, fromUserID int, recipientUsername string, amountCents int64) (Transfer, error)
}

type CoreService struct {
	repo  Repository
	users UserDirectory
	now   func() time.Time
}

func NewService(repo Repository, users UserDirectory) *CoreService {
	return &CoreService{
		repo:  repo,
		users: users,
		now:   time.Now,
	}
}

func (s *CoreService) GetAccount(ctx context.Context, userID int) (Account, error) {
	return s.repo.GetAccount(ctx, userID)
}

func (s *CoreService) SendTransferByUsername(ctx context.Context, fromUserID int, recipientUsername string, amountCents int64) (Transfer, error) {
	toUserID, err := s.users.ResolveUserIDByUsername(ctx, strings.TrimSpace(recipientUsername))
	if err != nil {
		return Transfer{}, ErrRecipientNotFound
	}

	transfer := Transfer{
		FromUserID:   fromUserID,
		ToUserID:     toUserID,
		AmountCents:  amountCents,
		CurrencyCode: "USD",
		CreatedAt:    s.now(),
	}
	return s.repo.Transfer(ctx, transfer)
}
