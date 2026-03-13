package identity

import (
	"context"
	"errors"
	"testing"
)

func TestSessionService_RefreshSession_DelegatesToTokenService(t *testing.T) {
	tokens := &fakeTokenService{tokens: SessionTokens{AccessToken: "access-1", RefreshToken: "refresh-1"}}
	svc := NewSessionService(tokens)

	got, err := svc.RefreshSession(context.Background(), "refresh-token", SessionMetadata{
		DeviceLabel: "This browser",
		UserAgent:   "agent",
		IPAddress:   "127.0.0.1",
	})
	if err != nil {
		t.Fatalf("RefreshSession error: %v", err)
	}
	if got.AccessToken != "access-1" || got.RefreshToken != "refresh-1" {
		t.Fatalf("unexpected tokens: %+v", got)
	}
	if tokens.lastMeta.DeviceLabel != "This browser" {
		t.Fatalf("unexpected metadata: %+v", tokens.lastMeta)
	}
}

func TestSessionService_ListAndRevoke_DelegatesToTokenService(t *testing.T) {
	tokens := &fakeTokenService{}
	svc := NewSessionService(tokens)

	if _, err := svc.ListSessions(context.Background(), 7); err != nil {
		t.Fatalf("ListSessions error: %v", err)
	}
	if err := svc.RevokeSession(context.Background(), 7, 41); err != nil {
		t.Fatalf("RevokeSession error: %v", err)
	}
}

func TestSessionService_PropagatesRefreshErrors(t *testing.T) {
	tokens := &fakeTokenService{err: errors.New("db down")}
	svc := NewSessionService(tokens)

	if _, err := svc.RefreshSession(context.Background(), "bad", SessionMetadata{}); err == nil {
		t.Fatal("expected error")
	}
}
