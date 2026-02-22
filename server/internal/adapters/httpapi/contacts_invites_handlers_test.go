package httpapi

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/kyambuthia/go-chat-site/server/internal/auth"
	corecontacts "github.com/kyambuthia/go-chat-site/server/internal/core/contacts"
	"github.com/kyambuthia/go-chat-site/server/internal/store"
)

type fakeContactsService struct {
	contacts   []corecontacts.Contact
	invites    []corecontacts.Invite
	err        error
	lastUserID corecontacts.UserID
	lastFrom   corecontacts.UserID
	lastTo     corecontacts.UserID
	lastInvite int
	lastStatus corecontacts.InviteStatus
	lastLookup string
	lastRemove corecontacts.UserID
}

func (f *fakeContactsService) ListContacts(ctx context.Context, userID corecontacts.UserID) ([]corecontacts.Contact, error) {
	_ = ctx
	f.lastUserID = userID
	return f.contacts, f.err
}

func (f *fakeContactsService) SendInvite(ctx context.Context, fromUser, toUser corecontacts.UserID) error {
	_ = ctx
	f.lastFrom = fromUser
	f.lastTo = toUser
	return f.err
}

func (f *fakeContactsService) AddContactByUsername(ctx context.Context, userID corecontacts.UserID, username string) error {
	_ = ctx
	f.lastUserID = userID
	f.lastLookup = username
	return f.err
}

func (f *fakeContactsService) RemoveContact(ctx context.Context, userID, contactID corecontacts.UserID) error {
	_ = ctx
	f.lastUserID = userID
	f.lastRemove = contactID
	return f.err
}

func (f *fakeContactsService) SendInviteByUsername(ctx context.Context, fromUser corecontacts.UserID, username string) error {
	_ = ctx
	f.lastFrom = fromUser
	f.lastLookup = username
	return f.err
}

func (f *fakeContactsService) ListInvites(ctx context.Context, userID corecontacts.UserID) ([]corecontacts.Invite, error) {
	_ = ctx
	f.lastUserID = userID
	return f.invites, f.err
}

func (f *fakeContactsService) RespondToInvite(ctx context.Context, inviteID int, userID corecontacts.UserID, status corecontacts.InviteStatus) error {
	_ = ctx
	f.lastInvite = inviteID
	f.lastUserID = userID
	f.lastStatus = status
	return f.err
}

func authedJSONReq(t *testing.T, method, target string, body []byte, userID int) *http.Request {
	t.Helper()
	req := httptest.NewRequest(method, target, bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	return req.WithContext(auth.WithUserID(req.Context(), userID))
}

func TestContactsHandler_GetContacts_UsesCoreServiceAndPreservesResponseShape(t *testing.T) {
	svc := &fakeContactsService{
		contacts: []corecontacts.Contact{{
			UserID:      2,
			Username:    "bob",
			DisplayName: "Bob",
			AvatarURL:   "https://example.com/bob.png",
		}},
	}
	h := &ContactsHandler{Contacts: svc}

	rr := httptest.NewRecorder()
	h.GetContacts(rr, authedJSONReq(t, http.MethodGet, "/api/contacts", nil, 1))

	if rr.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", rr.Code)
	}
	if svc.lastUserID != 1 {
		t.Fatalf("ListContacts userID = %d, want 1", svc.lastUserID)
	}

	var resp []map[string]any
	if err := json.Unmarshal(rr.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal response: %v", err)
	}
	if len(resp) != 1 {
		t.Fatalf("expected 1 contact, got %d", len(resp))
	}
	if got := int(resp[0]["id"].(float64)); got != 2 {
		t.Fatalf("id = %d, want 2", got)
	}
	if got := resp[0]["display_name"].(string); got != "Bob" {
		t.Fatalf("display_name = %q, want Bob", got)
	}
}

func TestContactsHandler_AddContact_UsesCoreServiceAndMapsErrors(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		svc := &fakeContactsService{}
		h := &ContactsHandler{Contacts: svc}

		rr := httptest.NewRecorder()
		h.AddContact(rr, authedJSONReq(t, http.MethodPost, "/api/contacts", []byte(`{"username":"bob"}`), 1))

		if rr.Code != http.StatusCreated {
			t.Fatalf("status = %d, want 201", rr.Code)
		}
		if svc.lastUserID != 1 || svc.lastLookup != "bob" {
			t.Fatalf("unexpected AddContactByUsername call user=%d username=%q", svc.lastUserID, svc.lastLookup)
		}
	})

	t.Run("user not found", func(t *testing.T) {
		svc := &fakeContactsService{err: corecontacts.ErrUserNotFound}
		h := &ContactsHandler{Contacts: svc}

		rr := httptest.NewRecorder()
		h.AddContact(rr, authedJSONReq(t, http.MethodPost, "/api/contacts", []byte(`{"username":"missing"}`), 1))

		if rr.Code != http.StatusNotFound {
			t.Fatalf("status = %d, want 404", rr.Code)
		}
	})
}

func TestContactsHandler_RemoveContact_UsesCoreService(t *testing.T) {
	svc := &fakeContactsService{}
	h := &ContactsHandler{Contacts: svc}

	rr := httptest.NewRecorder()
	h.RemoveContact(rr, authedJSONReq(t, http.MethodDelete, "/api/contacts", []byte(`{"contact_id":9}`), 1))

	if rr.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", rr.Code)
	}
	if svc.lastUserID != 1 || svc.lastRemove != 9 {
		t.Fatalf("unexpected RemoveContact call user=%d contact=%d", svc.lastUserID, svc.lastRemove)
	}
}

func TestInviteHandler_SendInvite_MapsCoreErrorsAndPreservesBehavior(t *testing.T) {
	t.Run("user not found", func(t *testing.T) {
		svc := &fakeContactsService{err: corecontacts.ErrUserNotFound}
		h := &InviteHandler{Contacts: svc}

		rr := httptest.NewRecorder()
		h.SendInvite(rr, authedJSONReq(t, http.MethodPost, "/api/invites/send", []byte(`{"username":"missing"}`), 1))

		if rr.Code != http.StatusNotFound {
			t.Fatalf("status = %d, want 404", rr.Code)
		}
		if svc.lastFrom != 1 || svc.lastLookup != "missing" {
			t.Fatalf("unexpected SendInviteByUsername call from=%d username=%q", svc.lastFrom, svc.lastLookup)
		}
	})

	t.Run("duplicate invite", func(t *testing.T) {
		svc := &fakeContactsService{err: store.ErrInviteExists}
		h := &InviteHandler{Contacts: svc}

		rr := httptest.NewRecorder()
		h.SendInvite(rr, authedJSONReq(t, http.MethodPost, "/api/invites/send", []byte(`{"username":"bob"}`), 1))

		if rr.Code != http.StatusConflict {
			t.Fatalf("status = %d, want 409", rr.Code)
		}
	})
}

func TestInviteHandler_GetInvites_UsesCoreServiceAndPreservesResponseShape(t *testing.T) {
	svc := &fakeContactsService{
		invites: []corecontacts.Invite{{
			ID:              9,
			InviterUsername: "alice",
		}},
	}
	h := &InviteHandler{Contacts: svc}

	rr := httptest.NewRecorder()
	h.GetInvites(rr, authedJSONReq(t, http.MethodGet, "/api/invites", nil, 2))

	if rr.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", rr.Code)
	}
	var resp []map[string]any
	if err := json.Unmarshal(rr.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal response: %v", err)
	}
	if len(resp) != 1 {
		t.Fatalf("expected 1 invite, got %d", len(resp))
	}
	if got := int(resp[0]["id"].(float64)); got != 9 {
		t.Fatalf("id = %d, want 9", got)
	}
	if got := resp[0]["inviter_username"].(string); got != "alice" {
		t.Fatalf("inviter_username = %q, want alice", got)
	}
}

func TestInviteHandler_UpdateInviteStatus_MapsCoreNotFoundTo404(t *testing.T) {
	svc := &fakeContactsService{err: corecontacts.ErrInviteNotFound}
	h := &InviteHandler{Contacts: svc}

	rr := httptest.NewRecorder()
	h.AcceptInvite(rr, authedJSONReq(t, http.MethodPost, "/api/invites/accept", []byte(`{"invite_id":7}`), 2))

	if rr.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want 404", rr.Code)
	}
	if svc.lastInvite != 7 || svc.lastUserID != 2 || svc.lastStatus != corecontacts.InviteAccepted {
		t.Fatalf("unexpected RespondToInvite call invite=%d user=%d status=%q", svc.lastInvite, svc.lastUserID, svc.lastStatus)
	}
}

func TestInviteHandler_UpdateInviteStatus_PropagatesUnexpectedError(t *testing.T) {
	svc := &fakeContactsService{err: errors.New("db down")}
	h := &InviteHandler{Contacts: svc}

	rr := httptest.NewRecorder()
	h.RejectInvite(rr, authedJSONReq(t, http.MethodPost, "/api/invites/reject", []byte(`{"invite_id":7}`), 2))

	if rr.Code != http.StatusInternalServerError {
		t.Fatalf("status = %d, want 500", rr.Code)
	}
}
