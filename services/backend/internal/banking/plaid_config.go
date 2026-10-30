package banking

import (
	"context"
	"errors"
	"fmt"
	"strings"
)

var ErrBadPlaidConfig = errors.New("banking: invalid Plaid configuration")

// PlaidConfig holds Plaid credentials; String/GoString never reveal them.
type PlaidConfig struct {
	Env      string // "sandbox" or "production"
	ClientID string
	Secret   string
}

func (c PlaidConfig) String() string   { return "PlaidConfig{env=" + c.Env + ", credentials redacted}" }
func (c PlaidConfig) GoString() string { return c.String() }

// LoadPlaidConfig reads PLAID_ENV (default sandbox; unknown fails closed) and the credentials from src.
func LoadPlaidConfig(ctx context.Context, getenv func(string) string, src SecretSource) (PlaidConfig, error) {
	env := strings.TrimSpace(getenv("PLAID_ENV"))
	if env == "" {
		env = "sandbox"
	}
	if env != "sandbox" && env != "production" {
		return PlaidConfig{}, fmt.Errorf("%w: PLAID_ENV must be sandbox or production", ErrBadPlaidConfig)
	}
	id, err := src.Get(ctx, SecretPlaidClientID)
	if err != nil {
		return PlaidConfig{}, fmt.Errorf("%w: client id unavailable", ErrBadPlaidConfig)
	}
	sec, err := src.Get(ctx, SecretPlaidSecret)
	if err != nil {
		return PlaidConfig{}, fmt.Errorf("%w: secret unavailable", ErrBadPlaidConfig)
	}
	c := PlaidConfig{Env: env, ClientID: strings.TrimSpace(string(id)), Secret: strings.TrimSpace(string(sec))}
	if c.ClientID == "" || c.Secret == "" {
		return PlaidConfig{}, fmt.Errorf("%w: credentials empty", ErrBadPlaidConfig)
	}
	return c, nil
}
