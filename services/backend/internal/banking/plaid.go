package banking

import (
	"context"
	"errors"
	"net/http"
	"time"

	plaid "github.com/plaid/plaid-go/v49"
	"github.com/plaid/plaid-go/v49/client"
	"github.com/plaid/plaid-go/v49/core"
	"github.com/plaid/plaid-go/v49/option"
)

const (
	plaidMaxBody     = 1 << 20
	plaidMaxAttempts = 3 // SDK retries 408/429/5xx with backoff and jitter
)

// Plaid implements Provider. plaid-go types stay in this file (AGENTS §19).
type Plaid struct {
	c   *client.Client
	sem chan struct{}
}

// NewPlaid builds the adapter: 10s request timeout, 1 MiB response cap, at most 4 calls in flight.
func NewPlaid(cfg PlaidConfig) *Plaid {
	base := plaid.Environments.Sandbox
	if cfg.Env == "production" {
		base = plaid.Environments.Production
	}
	return newPlaid(cfg, base)
}

func newPlaid(cfg PlaidConfig, baseURL string) *Plaid {
	hc := &http.Client{Timeout: 10 * time.Second, Transport: capBody{http.DefaultTransport}}
	opts := []core.RequestOption{
		option.WithClientID(plaid.String(cfg.ClientID)),
		option.WithSecret(cfg.Secret),
		option.WithMaxAttempts(plaidMaxAttempts),
	}
	if baseURL == plaid.Environments.Sandbox || baseURL == plaid.Environments.Production {
		opts = append(opts, option.WithBaseURL(baseURL), option.WithHTTPClient(hc))
	} else { // the SDK refuses non-Plaid hosts unless named explicitly; only tests reach this
		opts = append(opts, option.WithCustomEndpoint(baseURL, hc))
	}
	return &Plaid{c: client.NewClient(opts...), sem: make(chan struct{}, 4)}
}

// capBody makes reads past plaidMaxBody fail instead of buffering unbounded provider output.
type capBody struct{ http.RoundTripper }

func (t capBody) RoundTrip(r *http.Request) (*http.Response, error) {
	resp, err := t.RoundTripper.RoundTrip(r)
	if resp != nil {
		resp.Body = http.MaxBytesReader(nil, resp.Body, plaidMaxBody)
	}
	return resp, err
}

func (p *Plaid) acquire(ctx context.Context) (func(), error) {
	select {
	case p.sem <- struct{}{}:
		return func() { <-p.sem }, nil
	case <-ctx.Done():
		return nil, ErrUnavailable
	}
}

// mapErr reduces SDK errors to typed ones; messages never carry provider bodies or tokens.
func mapErr(err error) error {
	if err == nil {
		return nil
	}
	var pe interface{ GetBody() *plaid.PlaidError }
	code, typ := "", ""
	if errors.As(err, &pe) && pe.GetBody() != nil {
		code, typ = pe.GetBody().ErrorCode, string(pe.GetBody().ErrorType)
	}
	var ae *core.APIError
	status := 0
	if errors.As(err, &ae) {
		status = ae.StatusCode
	}
	switch {
	case code == "ITEM_LOGIN_REQUIRED":
		return ErrLoginRequired
	case code == "INVALID_PUBLIC_TOKEN" || code == "INVALID_ACCESS_TOKEN" || code == "INVALID_LINK_TOKEN":
		return ErrInvalidToken
	case typ == "RATE_LIMIT_EXCEEDED" || status == http.StatusTooManyRequests:
		return ErrRateLimited
	case status == 0 || status >= 500 || typ == "INSTITUTION_ERROR" || typ == "API_ERROR":
		return ErrUnavailable // transport/decode failures too
	}
	return ErrRejected
}

func (p *Plaid) CreateLinkToken(ctx context.Context, userID string) (LinkToken, error) {
	if userID == "" {
		return LinkToken{}, ErrRejected
	}
	done, err := p.acquire(ctx)
	if err != nil {
		return LinkToken{}, err
	}
	defer done()
	r, err := p.c.Link.CreateToken(ctx, &plaid.LinkTokenCreateRequest{
		ClientName:   "Vantage Point",
		Language:     "en",
		CountryCodes: []plaid.CountryCode{plaid.CountryCodeUs},
		User:         &plaid.LinkTokenCreateRequestUser{ClientUserID: userID},
		Products:     []plaid.Products{plaid.ProductsTransactions},
	})
	if err != nil {
		return LinkToken{}, mapErr(err)
	}
	if r.LinkToken == "" {
		return LinkToken{}, ErrUnavailable
	}
	return LinkToken{Token: r.LinkToken, ExpiresAt: r.Expiration.Format(time.RFC3339)}, nil
}

func (p *Plaid) ExchangePublicToken(ctx context.Context, publicToken string) (Exchange, error) {
	if publicToken == "" {
		return Exchange{}, ErrInvalidToken
	}
	done, err := p.acquire(ctx)
	if err != nil {
		return Exchange{}, err
	}
	defer done()
	// Public tokens are single-use, so this call is never retried.
	r, err := p.c.Item.ExchangePublicToken(ctx, &plaid.ItemPublicTokenExchangeRequest{PublicToken: publicToken}, option.WithoutRetries())
	if err != nil {
		return Exchange{}, mapErr(err)
	}
	if r.AccessToken == "" || r.ItemID == "" {
		return Exchange{}, ErrUnavailable
	}
	return Exchange{AccessToken: r.AccessToken, ItemID: r.ItemID}, nil
}

func (p *Plaid) GetItem(ctx context.Context, accessToken string) (Item, error) {
	done, err := p.acquire(ctx)
	if err != nil {
		return Item{}, err
	}
	defer done()
	r, err := p.c.Item.Get(ctx, &plaid.ItemGetRequest{AccessToken: accessToken})
	if err != nil {
		return Item{}, mapErr(err)
	}
	if r.Item == nil {
		return Item{}, ErrUnavailable
	}
	it := Item{ItemID: r.Item.ItemID}
	if r.Item.InstitutionID != nil {
		it.InstitutionID = *r.Item.InstitutionID
	}
	if r.Item.InstitutionName != nil {
		it.InstitutionName = *r.Item.InstitutionName
	}
	return it, nil
}

func (p *Plaid) RemoveItem(ctx context.Context, accessToken string) error {
	done, err := p.acquire(ctx)
	if err != nil {
		return err
	}
	defer done()
	_, err = p.c.Item.Remove(ctx, &plaid.ItemRemoveRequest{AccessToken: accessToken})
	if err = mapErr(err); errors.Is(err, ErrInvalidToken) {
		return nil // already removed; retry-safe
	}
	return err
}
