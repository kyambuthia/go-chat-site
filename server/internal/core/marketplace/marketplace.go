package marketplace

import "context"

type ListingID string
type OfferID string
type OrderID string
type EscrowID string

type Listing struct {
	ID           ListingID
	SellerUserID int
	Title        string
	Description  string
	PriceCents   int64
	CurrencyCode string
}

type Offer struct {
	ID          OfferID
	ListingID   ListingID
	BuyerUserID int
	AmountCents int64
	Status      string
}

type Order struct {
	ID           OrderID
	ListingID    ListingID
	BuyerUserID  int
	SellerUserID int
	Status       string
}

type Escrow struct {
	ID           EscrowID
	OrderID      OrderID
	Status       string
	AmountCents  int64
	CurrencyCode string
}

// Repository is a stub seam for future marketplace persistence and workflow engines.
type Repository interface {
	CreateListing(ctx context.Context, listing Listing) (Listing, error)
	CreateOffer(ctx context.Context, offer Offer) (Offer, error)
	GetOrder(ctx context.Context, orderID OrderID) (Order, error)
	GetEscrow(ctx context.Context, escrowID EscrowID) (Escrow, error)
}
