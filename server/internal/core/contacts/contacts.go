package contacts

import "context"

type UserID int

type Contact struct {
	UserID      UserID
	Username    string
	DisplayName string
	AvatarURL   string
}

type InviteStatus string

const (
	InvitePending  InviteStatus = "pending"
	InviteAccepted InviteStatus = "accepted"
	InviteRejected InviteStatus = "rejected"
)

type Invite struct {
	ID       int
	FromUser UserID
	ToUser   UserID
	Status   InviteStatus
}

// GraphRepository abstracts contact graph persistence.
type GraphRepository interface {
	ListContacts(ctx context.Context, userID UserID) ([]Contact, error)
	CreateInvite(ctx context.Context, fromUser, toUser UserID) error
	ListInvites(ctx context.Context, userID UserID) ([]Invite, error)
	UpdateInvite(ctx context.Context, inviteID int, userID UserID, status InviteStatus) error
}
