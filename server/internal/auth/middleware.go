package auth

import (
	"errors"
	"net/http"
	"strings"

	"github.com/kyambuthia/go-chat-site/server/internal/web"
)

func Middleware(next http.Handler) http.Handler {
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

		claims, err := ValidateToken(tokenString)
		if err != nil {
			web.JSONError(w, errors.New("invalid token"), http.StatusUnauthorized)
			return
		}

		ctx := WithUserID(r.Context(), claims.UserID)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}
