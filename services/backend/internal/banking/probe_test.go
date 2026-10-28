package banking

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/ryanzhou07/Vantage-Point/services/backend/internal/foundation"
)

type fakeProvider struct{ err error }

func (f fakeProvider) Verify(context.Context, string) error { return f.err }

func env(m map[string]string) func(string) string { return func(k string) string { return m[k] } }

func TestProbeGuard(t *testing.T) {
	good := map[string]string{"TELLER_PROBE_ENABLED": "true", "TELLER_ENV": "sandbox", "APP_ENV": "development", "TELLER_PROBE_TOKEN": "t"}
	if on, err := probeGuard(env(good)); !on || err != nil {
		t.Fatal(on, err)
	}
	if on, err := probeGuard(env(map[string]string{"TELLER_ENV": "development", "APP_ENV": "production"})); on || err != nil {
		t.Fatal("flag off must be a no-op regardless of env")
	}
	for k, v := range map[string]string{"APP_ENV": "production", "TELLER_ENV": "development", "TELLER_PROBE_TOKEN": ""} {
		bad := map[string]string{}
		for a, b := range good {
			bad[a] = b
		}
		bad[k] = v
		if on, err := probeGuard(env(bad)); on || err == nil {
			t.Fatalf("%s=%q allowed", k, v)
		}
	}
	delete(good, "APP_ENV")
	if _, err := probeGuard(env(good)); err == nil {
		t.Fatal("unset APP_ENV allowed")
	}
}

func TestRouteAbsentWhenFlagOff(t *testing.T) {
	mux := http.NewServeMux()
	MustRegisterProbe(mux, nil, env(map[string]string{}))
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, httptest.NewRequest("GET", "/api/v1/banking/probe", nil))
	if w.Code != 404 {
		t.Fatal(w.Code)
	}
}

func TestProbeHandler(t *testing.T) {
	h := func(err error) *probeHandler {
		return &probeHandler{p: fakeProvider{err}, token: "tok_secret", userID: func(context.Context) (string, bool) { return "u1", true }, seen: map[string]time.Time{}}
	}
	call := func(h *probeHandler) *httptest.ResponseRecorder {
		w := httptest.NewRecorder()
		h.ServeHTTP(w, httptest.NewRequest("GET", "/", nil))
		return w
	}
	if w := call(h(nil)); w.Code != 200 || strings.TrimSpace(w.Body.String()) != `{"ok":true}` {
		t.Fatal(w.Code, w.Body.String())
	}
	if w := call(h(ErrProviderAuth)); w.Code != 502 || !strings.Contains(w.Body.String(), "PROVIDER_AUTH") || strings.Contains(w.Body.String(), "tok_secret") {
		t.Fatal(w.Body.String())
	}
	limited := h(nil)
	call(limited)
	if w := call(limited); w.Code != 429 {
		t.Fatal("not rate limited", w.Code)
	}
	anon := &probeHandler{p: fakeProvider{}, userID: foundation.UserID, seen: map[string]time.Time{}}
	if w := call(anon); w.Code != 401 {
		t.Fatal("unauthenticated served", w.Code)
	}
}

func TestProbeWithTellerClientAgainstFakeServer(t *testing.T) {
	a := newCA(t)
	f := startTeller(t, a, a, ok)
	cc, ck := a.issue(t, time.Now().Add(time.Hour), false)
	c, err := client(t, EnvSandbox, mapSource{SecretTellerCert: string(cc), SecretTellerKey: string(ck)}, f, a.pool, time.Second)
	if err != nil {
		t.Fatal(err)
	}
	h := &probeHandler{p: c, token: "tok", userID: func(context.Context) (string, bool) { return "u", true }, seen: map[string]time.Time{}}
	w := httptest.NewRecorder()
	h.ServeHTTP(w, httptest.NewRequest("GET", "/", nil))
	if w.Code != 200 {
		t.Fatal(w.Code, w.Body.String())
	}
}
