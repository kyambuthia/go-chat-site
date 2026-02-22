package identity

import "context"

// UserID is the internal stable identifier used by the current centralized backend.
type UserID int

// Principal models an authenticated actor independent of credential type.
type Principal struct {
	ID       UserID
	Username string
}

// PasswordCredential represents the current username/password auth mechanism.
type PasswordCredential struct {
	Username string
	Password string
}

// TokenClaims is the normalized auth token payload used by adapters.
type TokenClaims struct {
	SubjectUserID UserID
}

// Authenticator validates credentials and returns a principal.
type Authenticator interface {
	AuthenticatePassword(ctx context.Context, cred PasswordCredential) (Principal, error)
}

// TokenService abstracts token creation/validation (today: JWT, future: DID/WebAuthn assertions).
type TokenService interface {
	IssueToken(ctx context.Context, principal Principal) (string, error)
	ValidateToken(ctx context.Context, token string) (TokenClaims, error)
}

// KeyMaterialProvider abstracts future DID/WebAuthn/passkey key resolution.
type KeyMaterialProvider interface {
	ResolveSigningKey(ctx context.Context, principal Principal, keyRef string) ([]byte, error)
}
