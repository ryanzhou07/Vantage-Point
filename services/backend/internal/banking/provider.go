package banking

import (
	"context"
	"errors"
)

// Provider is the Banking-owned bank-connection boundary; implementations return only these types (AGENTS §19).
type Provider interface {
	CreateLinkToken(ctx context.Context, userID string) (LinkToken, error)
	ExchangePublicToken(ctx context.Context, publicToken string) (Exchange, error)
	GetItem(ctx context.Context, accessToken string) (Item, error)
	// RemoveItem is idempotent: an already-removed item is success.
	RemoveItem(ctx context.Context, accessToken string) error
}

type LinkToken struct {
	Token     string
	ExpiresAt string // RFC 3339 as returned by the provider
}

type Exchange struct {
	AccessToken string // secret: encrypt before storing, never log or return
	ItemID      string
}

type Item struct {
	ItemID          string
	InstitutionID   string
	InstitutionName string
}

// Typed provider failures; messages never contain tokens or provider bodies.
var (
	ErrInvalidToken  = errors.New("banking: provider rejected token")
	ErrLoginRequired = errors.New("banking: institution login required")
	ErrRateLimited   = errors.New("banking: provider rate limited")
	ErrUnavailable   = errors.New("banking: provider unavailable")
	ErrRejected      = errors.New("banking: provider rejected request")
)
