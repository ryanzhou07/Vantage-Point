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
	enrollPerMinute = 10
	maxRateUsers    = 10000
)

var (
	uuidRe       = regexp.MustCompile(`^[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}$`)
	enrollmentRe = regexp.MustCompile(`^[A-Za-z0-9_\-]{1,100}$`)
	tokenRe      = regexp.MustCompile(`^[\x21-\x7e]{1,512}$`)
)

// API serves /api/v1/banking/connections. Wrap it with foundation ProtectMethods(GET, POST, DELETE).
type API struct {
	store    ConnectionStore
	provider Provider
	keys     *Keyring
	userID   func(context.Context) (string, bool)
	mu       sync.Mutex
	rates    map[string][]time.Time
}

func NewAPI(store ConnectionStore, provider Provider, keys *Keyring) *API {
	return &API{store: store, provider: provider, keys: keys, userID: foundation.UserID, rates: map[string][]time.Time{}}
}

// Mount registers the routes behind the shared Foundation authentication middleware.
func (a *API) Mount(mux *http.ServeMux, auth *foundation.API) {
	h := auth.ProtectMethods(a, "GET", "POST", "DELETE")
	mux.Handle("/api/v1/banking/connections", h)
	mux.Handle("/api/v1/banking/connections/", h)
}

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
	if len(recent) >= enrollPerMinute || (a.rates[user] == nil && len(a.rates) >= maxRateUsers) {
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
	id := strings.TrimPrefix(strings.TrimPrefix(r.URL.Path, "/api/v1/banking/connections"), "/")
	switch {
	case id == "" && r.Method == "POST":
		a.enroll(w, r, user)
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

type enrollBody struct {
	EnrollmentID    string `json:"enrollment_id"`
	AccessToken     string `json:"access_token"`
	InstitutionName string `json:"institution_name"`
}

func (a *API) enroll(w http.ResponseWriter, r *http.Request, user string) {
	var in enrollBody
	dec := json.NewDecoder(io.LimitReader(r.Body, 8193))
	dec.DisallowUnknownFields() // rejects any client-supplied user_id
	if dec.Decode(&in) != nil || dec.More() || !enrollmentRe.MatchString(in.EnrollmentID) || !tokenRe.MatchString(in.AccessToken) || len(in.InstitutionName) > 200 {
		fail(w, 400, "INVALID_REQUEST")
		return
	}
	if !a.allow(user) {
		fail(w, 429, "RATE_LIMITED")
		return
	}
	ctx, cancel := context.WithTimeout(r.Context(), 30*time.Second)
	defer cancel()
	if err := a.provider.Verify(ctx, in.AccessToken); err != nil {
		if errors.Is(err, ErrProviderAuth) {
			fail(w, 400, "INVALID_ENROLLMENT")
		} else {
			fail(w, 502, "PROVIDER_UNAVAILABLE")
		}
		return
	}
	id, err := newUUID()
	if err != nil {
		fail(w, 500, "INTERNAL")
		return
	}
	blob, err := a.keys.Seal(user, id, []byte(in.AccessToken))
	if err != nil {
		fail(w, 500, "INTERNAL")
		return
	}
	c, created, err := a.store.Enroll(ctx, EnrollInput{UserID: user, ID: id, Provider: "TELLER", ProviderConnectionID: in.EnrollmentID, InstitutionName: in.InstitutionName, Blob: blob, KeyVersion: a.keys.Version()})
	if err != nil {
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
	// Revoke first so a failed revoke leaves the credential available for retry; an already-invalid token is fine.
	if token, err := a.keys.Open(user, id, blob); err == nil {
		if err := a.provider.Revoke(ctx, string(token)); err != nil && !errors.Is(err, ErrProviderAuth) {
			fail(w, 502, "PROVIDER_UNAVAILABLE")
			return
		}
	}
	if _, err := a.store.Delete(ctx, user, id); err != nil {
		fail(w, 503, "UNAVAILABLE")
		return
	}
	w.Header().Set("Cache-Control", "no-store")
	w.WriteHeader(204)
}
