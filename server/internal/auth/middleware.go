package auth

import (
	"context"
	"errors"
	"net"
	"net/http"
	"strings"

	coreid "github.com/kyambuthia/go-chat-site/server/internal/core/identity"
	"github.com/kyambuthia/go-chat-site/server/internal/web"
)

type AccessTokenService interface {
	ValidateToken(ctx context.Context, token string) (coreid.TokenClaims, error)
	TouchSession(ctx context.Context, sessionID int64, meta coreid.SessionMetadata) error
}

func Middleware(tokens AccessTokenService) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			authHeader := r.Header.Get("Authorization")
			if authHeader == "" {
				web.JSONError(w, errors.New("authorization header required"), http.StatusUnauthorized)
				return
			}

			tokenString := strings.TrimSpace(strings.TrimPrefix(authHeader, "Bearer "))
			if tokenString == "" || tokenString == authHeader {
				web.JSONError(w, errors.New("could not find bearer token in authorization header"), http.StatusUnauthorized)
				return
			}
			if tokens == nil {
				web.JSONError(w, errors.New("auth unavailable"), http.StatusServiceUnavailable)
				return
			}

			claims, err := tokens.ValidateToken(r.Context(), tokenString)
			if err != nil {
				web.JSONError(w, errors.New("invalid token"), http.StatusUnauthorized)
				return
			}

			if claims.SessionID > 0 {
				_ = tokens.TouchSession(r.Context(), claims.SessionID, coreid.SessionMetadata{
					UserAgent: r.UserAgent(),
					IPAddress: clientIPFromRequest(r),
				})
			}

			ctx := WithUserID(r.Context(), int(claims.SubjectUserID))
			if claims.SessionID > 0 {
				ctx = WithSessionID(ctx, claims.SessionID)
			}
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

func clientIPFromRequest(r *http.Request) string {
	if xff := strings.TrimSpace(r.Header.Get("X-Forwarded-For")); xff != "" {
		parts := strings.Split(xff, ",")
		if len(parts) > 0 {
			if v := strings.TrimSpace(parts[0]); v != "" {
				return v
			}
		}
	}

	host, _, err := net.SplitHostPort(strings.TrimSpace(r.RemoteAddr))
	if err == nil && host != "" {
		return host
	}
	if ra := strings.TrimSpace(r.RemoteAddr); ra != "" {
		return ra
	}
	return "unknown"
}
