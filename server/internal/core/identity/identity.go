package identity

import (
	"context"
	"time"
)

// UserID is the internal stable identifier used by the current centralized backend.
type UserID int

// Principal models an authenticated actor independent of credential type.
type Principal struct {
	ID       UserID
	Username string
}

// Profile is the HTTP-facing account profile shape used by the current /api/me route.
type Profile struct {
	UserID      UserID
	Username    string
	DisplayName string
	AvatarURL   string
}

// ProfileUpdate carries mutable user-facing profile fields.
type ProfileUpdate struct {
	DisplayName string
	AvatarURL   string
}

// PasswordCredential represents the current username/password auth mechanism.
type PasswordCredential struct {
	Username string
	Password string
}

// SessionMetadata carries request/device context used for session creation and refresh rotation.
type SessionMetadata struct {
	DeviceLabel string
	UserAgent   string
	IPAddress   string
}

// Session is the device/session projection exposed to the authenticated account surface.
type Session struct {
	ID                    int64
	UserID                UserID
	DeviceLabel           string
	UserAgent             string
	LastSeenIP            string
	CreatedAt             time.Time
	LastSeenAt            time.Time
	AccessTokenExpiresAt  time.Time
	RefreshTokenExpiresAt time.Time
	RevokedAt             *time.Time
}

// SessionTokens is the access/refresh token bundle returned by login and refresh flows.
type SessionTokens struct {
	AccessToken           string
	RefreshToken          string
	AccessTokenExpiresAt  time.Time
	RefreshTokenExpiresAt time.Time
	Session               Session
}

// TokenClaims is the normalized auth token payload used by adapters.
type TokenClaims struct {
	SubjectUserID UserID
	SessionID     int64
}

// Authenticator validates credentials and returns a principal.
type Authenticator interface {
	AuthenticatePassword(ctx context.Context, cred PasswordCredential) (Principal, error)
}

// TokenService abstracts token creation/validation and session lifecycle.
type TokenService interface {
	IssueSession(ctx context.Context, principal Principal, meta SessionMetadata) (SessionTokens, error)
	RefreshSession(ctx context.Context, refreshToken string, meta SessionMetadata) (SessionTokens, error)
	ValidateToken(ctx context.Context, token string) (TokenClaims, error)
	ListSessions(ctx context.Context, userID UserID) ([]Session, error)
	RevokeSession(ctx context.Context, actorUserID UserID, sessionID int64) error
	TouchSession(ctx context.Context, sessionID int64, meta SessionMetadata) error
}

// KeyMaterialProvider abstracts future DID/WebAuthn/passkey key resolution.
type KeyMaterialProvider interface {
	ResolveSigningKey(ctx context.Context, principal Principal, keyRef string) ([]byte, error)
}
