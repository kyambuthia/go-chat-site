package sqliteidentity

import (
	"context"
	"errors"
	"testing"

	coreid "github.com/kyambuthia/go-chat-site/server/internal/core/identity"
	"github.com/kyambuthia/go-chat-site/server/internal/store"
)

type fakeMeStore struct {
	user    *store.User
	err     error
	lastID  int
}

func (f *fakeMeStore) GetUserByID(id int) (*store.User, error) {
	f.lastID = id
	if f.err != nil {
		return nil, f.err
	}
	return f.user, nil
}

func TestAdapter_GetProfile_MapsStoreUserToIdentityProfile(t *testing.T) {
	a := &Adapter{Store: &fakeMeStore{user: &store.User{
		ID: 4, Username: "alice", DisplayName: "Alice", AvatarURL: "https://example.com/a.png",
	}}}

	got, err := a.GetProfile(context.Background(), 4)
	if err != nil {
		t.Fatalf("GetProfile error: %v", err)
	}
	if got.UserID != coreid.UserID(4) || got.Username != "alice" || got.DisplayName != "Alice" {
		t.Fatalf("unexpected profile: %+v", got)
	}
}

func TestAdapter_GetProfile_PropagatesErrors(t *testing.T) {
	wantErr := errors.New("db down")
	a := &Adapter{Store: &fakeMeStore{err: wantErr}}

	_, err := a.GetProfile(context.Background(), 1)
	if !errors.Is(err, wantErr) {
		t.Fatalf("expected %v, got %v", wantErr, err)
	}
}
