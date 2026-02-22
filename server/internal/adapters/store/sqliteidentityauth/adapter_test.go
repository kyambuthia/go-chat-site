package sqliteidentityauth

import (
	"context"
	"errors"
	"testing"

	coreid "github.com/kyambuthia/go-chat-site/server/internal/core/identity"
	"github.com/kyambuthia/go-chat-site/server/internal/store"
)

type fakeDeps struct {
	createID  int
	createErr error
	user      *store.User
	userErr   error
	lastCreds struct {
		username string
		password string
	}
	lastLookup string
}

func (f *fakeDeps) CreateUser(username, password string) (int, error) {
	f.lastCreds.username = username
	f.lastCreds.password = password
	return f.createID, f.createErr
}

func (f *fakeDeps) GetUserByUsername(username string) (*store.User, error) {
	f.lastLookup = username
	if f.userErr != nil {
		return nil, f.userErr
	}
	return f.user, nil
}

func TestAdapter_CreateUser_MapsPrincipal(t *testing.T) {
	deps := &fakeDeps{createID: 12}
	a := &Adapter{Store: deps}

	got, err := a.CreateUser(context.Background(), coreid.PasswordCredential{Username: "alice", Password: "password123"})
	if err != nil {
		t.Fatalf("CreateUser error: %v", err)
	}
	if deps.lastCreds.username != "alice" || deps.lastCreds.password != "password123" {
		t.Fatalf("unexpected CreateUser args: %+v", deps.lastCreds)
	}
	if got.ID != 12 || got.Username != "alice" {
		t.Fatalf("unexpected principal: %+v", got)
	}
}

func TestAdapter_GetPasswordLoginRecord_MapsStoreUser(t *testing.T) {
	deps := &fakeDeps{user: &store.User{ID: 7, Username: "alice", PasswordHash: "hash"}}
	a := &Adapter{Store: deps}

	got, err := a.GetPasswordLoginRecord(context.Background(), "alice")
	if err != nil {
		t.Fatalf("GetPasswordLoginRecord error: %v", err)
	}
	if deps.lastLookup != "alice" {
		t.Fatalf("lookup = %q, want alice", deps.lastLookup)
	}
	if got.Principal.ID != 7 || got.Principal.Username != "alice" || got.PasswordHash != "hash" {
		t.Fatalf("unexpected login record: %+v", got)
	}
}

func TestAdapter_GetPasswordLoginRecord_PropagatesErrors(t *testing.T) {
	wantErr := errors.New("db down")
	a := &Adapter{Store: &fakeDeps{userErr: wantErr}}

	_, err := a.GetPasswordLoginRecord(context.Background(), "alice")
	if !errors.Is(err, wantErr) {
		t.Fatalf("expected %v, got %v", wantErr, err)
	}
}
