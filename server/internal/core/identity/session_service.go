package identity

import (
	"context"
	"errors"
)

var (
	ErrInvalidRefreshToken = errors.New("invalid refresh token")
	ErrRefreshTokenReplay  = errors.New("refresh token replay detected")
	ErrSessionNotFound     = errors.New("session not found")
)

type SessionService interface {
	RefreshSession(ctx context.Context, refreshToken string, meta SessionMetadata) (SessionTokens, error)
	ListSessions(ctx context.Context, userID UserID) ([]Session, error)
	RevokeSession(ctx context.Context, actorUserID UserID, sessionID int64) error
}

type sessionService struct {
	tokens TokenService
}

func NewSessionService(tokens TokenService) SessionService {
	return &sessionService{tokens: tokens}
}

func (s *sessionService) RefreshSession(ctx context.Context, refreshToken string, meta SessionMetadata) (SessionTokens, error) {
	return s.tokens.RefreshSession(ctx, refreshToken, meta)
}

func (s *sessionService) ListSessions(ctx context.Context, userID UserID) ([]Session, error) {
	return s.tokens.ListSessions(ctx, userID)
}

func (s *sessionService) RevokeSession(ctx context.Context, actorUserID UserID, sessionID int64) error {
	return s.tokens.RevokeSession(ctx, actorUserID, sessionID)
}
