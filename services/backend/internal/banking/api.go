package banking

import (
	"context"
	"crypto/rand"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"regexp"
	"strings"
	"sync"
	"time"

	"github.com/ryanzhou07/Vantage-Point/services/backend/internal/foundation"
)

const (
	providerCallsPerMinute = 10
	maxRateUsers           = 10000
	providerName           = "plaid"
)

var (
	uuidRe        = regexp.MustCompile(`^[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}$`)
	publicTokenRe = regexp.MustCompile(`^[\x21-\x7e]{1,512}$`)
)

// API serves /api/v1/banking/link-token and /api/v1/banking/connections.
type API struct {
	store    ConnectionStore
	provider Provider
	keys     *Keyring
	userID   func(context.Context) (string, bool)
	mu       sync.Mutex
	rates    map[string][]time.Time // per instance; shared limiter is a later infrastructure task
}

func NewAPI(store ConnectionStore, provider Provider, keys *Keyring) *API {
	return &API{store: store, provider: provider, keys: keys, userID: foundation.UserID, rates: map[string][]time.Time{}}
}

// Mount registers the routes behind the shared Foundation authentication middleware.
func (a *API) Mount(mux *http.ServeMux, auth *foundation.API) {
	mux.Handle("/api/v1/banking/link-token", auth.ProtectMethods(a, "POST"))
	h := auth.ProtectMethods(a, "GET", "POST", "DELETE")
	mux.Handle("/api/v1/banking/connections", h)
	mux.Handle("/api/v1/banking/connections/", h)
}

// allow bounds expensive provider calls per user (link tokens and exchanges share one budget).
func (a *API) allow(user string) bool {
	a.mu.Lock()
	defer a.mu.Unlock()
	now := time.Now()
	recent := a.rates[user][:0]
	for _, t := range a.rates[user] {
		if now.Sub(t) < time.Minute {
			recent = append(recent, t)
		}
	}
	if len(recent) >= providerCallsPerMinute || (a.rates[user] == nil && len(a.rates) >= maxRateUsers) {
		a.rates[user] = recent
		return false
	}
	a.rates[user] = append(recent, now)
	return true
}

func write(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Cache-Control", "no-store")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}

func fail(w http.ResponseWriter, status int, code string) {
	write(w, status, map[string]any{"error": map[string]string{"code": code}})
}

func newUUID() (string, error) {
	b := make([]byte, 16)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	b[6], b[8] = b[6]&0x0f|0x40, b[8]&0x3f|0x80
	return fmt.Sprintf("%x-%x-%x-%x-%x", b[0:4], b[4:6], b[6:8], b[8:10], b[10:]), nil
}

func (a *API) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	user, ok := a.userID(r.Context())
	if !ok {
		fail(w, 401, "UNAUTHENTICATED")
		return
	}
	if r.URL.Path == "/api/v1/banking/link-token" {
		if r.Method != "POST" {
			fail(w, 405, "METHOD_NOT_ALLOWED")
			return
		}
		a.linkToken(w, r, user)
		return
	}
	id := strings.TrimPrefix(strings.TrimPrefix(r.URL.Path, "/api/v1/banking/connections"), "/")
	switch {
	case id == "" && r.Method == "POST":
		a.connect(w, r, user)
	case id == "" && r.Method == "GET":
		rows, err := a.store.List(r.Context(), user)
		if err != nil {
			fail(w, 503, "UNAVAILABLE")
			return
		}
		if rows == nil {
			rows = []Connection{}
		}
		write(w, 200, map[string]any{"connections": rows})
	case id != "" && !uuidRe.MatchString(id):
		fail(w, 404, "NOT_FOUND")
	case r.Method == "GET":
		c, found, err := a.store.Get(r.Context(), user, id)
		switch {
		case err != nil:
			fail(w, 503, "UNAVAILABLE")
		case !found:
			fail(w, 404, "NOT_FOUND")
		default:
			write(w, 200, c)
		}
	case r.Method == "DELETE":
		a.disconnect(w, r, user, id)
	default:
		fail(w, 405, "METHOD_NOT_ALLOWED")
	}
}

// providerFail maps typed provider errors to stable client codes.
func providerFail(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, ErrInvalidToken):
		fail(w, 400, "INVALID_TOKEN")
	case errors.Is(err, ErrRateLimited):
		fail(w, 429, "RATE_LIMITED")
	default:
		fail(w, 502, "PROVIDER_UNAVAILABLE")
	}
}

func (a *API) linkToken(w http.ResponseWriter, r *http.Request, user string) {
	if !a.allow(user) {
		fail(w, 429, "RATE_LIMITED")
		return
	}
	ctx, cancel := context.WithTimeout(r.Context(), 30*time.Second)
	defer cancel()
	lt, err := a.provider.CreateLinkToken(ctx, user)
	if err != nil {
		providerFail(w, err)
		return
	}
	write(w, 200, map[string]string{"link_token": lt.Token, "expiration": lt.ExpiresAt})
}

func (a *API) connect(w http.ResponseWriter, r *http.Request, user string) {
	var in struct {
		PublicToken string `json:"public_token"`
	}
	dec := json.NewDecoder(io.LimitReader(r.Body, 8193))
	dec.DisallowUnknownFields() // rejects any client-supplied user_id; institution data comes from Plaid, not the browser
	if dec.Decode(&in) != nil || dec.More() || !publicTokenRe.MatchString(in.PublicToken) {
		fail(w, 400, "INVALID_REQUEST")
		return
	}
	if !a.allow(user) {
		fail(w, 429, "RATE_LIMITED")
		return
	}
	ctx, cancel := context.WithTimeout(r.Context(), 30*time.Second)
	defer cancel()
	ex, err := a.provider.ExchangePublicToken(ctx, in.PublicToken)
	if err != nil {
		providerFail(w, err)
		return
	}
	// Name is cosmetic: a lookup failure must not strand a freshly exchanged token.
	var name string
	if it, err := a.provider.GetItem(ctx, ex.AccessToken); err == nil {
		name = it.InstitutionName
	}
	id, err := newUUID()
	var blob []byte
	if err == nil {
		blob, err = a.keys.Seal(user, id, []byte(ex.AccessToken))
	}
	var c Connection
	var created bool
	if err == nil {
		c, created, err = a.store.Enroll(ctx, EnrollInput{UserID: user, ID: id, Provider: providerName, ProviderConnectionID: ex.ItemID, InstitutionName: name, Blob: blob, KeyVersion: a.keys.Version()})
	}
	if err != nil {
		// Do not leave an unrecorded Item linked at the provider; best effort.
		_ = a.provider.RemoveItem(ctx, ex.AccessToken)
		fail(w, 503, "UNAVAILABLE")
		return
	}
	status := 200
	if created {
		status = 201
	}
	write(w, status, c)
}

func (a *API) disconnect(w http.ResponseWriter, r *http.Request, user, id string) {
	ctx, cancel := context.WithTimeout(r.Context(), 30*time.Second)
	defer cancel()
	blob, found, err := a.store.Credential(ctx, user, id)
	switch {
	case err != nil:
		fail(w, 503, "UNAVAILABLE")
		return
	case !found:
		fail(w, 404, "NOT_FOUND")
		return
	}
	// Remove at the provider first so a failure leaves the credential available for retry.
	token, err := a.keys.Open(user, id, blob)
	if err != nil {
		fail(w, 500, "INTERNAL")
		return
	}
	if err := a.provider.RemoveItem(ctx, string(token)); err != nil {
		fail(w, 502, "PROVIDER_UNAVAILABLE")
		return
	}
	if _, err := a.store.Delete(ctx, user, id); err != nil {
		fail(w, 503, "UNAVAILABLE")
		return
	}
	w.Header().Set("Cache-Control", "no-store")
	w.WriteHeader(204)
}
