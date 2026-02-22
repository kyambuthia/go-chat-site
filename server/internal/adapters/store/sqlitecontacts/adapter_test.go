package sqlitecontacts

import (
	"context"
	"errors"
	"testing"

	corecontacts "github.com/kyambuthia/go-chat-site/server/internal/core/contacts"
	"github.com/kyambuthia/go-chat-site/server/internal/store"
)

type fakeDeps struct {
	contacts []store.User
	invites  []store.Invite
	user     *store.User

	listContactsErr error
	createInviteErr error
	listInvitesErr  error
	updateInviteErr error
	userErr         error

	lastListContacts int
	lastCreateFrom   int
	lastCreateTo     int
	lastListInvites  int
	lastUpdateInvite int
	lastUpdateUser   int
	lastUpdateStatus string
	lastLookup       string
}

func (f *fakeDeps) ListContacts(userID int) ([]store.User, error) {
	f.lastListContacts = userID
	return f.contacts, f.listContactsErr
}
func (f *fakeDeps) GetUserByUsername(username string) (*store.User, error) {
	f.lastLookup = username
	if f.userErr != nil {
		return nil, f.userErr
	}
	return f.user, nil
}
func (f *fakeDeps) AddContact(userID, contactID int) error { _ = userID; _ = contactID; return nil }
func (f *fakeDeps) RemoveContact(userID, contactID int) error { _ = userID; _ = contactID; return nil }
func (f *fakeDeps) CreateInvite(inviterID, inviteeID int) error {
	f.lastCreateFrom, f.lastCreateTo = inviterID, inviteeID
	return f.createInviteErr
}
func (f *fakeDeps) ListInvites(userID int) ([]store.Invite, error) {
	f.lastListInvites = userID
	return f.invites, f.listInvitesErr
}
func (f *fakeDeps) UpdateInviteStatus(inviteID, userID int, status string) error {
	f.lastUpdateInvite, f.lastUpdateUser, f.lastUpdateStatus = inviteID, userID, status
	return f.updateInviteErr
}

func TestAdapter_ListContacts_MapsStoreUsersToCoreContacts(t *testing.T) {
	deps := &fakeDeps{contacts: []store.User{{ID: 2, Username: "bob", DisplayName: "Bob", AvatarURL: "x"}}}
	a := &Adapter{Store: deps}

	got, err := a.ListContacts(context.Background(), 1)
	if err != nil {
		t.Fatalf("ListContacts error: %v", err)
	}
	if deps.lastListContacts != 1 {
		t.Fatalf("ListContacts userID=%d, want 1", deps.lastListContacts)
	}
	if len(got) != 1 || got[0].UserID != 2 || got[0].Username != "bob" || got[0].DisplayName != "Bob" {
		t.Fatalf("unexpected contacts mapping: %+v", got)
	}
}

func TestAdapter_ListInvites_MapsStoreInvitesToCoreInvites(t *testing.T) {
	deps := &fakeDeps{invites: []store.Invite{{ID: 9, InviterUsername: "alice"}}}
	a := &Adapter{Store: deps}

	got, err := a.ListInvites(context.Background(), 3)
	if err != nil {
		t.Fatalf("ListInvites error: %v", err)
	}
	if deps.lastListInvites != 3 {
		t.Fatalf("ListInvites userID=%d, want 3", deps.lastListInvites)
	}
	if len(got) != 1 || got[0].ID != 9 || got[0].InviterUsername != "alice" || got[0].Status != corecontacts.InvitePending {
		t.Fatalf("unexpected invites mapping: %+v", got)
	}
}

func TestAdapter_UpdateInvite_TranslatesNotFoundError(t *testing.T) {
	deps := &fakeDeps{updateInviteErr: store.ErrNotFound}
	a := &Adapter{Store: deps}

	err := a.UpdateInvite(context.Background(), 7, 2, corecontacts.InviteAccepted)
	if !errors.Is(err, corecontacts.ErrInviteNotFound) {
		t.Fatalf("expected ErrInviteNotFound, got %v", err)
	}
	if deps.lastUpdateInvite != 7 || deps.lastUpdateUser != 2 || deps.lastUpdateStatus != "accepted" {
		t.Fatalf("unexpected UpdateInviteStatus call invite=%d user=%d status=%q", deps.lastUpdateInvite, deps.lastUpdateUser, deps.lastUpdateStatus)
	}
}

func TestAdapter_ResolveUserIDByUsername_DelegatesLookup(t *testing.T) {
	deps := &fakeDeps{user: &store.User{ID: 12, Username: "bob"}}
	a := &Adapter{Store: deps}

	got, err := a.ResolveUserIDByUsername(context.Background(), "bob")
	if err != nil {
		t.Fatalf("ResolveUserIDByUsername error: %v", err)
	}
	if deps.lastLookup != "bob" || got != 12 {
		t.Fatalf("unexpected lookup result lookup=%q id=%d", deps.lastLookup, got)
	}
}
