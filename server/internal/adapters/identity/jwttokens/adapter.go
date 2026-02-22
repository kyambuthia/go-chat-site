package jwttokens

import (
	"context"

	"github.com/kyambuthia/go-chat-site/server/internal/auth"
	coreid "github.com/kyambuthia/go-chat-site/server/internal/core/identity"
)

type Adapter struct{}

var _ coreid.TokenService = (*Adapter)(nil)

func (a *Adapter) IssueToken(ctx context.Context, principal coreid.Principal) (string, error) {
	_ = ctx
	return auth.GenerateToken(int(principal.ID))
}

func (a *Adapter) ValidateToken(ctx context.Context, token string) (coreid.TokenClaims, error) {
	_ = ctx
	claims, err := auth.ValidateToken(token)
	if err != nil {
		return coreid.TokenClaims{}, err
	}
	return coreid.TokenClaims{SubjectUserID: coreid.UserID(claims.UserID)}, nil
}
