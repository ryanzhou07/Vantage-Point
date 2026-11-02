package banking

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

type memRow struct {
	user, item string
	c          Connection
	blob       []byte
}

type memStore struct{ rows []*memRow }

func (m *memStore) Enroll(_ context.Context, in EnrollInput) (Connection, bool, error) {
	for _, r := range m.rows {
		if r.user == in.UserID && r.item == in.ProviderConnectionID {
			return r.c, false, nil
		}
	}
	c := Connection{ID: in.ID, Provider: in.Provider, InstitutionName: in.InstitutionName, Status: "ACTIVE"}
	m.rows = append(m.rows, &memRow{in.UserID, in.ProviderConnectionID, c, in.Blob})
	return c, true, nil
}
func (m *memStore) List(_ context.Context, u string) ([]Connection, error) {
	var out []Connection
	for _, r := range m.rows {
		if r.user == u {
			out = append(out, r.c)
		}
	}
	return out, nil
}
func (m *memStore) Get(_ context.Context, u, id string) (Connection, bool, error) {
	for _, r := range m.rows {
		if r.user == u && r.c.ID == id {
			return r.c, true, nil
		}
	}
	return Connection{}, false, nil
}
func (m *memStore) Credential(_ context.Context, u, id string) ([]byte, bool, error) {
	for _, r := range m.rows {
		if r.user == u && r.c.ID == id {
			return r.blob, true, nil
		}
	}
	return nil, false, nil
}
func (m *memStore) Delete(_ context.Context, u, id string) (bool, error) {
	for i, r := range m.rows {
		if r.user == u && r.c.ID == id {
			m.rows = append(m.rows[:i], m.rows[i+1:]...)
			return true, nil
		}
	}
	return false, nil
}

type recProvider struct {
	exchangeErr, removeErr, linkErr, itemErr error
	removed                                  []string
	items                                    int
}

func (p *recProvider) CreateLinkToken(_ context.Context, u string) (LinkToken, error) {
	return LinkToken{Token: "link-for-" + u, ExpiresAt: "2026-01-01T00:00:00Z"}, p.linkErr
}
func (p *recProvider) ExchangePublicToken(_ context.Context, pt string) (Exchange, error) {
	if p.exchangeErr != nil {
		return Exchange{}, p.exchangeErr
	}
	return Exchange{AccessToken: "access-secret-123", ItemID: "item_" + pt}, nil
}
func (p *recProvider) GetItem(context.Context, string) (Item, error) {
	return Item{InstitutionName: "First Platypus"}, p.itemErr
}
func (p *recProvider) RemoveItem(_ context.Context, t string) error {
	p.removed = append(p.removed, t)
	return p.removeErr
}

const (
	userA = "00000000-0000-4000-8000-0000000000a1"
	userB = "00000000-0000-4000-8000-0000000000b2"
)

func setup(t *testing.T) (*API, *memStore, *recProvider) {
	t.Helper()
	st, pr := &memStore{}, &recProvider{}
	a := NewAPI(st, pr, ring(t))
	a.userID = func(ctx context.Context) (string, bool) { u, ok := ctx.Value(testUser{}).(string); return u, ok }
	return a, st, pr
}

type testUser struct{}

func do(a *API, user, method, path, body string) *httptest.ResponseRecorder {
	r := httptest.NewRequest(method, path, strings.NewReader(body))
	if user != "" {
		r = r.WithContext(context.WithValue(r.Context(), testUser{}, user))
	}
	w := httptest.NewRecorder()
	a.ServeHTTP(w, r)
	return w
}

const connectJSON = `{"public_token":"public-sandbox-1"}`

func TestConnectListDisconnectAndRedaction(t *testing.T) {
	a, st, pr := setup(t)
	w := do(a, userA, "POST", "/api/v1/banking/connections", connectJSON)
	if w.Code != 201 || strings.Contains(w.Body.String(), "access-secret") || strings.Contains(w.Body.String(), "credential") {
		t.Fatal(w.Code, w.Body.String())
	}
	var c Connection
	json.Unmarshal(w.Body.Bytes(), &c)
	if c.InstitutionName != "First Platypus" || c.Provider != "plaid" {
		t.Fatal("name/provider", c)
	}
	if bytes.Contains(st.rows[0].blob, []byte("access-secret")) {
		t.Fatal("plaintext stored")
	}
	if got, err := a.keys.Open(userA, c.ID, st.rows[0].blob); err != nil || string(got) != "access-secret-123" {
		t.Fatal("blob not bound", err)
	}
	w2 := do(a, userA, "POST", "/api/v1/banking/connections", connectJSON)
	var c2 Connection
	json.Unmarshal(w2.Body.Bytes(), &c2)
	if w2.Code != 200 || c2.ID != c.ID || len(st.rows) != 1 || len(pr.removed) != 0 {
		t.Fatal("not idempotent", w2.Code)
	}
	if w := do(a, userA, "GET", "/api/v1/banking/connections", ""); !strings.Contains(w.Body.String(), c.ID) {
		t.Fatal(w.Body.String())
	}
	if w := do(a, userA, "DELETE", "/api/v1/banking/connections/"+c.ID, ""); w.Code != 204 || len(st.rows) != 0 {
		t.Fatal("disconnect", w.Code)
	}
	if len(pr.removed) != 1 || pr.removed[0] != "access-secret-123" {
		t.Fatal("not removed at provider")
	}
}

func TestLinkToken(t *testing.T) {
	a, _, pr := setup(t)
	w := do(a, userA, "POST", "/api/v1/banking/link-token", "")
	if w.Code != 200 || !strings.Contains(w.Body.String(), "link-for-"+userA) {
		t.Fatal(w.Code, w.Body.String())
	}
	if w := do(a, userA, "GET", "/api/v1/banking/link-token", ""); w.Code != 405 {
		t.Fatal(w.Code)
	}
	if w := do(a, "", "POST", "/api/v1/banking/link-token", ""); w.Code != 401 {
		t.Fatal(w.Code)
	}
	pr.linkErr = ErrUnavailable
	if w := do(a, userA, "POST", "/api/v1/banking/link-token", ""); w.Code != 502 {
		t.Fatal(w.Code)
	}
}

func TestTwoUserIsolation(t *testing.T) {
	a, st, pr := setup(t)
	var c Connection
	json.Unmarshal(do(a, userA, "POST", "/api/v1/banking/connections", connectJSON).Body.Bytes(), &c)
	path := "/api/v1/banking/connections/" + c.ID
	if w := do(a, userB, "GET", path, ""); w.Code != 404 {
		t.Fatal("B read A", w.Code)
	}
	if w := do(a, userB, "DELETE", path, ""); w.Code != 404 || len(st.rows) != 1 || len(pr.removed) != 0 {
		t.Fatal("B disconnected A", w.Code)
	}
	if w := do(a, userB, "GET", "/api/v1/banking/connections", ""); strings.Contains(w.Body.String(), c.ID) {
		t.Fatal("B listed A")
	}
	if w := do(a, userB, "POST", "/api/v1/banking/connections", connectJSON); w.Code != 201 {
		t.Fatal(w.Code)
	}
}

func TestConnectValidationAndIdentity(t *testing.T) {
	a, st, pr := setup(t)
	if w := do(a, "", "POST", "/api/v1/banking/connections", connectJSON); w.Code != 401 {
		t.Fatal("anonymous", w.Code)
	}
	for _, b := range []string{
		`{"public_token":"t","user_id":"` + userB + `"}`, // body identity rejected
		`{"public_token":""}`, `{"public_token":"has space"}`, `{}`, `not json`, connectJSON + connectJSON,
		`{"public_token":"` + strings.Repeat("a", 513) + `"}`,
	} {
		if w := do(a, userA, "POST", "/api/v1/banking/connections", b); w.Code != 400 {
			t.Fatalf("accepted %q: %d", b, w.Code)
		}
	}
	pr.exchangeErr = ErrInvalidToken
	if w := do(a, userA, "POST", "/api/v1/banking/connections", connectJSON); w.Code != 400 || len(st.rows) != 0 {
		t.Fatal("bad token stored", w.Code)
	}
	pr.exchangeErr = ErrUnavailable
	if w := do(a, userA, "POST", "/api/v1/banking/connections", connectJSON); w.Code != 502 || strings.Contains(w.Body.String(), "access-") {
		t.Fatal(w.Code)
	}
	pr.exchangeErr = nil
	pr.itemErr = ErrUnavailable // name lookup failure is not fatal
	if w := do(a, userA, "POST", "/api/v1/banking/connections", connectJSON); w.Code != 201 {
		t.Fatal(w.Code)
	}
	if w := do(a, userA, "GET", "/api/v1/banking/connections/not-a-uuid", ""); w.Code != 404 {
		t.Fatal(w.Code)
	}
}

type failStore struct{ memStore }

func (failStore) Enroll(context.Context, EnrollInput) (Connection, bool, error) {
	return Connection{}, false, errStore
}

func TestStoreFailureRemovesFreshItem(t *testing.T) {
	pr := &recProvider{}
	a := NewAPI(&failStore{}, pr, ring(t))
	a.userID = func(ctx context.Context) (string, bool) { u, ok := ctx.Value(testUser{}).(string); return u, ok }
	if w := do(a, userA, "POST", "/api/v1/banking/connections", connectJSON); w.Code != 503 || len(pr.removed) != 1 {
		t.Fatal(w.Code, pr.removed)
	}
}

func TestDisconnectProviderFailureKeepsCredential(t *testing.T) {
	a, st, pr := setup(t)
	var c Connection
	json.Unmarshal(do(a, userA, "POST", "/api/v1/banking/connections", connectJSON).Body.Bytes(), &c)
	pr.removeErr = ErrUnavailable
	if w := do(a, userA, "DELETE", "/api/v1/banking/connections/"+c.ID, ""); w.Code != 502 || len(st.rows) != 1 {
		t.Fatal("credential dropped without provider removal", w.Code)
	}
	pr.removeErr = nil // retry succeeds
	if w := do(a, userA, "DELETE", "/api/v1/banking/connections/"+c.ID, ""); w.Code != 204 || len(st.rows) != 0 {
		t.Fatal(w.Code)
	}
}

func TestProviderRateLimit(t *testing.T) {
	a, _, _ := setup(t)
	var last int
	for i := 0; i < providerCallsPerMinute+1; i++ {
		last = do(a, userA, "POST", "/api/v1/banking/link-token", "").Code
	}
	if last != 429 {
		t.Fatal("not limited", last)
	}
	if w := do(a, userB, "POST", "/api/v1/banking/link-token", ""); w.Code == 429 {
		t.Fatal("limit shared across users")
	}
}

func TestHandlerWithoutFoundationRejects(t *testing.T) {
	a := NewAPI(&memStore{}, &recProvider{}, ring(t))
	r := httptest.NewRequest("GET", "/api/v1/banking/connections", nil)
	r.Header.Set("X-User-ID", "forged")
	w := httptest.NewRecorder()
	a.ServeHTTP(w, r)
	if w.Code != http.StatusUnauthorized {
		t.Fatal(w.Code)
	}
}
