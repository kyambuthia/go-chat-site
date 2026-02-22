package auth

import (
	"strings"
	"testing"
	"time"
)

func TestGenerateAndValidateToken(t *testing.T) {
	if err := ConfigureJWT("test-secret-123456"); err != nil {
		t.Fatal(err)
	}

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

func TestConfigureJWT_RejectsMissingOrShortSecret(t *testing.T) {
	t.Cleanup(func() {
		jwtKey = nil
	})

	if err := ConfigureJWT(""); err == nil {
		t.Fatal("expected missing secret to fail")
	}

	if err := ConfigureJWT("short-secret"); err == nil {
		t.Fatal("expected short secret to fail")
	}
}

func TestGenerateAndValidateToken_FailsWhenJWTNotConfigured(t *testing.T) {
	original := jwtKey
	jwtKey = nil
	t.Cleanup(func() {
		jwtKey = original
	})

	if _, err := GenerateToken(42); err == nil || !strings.Contains(err.Error(), "not configured") {
		t.Fatalf("expected GenerateToken to fail when jwt secret missing, got %v", err)
	}

	if _, err := ValidateToken("anything"); err == nil || !strings.Contains(err.Error(), "not configured") {
		t.Fatalf("expected ValidateToken to fail when jwt secret missing, got %v", err)
	}
}

func TestValidateToken_FailsWithDifferentSecret(t *testing.T) {
	if err := ConfigureJWT("test-secret-123456"); err != nil {
		t.Fatal(err)
	}

	token, err := GenerateToken(7)
	if err != nil {
		t.Fatal(err)
	}

	if err := ConfigureJWT("different-secret-123"); err != nil {
		t.Fatal(err)
	}

	if _, err := ValidateToken(token); err == nil {
		t.Fatal("expected token validation to fail with different secret")
	}
}
