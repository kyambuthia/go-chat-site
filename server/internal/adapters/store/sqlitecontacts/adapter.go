package sqlitecontacts

import (
	"context"
	"errors"

	corecontacts "github.com/kyambuthia/go-chat-site/server/internal/core/contacts"
	"github.com/kyambuthia/go-chat-site/server/internal/store"
)

type Dependencies interface {
	store.ContactsStore
	store.InviteStore
}

// Adapter bridges the existing SQLite contact/invite store methods to core contacts interfaces.
type Adapter struct {
	Store Dependencies
}

var _ corecontacts.GraphRepository = (*Adapter)(nil)
var _ corecontacts.UserDirectory = (*Adapter)(nil)

func (a *Adapter) ListContacts(ctx context.Context, userID corecontacts.UserID) ([]corecontacts.Contact, error) {
	_ = ctx
	rows, err := a.Store.ListContacts(int(userID))
	if err != nil {
		return nil, err
	}
	out := make([]corecontacts.Contact, 0, len(rows))
	for _, row := range rows {
		out = append(out, corecontacts.Contact{
			UserID:      corecontacts.UserID(row.ID),
			Username:    row.Username,
			DisplayName: row.DisplayName,
			AvatarURL:   row.AvatarURL,
		})
	}
	return out, nil
}

func (a *Adapter) CreateInvite(ctx context.Context, fromUser, toUser corecontacts.UserID) error {
	_ = ctx
	return a.Store.CreateInvite(int(fromUser), int(toUser))
}

func (a *Adapter) ListInvites(ctx context.Context, userID corecontacts.UserID) ([]corecontacts.Invite, error) {
	_ = ctx
	rows, err := a.Store.ListInvites(int(userID))
	if err != nil {
		return nil, err
	}
	out := make([]corecontacts.Invite, 0, len(rows))
	for _, row := range rows {
		out = append(out, corecontacts.Invite{
			ID:              row.ID,
			ToUser:          userID,
			Status:          corecontacts.InvitePending,
			InviterUsername: row.InviterUsername,
		})
	}
	return out, nil
}

func (a *Adapter) UpdateInvite(ctx context.Context, inviteID int, userID corecontacts.UserID, status corecontacts.InviteStatus) error {
	_ = ctx
	err := a.Store.UpdateInviteStatus(inviteID, int(userID), string(status))
	if errors.Is(err, store.ErrNotFound) {
		return corecontacts.ErrInviteNotFound
	}
	return err
}

func (a *Adapter) ResolveUserIDByUsername(ctx context.Context, username string) (corecontacts.UserID, error) {
	_ = ctx
	user, err := a.Store.GetUserByUsername(username)
	if err != nil {
		return 0, err
	}
	return corecontacts.UserID(user.ID), nil
}
