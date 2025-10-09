package auth

import (
	"context"
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

		tokenString := strings.TrimPrefix(authHeader, "Bearer ")
		if tokenString == authHeader {
			web.JSONError(w, errors.New("could not find bearer token in authorization header"), http.StatusUnauthorized)
			return
		}

		claims, err := ValidateToken(tokenString)
		if err != nil {
			web.JSONError(w, errors.New("invalid token"), http.StatusUnauthorized)
			return
		}

		ctx := context.WithValue(r.Context(), "userID", claims.UserID)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}