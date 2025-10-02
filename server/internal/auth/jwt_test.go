package auth

import (
	"testing"
	"time"
)

func TestGenerateAndValidateToken(t *testing.T) {
	userID := 1

	token, err := GenerateToken(userID)
	if err != nil {
		t.Fatal(err)
	}

	claims, err := ValidateToken(token)
	if err != nil {
		t.Fatal(err)
	}

	if claims.UserID != userID {
		t.Fatalf("expected userID %d, got %d", userID, claims.UserID)
	}

	if claims.ExpiresAt.Time.Before(time.Now()) {
		t.Fatal("token should not be expired")
	}
}
