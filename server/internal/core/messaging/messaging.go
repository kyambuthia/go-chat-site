package messaging

import (
	"context"
	"time"
)

type UserID int

type MessageKind string

const (
	KindDirectMessage MessageKind = "direct_message"
	KindMessageAck    MessageKind = "message_ack"
	KindUserOnline    MessageKind = "user_online"
	KindUserOffline   MessageKind = "user_offline"
	KindError         MessageKind = "error"
)

// Message is the normalized real-time payload envelope for the messaging domain.
type Message struct {
	ID   int64       `json:"id,omitempty"`
	Type MessageKind `json:"type"`
	From string      `json:"from,omitempty"`
	To   string      `json:"to,omitempty"`
	Body string      `json:"body,omitempty"`
}

// Thread models a future conversation primitive (DM thread, group thread, marketplace order thread).
type Thread struct {
	ID      string
	Members []UserID
}

// DeliveryReceipt captures adapter-level delivery outcomes.
type DeliveryReceipt struct {
	MessageID       int64
	StoredMessageID int64
	Delivered       bool
	Reason          string
}

// StoredMessage is the durable message record shape used for persistence and sync.
type StoredMessage struct {
	ID          int64
	FromUserID  int
	ToUserID    int
	Body        string
	CreatedAt   time.Time
	DeliveredAt *time.Time
	ReadAt      *time.Time
}

// Transport is the adapter seam for centralized relay today and P2P transports later.
type Transport interface {
	SendDirect(toUserID int, msg Message) bool
}

// SessionAuthenticator validates a real-time session token.
type SessionAuthenticator interface {
	AuthenticateSession(ctx context.Context, token string) (userID int, username string, err error)
}

// UserResolver resolves an addressable recipient identifier to an internal user ID.
type UserResolver interface {
	ResolveUserID(ctx context.Context, username string) (int, error)
}

// KeyMaterialProvider is reserved for future E2EE/P2P key lookup.
type KeyMaterialProvider interface {
	MessagingPublicKey(ctx context.Context, userID int, purpose string) ([]byte, error)
}
