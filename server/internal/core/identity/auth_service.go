package identity

import (
	"context"
	"errors"
	"strings"
)

var ErrInvalidCredentials = errors.New("invalid credentials")

type PasswordLoginRecord struct {
	Principal    Principal
	PasswordHash string
}

type AuthRepository interface {
	CreateUser(ctx context.Context, cred PasswordCredential) (Principal, error)
	GetPasswordLoginRecord(ctx context.Context, username string) (PasswordLoginRecord, error)
}

type PasswordVerifier interface {
	VerifyPassword(password, hash string) bool
}

type AuthService interface {
	RegisterPassword(ctx context.Context, cred PasswordCredential) (Principal, error)
	LoginPassword(ctx context.Context, cred PasswordCredential) (string, error)
}

type authService struct {
	repo     AuthRepository
	verifier PasswordVerifier
	tokens   TokenService
}

func NewAuthService(repo AuthRepository, verifier PasswordVerifier, tokens TokenService) AuthService {
	return &authService{repo: repo, verifier: verifier, tokens: tokens}
}

func (s *authService) RegisterPassword(ctx context.Context, cred PasswordCredential) (Principal, error) {
	cred.Username = strings.TrimSpace(cred.Username)
	return s.repo.CreateUser(ctx, cred)
}

func (s *authService) LoginPassword(ctx context.Context, cred PasswordCredential) (string, error) {
	record, err := s.repo.GetPasswordLoginRecord(ctx, strings.TrimSpace(cred.Username))
	if err != nil {
		return "", ErrInvalidCredentials
	}
	if s.verifier == nil || !s.verifier.VerifyPassword(cred.Password, record.PasswordHash) {
		return "", ErrInvalidCredentials
	}
	return s.tokens.IssueToken(ctx, record.Principal)
}
