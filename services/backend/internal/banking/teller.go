package banking

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"encoding/json"
	"errors"
	"io"
	"math/rand/v2"
	"net/http"
	"time"
)

// Env is the Teller environment. Sandbox is the default.
type Env string

const (
	EnvSandbox     Env = "sandbox"
	EnvDevelopment Env = "development"
	EnvProduction  Env = "production"
)

func ParseEnv(v string) (Env, error) {
	switch Env(v) {
	case "":
		return EnvSandbox, nil
	case EnvSandbox, EnvDevelopment, EnvProduction:
		return Env(v), nil
	}
	return "", errors.New("banking: TELLER_ENV must be sandbox, development or production")
}

// Typed provider errors; none carries token, certificate or provider payload content.
var (
	ErrProviderAuth        = errors.New("banking: provider rejected credentials")
	ErrProviderRateLimited = errors.New("banking: provider rate limited")
	ErrProviderUnavailable = errors.New("banking: provider unavailable")
	ErrProviderRequest     = errors.New("banking: provider rejected request")
	ErrProviderResponse    = errors.New("banking: invalid provider response")
)

// Provider is the Banking-owned boundary; Teller structs never cross it.
type Provider interface {
	// Verify makes one read-only authenticated call proving token, certificate and TLS work.
	Verify(ctx context.Context, token string) error
}

const (
	tellerBaseURL   = "https://api.teller.io"
	maxAttempts     = 3
	maxResponseSize = 1 << 20
	maxInFlight     = 8
)

// TellerClient talks to Teller over verified TLS, presenting the client certificate when configured.
type TellerClient struct {
	base    string
	http    *http.Client
	slots   chan struct{}
	backoff time.Duration
	sleep   func(time.Duration)
}

// NewTellerClient loads the client certificate via src. Development and Production fail closed without a
// valid, unexpired certificate; Sandbox may omit it, but a supplied certificate must be valid and is used.
func NewTellerClient(ctx context.Context, env Env, src SecretSource) (*TellerClient, error) {
	return newTellerClient(ctx, env, src, tellerBaseURL, nil, 10*time.Second, 200*time.Millisecond)
}

func newTellerClient(ctx context.Context, env Env, src SecretSource, base string, roots *x509.CertPool, timeout, backoff time.Duration) (*TellerClient, error) {
	cfg := &tls.Config{MinVersion: tls.VersionTLS12, RootCAs: roots} // nil roots = system pool
	certPEM, certErr := src.Get(ctx, SecretTellerCert)
	keyPEM, keyErr := src.Get(ctx, SecretTellerKey)
	switch {
	case certErr == nil && keyErr == nil:
		pair, err := tls.X509KeyPair(certPEM, keyPEM)
		if err != nil {
			return nil, errors.New("banking: Teller certificate and key are invalid or do not match")
		}
		leaf, err := x509.ParseCertificate(pair.Certificate[0])
		if err != nil {
			return nil, errors.New("banking: Teller certificate is invalid")
		}
		if now := time.Now(); now.Before(leaf.NotBefore) || now.After(leaf.NotAfter) {
			return nil, errors.New("banking: Teller certificate is expired or not yet valid")
		}
		cfg.Certificates = []tls.Certificate{pair}
	case env == EnvSandbox && errors.Is(certErr, ErrSecretUnavailable) && errors.Is(keyErr, ErrSecretUnavailable):
		// Sandbox without a certificate.
	default:
		return nil, errors.New("banking: Teller certificate and key are required and must both load")
	}
	return &TellerClient{
		base: base,
		http: &http.Client{
			Timeout:       timeout,
			Transport:     &http.Transport{TLSClientConfig: cfg, MaxIdleConnsPerHost: maxInFlight},
			CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse },
		},
		slots:   make(chan struct{}, maxInFlight),
		backoff: backoff,
		sleep:   time.Sleep,
	}, nil
}

func (c *TellerClient) Verify(ctx context.Context, token string) error {
	if token == "" {
		return ErrProviderAuth
	}
	select {
	case c.slots <- struct{}{}:
		defer func() { <-c.slots }()
	case <-ctx.Done():
		return ErrProviderUnavailable
	}
	var err error
	for attempt := 0; attempt < maxAttempts; attempt++ {
		if attempt > 0 {
			c.sleep(c.backoff<<(attempt-1) + rand.N(c.backoff+1))
		}
		if err = c.getAccounts(ctx, token); !errors.Is(err, ErrProviderUnavailable) && !errors.Is(err, ErrProviderRateLimited) {
			return err
		}
		if ctx.Err() != nil {
			return ErrProviderUnavailable
		}
	}
	return err
}

// getAccounts is the single read-only call; the payload is only checked for shape and discarded.
func (c *TellerClient) getAccounts(ctx context.Context, token string) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.base+"/accounts", nil)
	if err != nil {
		return ErrProviderRequest
	}
	req.SetBasicAuth(token, "")
	req.Header.Set("Accept", "application/json")
	resp, err := c.http.Do(req)
	if err != nil {
		return ErrProviderUnavailable // err text can embed URLs; never wrapped
	}
	defer resp.Body.Close()
	switch {
	case resp.StatusCode == 401 || resp.StatusCode == 403:
		return ErrProviderAuth
	case resp.StatusCode == 429:
		return ErrProviderRateLimited
	case resp.StatusCode >= 500:
		return ErrProviderUnavailable
	case resp.StatusCode != 200:
		return ErrProviderRequest
	}
	body, err := io.ReadAll(io.LimitReader(resp.Body, maxResponseSize+1))
	if err != nil {
		return ErrProviderUnavailable
	}
	var accounts []json.RawMessage
	if len(body) > maxResponseSize || json.Unmarshal(body, &accounts) != nil {
		return ErrProviderResponse
	}
	return nil
}
