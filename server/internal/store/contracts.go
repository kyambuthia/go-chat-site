package store

// LoginStore is the auth login dependency.
type LoginStore interface {
	GetUserByUsername(username string) (*User, error)
}

// AuthStore handles account registration operations.
type AuthStore interface {
	CreateUser(username, password string) (int, error)
}

// ContactsStore handles contact graph operations.
type ContactsStore interface {
	ListContacts(userID int) ([]User, error)
	GetUserByUsername(username string) (*User, error)
	AddContact(userID, contactID int) error
	RemoveContact(userID, contactID int) error
}

// InviteStore handles invite operations.
type InviteStore interface {
	GetUserByUsername(username string) (*User, error)
	CreateInvite(inviterID, inviteeID int) error
	ListInvites(userID int) ([]Invite, error)
	UpdateInviteStatus(inviteID, userID int, status string) error
}

// MeStore handles profile read operations for the authenticated user.
type MeStore interface {
	GetUserByID(id int) (*User, error)
}

// WalletStore handles wallet and transfer operations.
type WalletStore interface {
	GetWallet(userID int) (*Wallet, error)
	GetUserByUsername(username string) (*User, error)
	SendMoney(senderID, recipientID int, amountCents int64) error
}

// APIStore is the aggregate dependency required by API composition.
type APIStore interface {
	LoginStore
	AuthStore
	ContactsStore
	InviteStore
	MeStore
	WalletStore
}
