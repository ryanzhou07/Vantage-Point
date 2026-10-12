package foundation

import (
	"encoding/json"
	"github.com/golang-jwt/jwt/v5"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

const one = "00000000-0000-4000-8000-000000000001"
const two = "00000000-0000-4000-8000-000000000002"

var testKey = []byte("synthetic-unit-test-key-never-real")

func token(t *testing.T, base, id string, edit func(*jwt.RegisteredClaims), key []byte) string {
	t.Helper()
	c := jwt.RegisteredClaims{Subject: id, Issuer: base + "/auth/v1", Audience: jwt.ClaimStrings{"authenticated"}, ExpiresAt: jwt.NewNumericDate(time.Now().Add(time.Hour))}
	if edit != nil {
		edit(&c)
	}
	s, e := jwt.NewWithClaims(jwt.SigningMethodHS256, c).SignedString(key)
	if e != nil {
		t.Fatal(e)
	}
	return s
}
func fixture(t *testing.T) (*API, *httptest.Server, *atomic.Int32) {
	t.Helper()
	var calls atomic.Int32
	var server *httptest.Server
	server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls.Add(1)
		claims := jwt.RegisteredClaims{}
		parsed, e := jwt.ParseWithClaims(strings.TrimPrefix(r.Header.Get("Authorization"), "Bearer "), &claims, func(tok *jwt.Token) (any, error) { return testKey, nil }, jwt.WithValidMethods([]string{"HS256"}), jwt.WithIssuer(server.URL+"/auth/v1"), jwt.WithAudience("authenticated"), jwt.WithExpirationRequired())
		if e != nil || !parsed.Valid {
			w.WriteHeader(401)
			return
		}
		if r.Header.Get("apikey") != "fixture-public" {
			w.WriteHeader(403)
			return
		}
		if r.URL.Path == "/auth/v1/user" {
			_ = json.NewEncoder(w).Encode(map[string]any{"id": claims.Subject})
			return
		}
		if r.URL.Path == "/rest/v1/profiles" {
			if r.URL.Query().Get("id") != "eq."+claims.Subject {
				w.WriteHeader(403)
				return
			}
			_ = json.NewEncoder(w).Encode([]Profile{{ID: claims.Subject, DisplayName: claims.Subject, CreatedAt: time.Now(), UpdatedAt: time.Now()}})
			return
		}
		w.WriteHeader(404)
	}))
	t.Cleanup(server.Close)
	a, e := New(server.URL, "fixture-public", "http://localhost:3000")
	if e != nil {
		t.Fatal(e)
	}
	return a, server, &calls
}
func request(a *API, bearer, query string) *httptest.ResponseRecorder {
	r := httptest.NewRequest("GET", "/api/v1/me"+query, nil)
	if bearer != "" {
		r.Header.Set("Authorization", "Bearer "+bearer)
	}
	w := httptest.NewRecorder()
	a.Protect(http.HandlerFunc(a.Me)).ServeHTTP(w, r)
	return w
}
func TestVerifiedIdentityAndPrivateMe(t *testing.T) {
	a, s, _ := fixture(t)
	for _, id := range []string{one, two, one} {
		w := request(a, token(t, s.URL, id, nil, testKey), "")
		if w.Code != 200 || !strings.Contains(w.Body.String(), id) {
			t.Fatalf("status %d body %s", w.Code, w.Body.String())
		}
		if w.Header().Get("Cache-Control") != "no-store" {
			t.Fatal("private response cached")
		}
	}
	ctx := httptest.NewRequest("GET", "/", nil).Context()
	if _, ok := UserID(ctx); ok {
		t.Fatal("unauthenticated context has identity")
	}
}
func TestRejectedTokensAndInputs(t *testing.T) {
	a, s, calls := fixture(t)
	cases := []struct {
		name string
		edit func(*jwt.RegisteredClaims)
		key  []byte
	}{
		{"forged signature", nil, []byte("forged")}, {"expired", func(c *jwt.RegisteredClaims) { c.ExpiresAt = jwt.NewNumericDate(time.Now().Add(-time.Minute)) }, testKey},
		{"issuer", func(c *jwt.RegisteredClaims) { c.Issuer = "https://evil.invalid/auth/v1" }, testKey}, {"audience", func(c *jwt.RegisteredClaims) { c.Audience = []string{"service_role"} }, testKey},
		{"missing expiry", func(c *jwt.RegisteredClaims) { c.ExpiresAt = nil }, testKey}, {"future not-before", func(c *jwt.RegisteredClaims) { c.NotBefore = jwt.NewNumericDate(time.Now().Add(time.Hour)) }, testKey},
		{"invalid UUID", func(c *jwt.RegisteredClaims) { c.Subject = "not-a-uuid" }, testKey},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			w := request(a, token(t, s.URL, one, c.edit, c.key), "")
			if w.Code != 401 {
				t.Fatalf("got %d", w.Code)
			}
			if !strings.Contains(w.Body.String(), "request_id") {
				t.Fatal("missing error contract")
			}
		})
	}
	before := calls.Load()
	for _, raw := range []string{"", "garbage", strings.Repeat("x", 8200)} {
		if request(a, raw, "").Code != 401 {
			t.Fatal("bad header accepted")
		}
	}
	if calls.Load() != before {
		t.Fatal("malformed auth triggered provider")
	}
	if request(a, token(t, s.URL, one, nil, testKey), "?user_id="+two).Code != 400 {
		t.Fatal("forged query accepted")
	}
}
func TestOriginAndContext(t *testing.T) {
	a, s, _ := fixture(t)
	r := httptest.NewRequest("GET", "/api/v1/me", nil)
	r.Header.Set("Authorization", "Bearer "+token(t, s.URL, one, nil, testKey))
	r.Header.Set("Origin", "https://evil.invalid")
	w := httptest.NewRecorder()
	a.Protect(http.HandlerFunc(a.Me)).ServeHTTP(w, r)
	if w.Code != 403 {
		t.Fatal("foreign origin accepted")
	}
	r.Header.Set("Origin", "http://localhost:3000")
	w = httptest.NewRecorder()
	a.Protect(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		id, ok := UserID(r.Context())
		if !ok || id != one {
			t.Fatal("missing verified identity")
		}
		w.WriteHeader(200)
	})).ServeHTTP(w, r)
	if w.Code != 200 || w.Header().Get("Access-Control-Allow-Origin") != "http://localhost:3000" {
		t.Fatal("allowed origin failed")
	}
	r.Method = "OPTIONS"
	r.Header.Del("Authorization")
	w = httptest.NewRecorder()
	a.Protect(http.HandlerFunc(a.Me)).ServeHTTP(w, r)
	if w.Code != 204 || w.Header().Get("Access-Control-Allow-Credentials") != "" {
		t.Fatal("preflight/cookie policy")
	}
	r.Method = "GET"
	r.Header.Set("X-User-ID", two)
	w = httptest.NewRecorder()
	a.Protect(http.HandlerFunc(a.Me)).ServeHTTP(w, r)
	if w.Code != 400 {
		t.Fatal("forged header")
	}
}
func TestBoundedRemoteValidation(t *testing.T) {
	var calls atomic.Int32
	s := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { calls.Add(1); <-r.Context().Done() }))
	defer s.Close()
	a, _ := New(s.URL, "fixture", "http://localhost:3000")
	a.client.Timeout = 30 * time.Millisecond
	raw := token(t, s.URL, one, nil, testKey)
	start := time.Now()
	w := request(a, raw, "")
	if w.Code != 503 || time.Since(start) > time.Second || calls.Load() != 1 {
		t.Fatal("timeout/retry bounds")
	}
	for i := 0; i < cap(a.slots); i++ {
		a.slots <- struct{}{}
	}
	before := calls.Load()
	if request(a, raw, "").Code != 429 || calls.Load() != before {
		t.Fatal("backpressure failed")
	}
}
func TestConcurrentUsers(t *testing.T) {
	a, s, _ := fixture(t)
	var wg sync.WaitGroup
	for i := 0; i < 12; i++ {
		id := one
		if i%2 == 1 {
			id = two
		}
		raw := token(t, s.URL, id, nil, testKey)
		wg.Add(1)
		go func() {
			defer wg.Done()
			w := request(a, raw, "")
			if w.Code != 200 || !strings.Contains(w.Body.String(), id) {
				t.Errorf("cross-user response %d", w.Code)
			}
		}()
	}
	wg.Wait()
}
func TestMismatchAnonymousAndOversizedProvider(t *testing.T) {
	for _, body := range []string{`{"id":"` + two + `"}`, `{"id":"` + one + `","is_anonymous":true}`, strings.Repeat("x", 65537)} {
		s := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { _, _ = w.Write([]byte(body)) }))
		a, _ := New(s.URL, "fixture", "http://localhost:3000")
		w := request(a, token(t, s.URL, one, nil, testKey), "")
		s.Close()
		if w.Code == 200 {
			t.Fatal("invalid upstream identity accepted")
		}
	}
}
func TestConfigurationRejectsUnsafeTargets(t *testing.T) {
	for _, base := range []string{"http://evil.invalid", "https://user:pass@example.com", "https://example.com?key=secret", "file:///tmp/auth"} {
		if _, e := New(base, "fixture", "http://localhost:3000"); e == nil {
			t.Fatal("unsafe target accepted")
		}
	}
}
