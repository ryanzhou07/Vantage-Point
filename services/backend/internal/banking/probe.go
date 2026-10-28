package banking

// REMOVABLE (VP-009c): delete this file and the MustRegisterProbe line (plus its now-unused import)
// in cmd/api/main.go before production readiness. Nothing else may import this file's symbols.

import (
	"context"
	"encoding/json"
	"errors"
	"log"
	"net/http"
	"sync"
	"time"

	"github.com/ryanzhou07/Vantage-Point/services/backend/internal/foundation"
)

const (
	probeInterval = 10 * time.Second // minimum gap between probe calls per user
	probeMaxUsers = 1000
)

// probeGuard enforces the startup rules; any violation must stop the API.
func probeGuard(getenv func(string) string) (enabled bool, err error) {
	if getenv("TELLER_PROBE_ENABLED") != "true" {
		return false, nil
	}
	if app := getenv("APP_ENV"); app == "" || app == "production" {
		return false, errors.New("banking: Teller probe must not be enabled in production (APP_ENV unset or production)")
	}
	if env, err := ParseEnv(getenv("TELLER_ENV")); err != nil || env != EnvSandbox {
		return false, errors.New("banking: Teller probe requires TELLER_ENV=sandbox")
	}
	if getenv("TELLER_PROBE_TOKEN") == "" {
		return false, errors.New("banking: TELLER_PROBE_TOKEN required for the probe")
	}
	return true, nil
}

// MustRegisterProbe registers GET /api/v1/banking/probe only when the guard allows; it exits on a violation.
func MustRegisterProbe(mux *http.ServeMux, auth *foundation.API, getenv func(string) string) {
	on, err := probeGuard(getenv)
	if err != nil {
		log.Fatal(err)
	}
	if !on {
		return
	}
	ctx := context.Background()
	src, err := NewSecretSource(ctx, getenv)
	if err != nil {
		log.Fatal(err)
	}
	client, err := NewTellerClient(ctx, EnvSandbox, src)
	if err != nil {
		log.Fatal(err)
	}
	mux.Handle("/api/v1/banking/probe", auth.Protect(&probeHandler{p: client, token: getenv("TELLER_PROBE_TOKEN"), userID: foundation.UserID, seen: map[string]time.Time{}}))
}

type probeHandler struct {
	p      Provider
	token  string
	userID func(context.Context) (string, bool)
	mu     sync.Mutex
	seen   map[string]time.Time
}

func (h *probeHandler) allow(user string) bool {
	h.mu.Lock()
	defer h.mu.Unlock()
	now := time.Now()
	if len(h.seen) >= probeMaxUsers {
		for u, t := range h.seen {
			if now.Sub(t) >= probeInterval {
				delete(h.seen, u)
			}
		}
		if len(h.seen) >= probeMaxUsers {
			return false
		}
	}
	if t, ok := h.seen[user]; ok && now.Sub(t) < probeInterval {
		return false
	}
	h.seen[user] = now
	return true
}

func (h *probeHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Cache-Control", "no-store")
	reply := func(status int, body map[string]any) {
		w.WriteHeader(status)
		_ = json.NewEncoder(w).Encode(body)
	}
	user, ok := h.userID(r.Context())
	if !ok {
		reply(401, map[string]any{"ok": false, "error": "UNAUTHENTICATED"})
		return
	}
	if !h.allow(user) {
		reply(429, map[string]any{"ok": false, "error": "RATE_LIMITED"})
		return
	}
	ctx, cancel := context.WithTimeout(r.Context(), 15*time.Second)
	defer cancel()
	switch err := h.p.Verify(ctx, h.token); {
	case err == nil:
		reply(200, map[string]any{"ok": true})
	case errors.Is(err, ErrProviderAuth):
		reply(502, map[string]any{"ok": false, "error": "PROVIDER_AUTH"})
	case errors.Is(err, ErrProviderRateLimited):
		reply(502, map[string]any{"ok": false, "error": "PROVIDER_RATE_LIMITED"})
	default:
		reply(502, map[string]any{"ok": false, "error": "PROVIDER_UNAVAILABLE"})
	}
}
