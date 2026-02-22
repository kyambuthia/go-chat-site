package jwttokens

import (
	"context"
	"testing"

	coreid "github.com/kyambuthia/go-chat-site/server/internal/core/identity"
	"github.com/kyambuthia/go-chat-site/server/internal/auth"
)

func TestAdapter_IssueAndValidateToken(t *testing.T) {
	if err := auth.ConfigureJWT("test-secret-123456"); err != nil {
		t.Fatal(err)
	}

	a := &Adapter{}
	token, err := a.IssueToken(context.Background(), coreid.Principal{ID: 5, Username: "alice"})
	if err != nil {
		t.Fatalf("IssueToken error: %v", err)
	}

	claims, err := a.ValidateToken(context.Background(), token)
	if err != nil {
		t.Fatalf("ValidateToken error: %v", err)
	}
	if claims.SubjectUserID != 5 {
		t.Fatalf("SubjectUserID = %d, want 5", claims.SubjectUserID)
	}
}

func TestAdapter_ValidateToken_FailsForInvalidToken(t *testing.T) {
	if err := auth.ConfigureJWT("test-secret-123456"); err != nil {
		t.Fatal(err)
	}

	a := &Adapter{}
	if _, err := a.ValidateToken(context.Background(), "bad-token"); err == nil {
		t.Fatal("expected invalid token error")
	}
}
