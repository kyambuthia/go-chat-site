package passwordbcrypt

import (
	"github.com/kyambuthia/go-chat-site/server/internal/crypto"
)

type Verifier struct{}

func (Verifier) VerifyPassword(password, hash string) bool {
	return crypto.CheckPasswordHash(password, hash)
}
