package auth

import "context"

type contextKey string

const userIDContextKey contextKey = "user_id"
const sessionIDContextKey contextKey = "session_id"

func WithUserID(ctx context.Context, userID int) context.Context {
	return context.WithValue(ctx, userIDContextKey, userID)
}

func UserIDFromContext(ctx context.Context) (int, bool) {
	userID, ok := ctx.Value(userIDContextKey).(int)
	return userID, ok
}

func WithSessionID(ctx context.Context, sessionID int64) context.Context {
	return context.WithValue(ctx, sessionIDContextKey, sessionID)
}

func SessionIDFromContext(ctx context.Context) (int64, bool) {
	sessionID, ok := ctx.Value(sessionIDContextKey).(int64)
	return sessionID, ok
}
