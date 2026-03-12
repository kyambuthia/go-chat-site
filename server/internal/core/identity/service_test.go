package identity

import (
	"context"
	"errors"
	"testing"
)

type fakeProfileRepo struct {
	profile    Profile
	err        error
	lastUserID UserID
	lastUpdate ProfileUpdate
}

func (f *fakeProfileRepo) GetProfile(ctx context.Context, userID UserID) (Profile, error) {
	_ = ctx
	f.lastUserID = userID
	return f.profile, f.err
}

func (f *fakeProfileRepo) UpdateProfile(ctx context.Context, userID UserID, update ProfileUpdate) (Profile, error) {
	_ = ctx
	f.lastUserID = userID
	f.lastUpdate = update
	return f.profile, f.err
}

func TestProfileService_GetProfile_DelegatesToRepository(t *testing.T) {
	repo := &fakeProfileRepo{profile: Profile{UserID: 7, Username: "alice"}}
	svc := NewProfileService(repo)

	got, err := svc.GetProfile(context.Background(), 7)
	if err != nil {
		t.Fatalf("GetProfile returned error: %v", err)
	}
	if repo.lastUserID != 7 {
		t.Fatalf("repo called with userID=%d, want 7", repo.lastUserID)
	}
	if got.Username != "alice" {
		t.Fatalf("unexpected profile: %+v", got)
	}
}

func TestProfileService_GetProfile_PropagatesErrors(t *testing.T) {
	repo := &fakeProfileRepo{err: errors.New("db down")}
	svc := NewProfileService(repo)

	if _, err := svc.GetProfile(context.Background(), 1); err == nil {
		t.Fatal("expected error")
	}
}

func TestProfileService_UpdateProfile_DelegatesToRepository(t *testing.T) {
	repo := &fakeProfileRepo{profile: Profile{
		UserID:      7,
		Username:    "alice",
		DisplayName: "Alice",
		AvatarURL:   "https://example.com/a.png",
	}}
	svc := NewProfileService(repo)

	got, err := svc.UpdateProfile(context.Background(), 7, ProfileUpdate{
		DisplayName: "Alice Doe",
		AvatarURL:   "https://example.com/alice.png",
	})
	if err != nil {
		t.Fatalf("UpdateProfile returned error: %v", err)
	}
	if repo.lastUserID != 7 {
		t.Fatalf("repo called with userID=%d, want 7", repo.lastUserID)
	}
	if repo.lastUpdate.DisplayName != "Alice Doe" || repo.lastUpdate.AvatarURL != "https://example.com/alice.png" {
		t.Fatalf("unexpected update: %+v", repo.lastUpdate)
	}
	if got.DisplayName != "Alice" || got.AvatarURL != "https://example.com/a.png" {
		t.Fatalf("unexpected profile: %+v", got)
	}
}
