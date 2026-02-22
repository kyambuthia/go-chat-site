package ledger

import (
	"context"
	"time"
)

type AccountID string

type EventType string

const (
	EventTransferInitiated EventType = "transfer_initiated"
	EventTransferSettled   EventType = "transfer_settled"
	EventTransferRejected  EventType = "transfer_rejected"
)

// Account keeps the current centralized balance semantics while renaming the domain from wallet -> ledger.
type Account struct {
	ID           AccountID
	OwnerUserID  int
	BalanceCents int64
	CurrencyCode string
}

// Transfer represents a ledger movement request/result.
type Transfer struct {
	ID              string
	FromUserID      int
	ToUserID        int
	AmountCents     int64
	CurrencyCode    string
	CreatedAt       time.Time
	CorrelationID   string
	ExternalRailRef string
}

// Event is an append-only ledger event stub for future auditability/integration.
type Event struct {
	ID         string
	TransferID string
	Type       EventType
	OccurredAt time.Time
	Metadata   map[string]string
}

// Repository is the persistence seam for current sqlite and future external ledger/payment rails.
type Repository interface {
	GetAccount(ctx context.Context, userID int) (Account, error)
	Transfer(ctx context.Context, transfer Transfer) (Transfer, error)
}
