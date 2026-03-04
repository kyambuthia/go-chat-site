package auth

import (
	"context"
	"testing"
)

func TestWithUserIDAndUserIDFromContext(t *testing.T) {
	ctx := WithUserID(context.Background(), 42)
	userID, ok := UserIDFromContext(ctx)
	if !ok {
		t.Fatal("expected user id in context")
	}
	if userID != 42 {
		t.Fatalf("userID = %d, want 42", userID)
	}
}

func TestUserIDFromContext_Missing(t *testing.T) {
	if _, ok := UserIDFromContext(context.Background()); ok {
		t.Fatal("expected missing user id")
	}
}
