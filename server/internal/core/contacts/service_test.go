package contacts

import (
	"context"
	"errors"
	"testing"
)

type fakeGraphRepo struct {
	contacts     []Contact
	invites      []Invite
	err          error
	lastUserID   UserID
	lastFrom     UserID
	lastTo       UserID
	lastInviteID int
	lastStatus   InviteStatus
}

func (f *fakeGraphRepo) ListContacts(ctx context.Context, userID UserID) ([]Contact, error) {
	_ = ctx
	f.lastUserID = userID
	return f.contacts, f.err
}

func (f *fakeGraphRepo) CreateInvite(ctx context.Context, fromUser, toUser UserID) error {
	_ = ctx
	f.lastFrom = fromUser
	f.lastTo = toUser
	return f.err
}

func (f *fakeGraphRepo) ListInvites(ctx context.Context, userID UserID) ([]Invite, error) {
	_ = ctx
	f.lastUserID = userID
	return f.invites, f.err
}

func (f *fakeGraphRepo) UpdateInvite(ctx context.Context, inviteID int, userID UserID, status InviteStatus) error {
	_ = ctx
	f.lastInviteID = inviteID
	f.lastUserID = userID
	f.lastStatus = status
	return f.err
}

func TestCoreService_ListContacts_DelegatesToRepository(t *testing.T) {
	repo := &fakeGraphRepo{contacts: []Contact{{UserID: 2, Username: "bob"}}}
	svc := NewService(repo, nil)

	got, err := svc.ListContacts(context.Background(), 1)
	if err != nil {
		t.Fatalf("ListContacts returned error: %v", err)
	}
	if repo.lastUserID != 1 {
		t.Fatalf("repo called with userID=%d, want 1", repo.lastUserID)
	}
	if len(got) != 1 || got[0].Username != "bob" {
		t.Fatalf("unexpected contacts: %+v", got)
	}
}

func TestCoreService_SendInvite_DelegatesToRepository(t *testing.T) {
	repo := &fakeGraphRepo{}
	svc := NewService(repo, nil)

	if err := svc.SendInvite(context.Background(), 1, 2); err != nil {
		t.Fatalf("SendInvite returned error: %v", err)
	}
	if repo.lastFrom != 1 || repo.lastTo != 2 {
		t.Fatalf("unexpected invite call from=%d to=%d", repo.lastFrom, repo.lastTo)
	}
}

func TestCoreService_RespondToInvite_PropagatesErrors(t *testing.T) {
	repo := &fakeGraphRepo{err: errors.New("db down")}
	svc := NewService(repo, nil)

	err := svc.RespondToInvite(context.Background(), 9, 1, InviteAccepted)
	if err == nil || err.Error() != "db down" {
		t.Fatalf("expected propagated error, got %v", err)
	}
	if repo.lastInviteID != 9 || repo.lastUserID != 1 || repo.lastStatus != InviteAccepted {
		t.Fatalf("unexpected update call inviteID=%d userID=%d status=%q", repo.lastInviteID, repo.lastUserID, repo.lastStatus)
	}
}

type fakeUserDirectory struct {
	userID UserID
	err    error
	last   string
}

func (f *fakeUserDirectory) ResolveUserIDByUsername(ctx context.Context, username string) (UserID, error) {
	_ = ctx
	f.last = username
	return f.userID, f.err
}

func TestCoreService_SendInviteByUsername_ResolvesUserAndCreatesInvite(t *testing.T) {
	repo := &fakeGraphRepo{}
	dir := &fakeUserDirectory{userID: 22}
	svc := NewService(repo, dir)

	if err := svc.SendInviteByUsername(context.Background(), 11, "bob"); err != nil {
		t.Fatalf("SendInviteByUsername returned error: %v", err)
	}
	if dir.last != "bob" {
		t.Fatalf("directory lookup username = %q, want bob", dir.last)
	}
	if repo.lastFrom != 11 || repo.lastTo != 22 {
		t.Fatalf("unexpected repo invite call from=%d to=%d", repo.lastFrom, repo.lastTo)
	}
}

func TestCoreService_SendInviteByUsername_ReturnsUserNotFound(t *testing.T) {
	repo := &fakeGraphRepo{}
	dir := &fakeUserDirectory{err: errors.New("not found")}
	svc := NewService(repo, dir)

	err := svc.SendInviteByUsername(context.Background(), 11, "missing")
	if !errors.Is(err, ErrUserNotFound) {
		t.Fatalf("expected ErrUserNotFound, got %v", err)
	}
}
