package passwordbcrypt

import (
	"testing"

	"github.com/kyambuthia/go-chat-site/server/internal/crypto"
)

func TestVerifier_VerifyPassword(t *testing.T) {
	hash, err := crypto.HashPassword("password123")
	if err != nil {
		t.Fatalf("HashPassword error: %v", err)
	}

	v := Verifier{}
	if !v.VerifyPassword("password123", hash) {
		t.Fatal("expected password verification to succeed")
	}
	if v.VerifyPassword("wrong-password", hash) {
		t.Fatal("expected password verification to fail")
	}
}
