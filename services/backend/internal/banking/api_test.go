package banking

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

type memRow struct {
	user, pconn string
	c           Connection
	blob        []byte
}

type memStore struct{ rows []*memRow }

func (m *memStore) Enroll(_ context.Context, in EnrollInput) (Connection, bool, error) {
	for _, r := range m.rows {
		if r.user == in.UserID && r.pconn == in.ProviderConnectionID {
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
	verifyErr, revokeErr error
	revoked              []string
}

func (p *recProvider) Verify(context.Context, string) error { return p.verifyErr }
func (p *recProvider) Revoke(_ context.Context, t string) error {
	p.revoked = append(p.revoked, t)
	return p.revokeErr
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

const enrollJSON = `{"enrollment_id":"enr_1","access_token":"tok_secret_123","institution_name":"Chase"}`

func TestEnrollListDisconnectAndRedaction(t *testing.T) {
	a, st, pr := setup(t)
	w := do(a, userA, "POST", "/api/v1/banking/connections", enrollJSON)
	if w.Code != 201 || strings.Contains(w.Body.String(), "tok_secret") || strings.Contains(w.Body.String(), "credential") {
		t.Fatal(w.Code, w.Body.String())
	}
	var c Connection
	json.Unmarshal(w.Body.Bytes(), &c)
	// Stored blob is ciphertext bound to user and connection id.
	if bytes.Contains(st.rows[0].blob, []byte("tok_secret")) {
		t.Fatal("plaintext stored")
	}
	if got, err := a.keys.Open(userA, c.ID, st.rows[0].blob); err != nil || string(got) != "tok_secret_123" {
		t.Fatal("blob not bound", err)
	}
	// Duplicate enroll returns the same connection.
	w2 := do(a, userA, "POST", "/api/v1/banking/connections", enrollJSON)
	var c2 Connection
	json.Unmarshal(w2.Body.Bytes(), &c2)
	if w2.Code != 200 || c2.ID != c.ID || len(st.rows) != 1 {
		t.Fatal("not idempotent", w2.Code)
	}
	if w := do(a, userA, "GET", "/api/v1/banking/connections", ""); !strings.Contains(w.Body.String(), c.ID) {
		t.Fatal(w.Body.String())
	}
	if w := do(a, userA, "DELETE", "/api/v1/banking/connections/"+c.ID, ""); w.Code != 204 || len(st.rows) != 0 {
		t.Fatal("disconnect", w.Code)
	}
	if len(pr.revoked) != 1 || pr.revoked[0] != "tok_secret_123" {
		t.Fatal("not revoked at provider")
	}
	if _, ok, _ := st.Credential(context.Background(), userA, c.ID); ok {
		t.Fatal("credential survived")
	}
}

func TestTwoUserIsolation(t *testing.T) {
	a, st, pr := setup(t)
	var c Connection
	json.Unmarshal(do(a, userA, "POST", "/api/v1/banking/connections", enrollJSON).Body.Bytes(), &c)
	path := "/api/v1/banking/connections/" + c.ID
	if w := do(a, userB, "GET", path, ""); w.Code != 404 {
		t.Fatal("B read A", w.Code)
	}
	if w := do(a, userB, "DELETE", path, ""); w.Code != 404 || len(st.rows) != 1 || len(pr.revoked) != 0 {
		t.Fatal("B disconnected A", w.Code)
	}
	if w := do(a, userB, "GET", "/api/v1/banking/connections", ""); strings.Contains(w.Body.String(), c.ID) {
		t.Fatal("B listed A")
	}
	// Same Teller enrollment id under B is a separate connection.
	if w := do(a, userB, "POST", "/api/v1/banking/connections", enrollJSON); w.Code != 201 {
		t.Fatal(w.Code)
	}
}

func TestEnrollValidationAndIdentity(t *testing.T) {
	a, st, pr := setup(t)
	if w := do(a, "", "POST", "/api/v1/banking/connections", enrollJSON); w.Code != 401 {
		t.Fatal("anonymous", w.Code)
	}
	for _, b := range []string{
		`{"enrollment_id":"e","access_token":"t","user_id":"` + userB + `"}`, // body identity rejected
		`{"enrollment_id":"bad id","access_token":"t"}`, `{"enrollment_id":"e","access_token":""}`,
		`{"enrollment_id":"e","access_token":"has space"}`, `{}`, `not json`, enrollJSON + enrollJSON,
		`{"enrollment_id":"e","access_token":"` + strings.Repeat("a", 513) + `"}`,
	} {
		if w := do(a, userA, "POST", "/api/v1/banking/connections", b); w.Code != 400 {
			t.Fatalf("accepted %q: %d", b, w.Code)
		}
	}
	pr.verifyErr = ErrProviderAuth
	if w := do(a, userA, "POST", "/api/v1/banking/connections", enrollJSON); w.Code != 400 || len(st.rows) != 0 {
		t.Fatal("bad token stored", w.Code)
	}
	pr.verifyErr = ErrProviderUnavailable
	if w := do(a, userA, "POST", "/api/v1/banking/connections", enrollJSON); w.Code != 502 || strings.Contains(w.Body.String(), "tok_") {
		t.Fatal(w.Code)
	}
	if w := do(a, userA, "GET", "/api/v1/banking/connections/not-a-uuid", ""); w.Code != 404 {
		t.Fatal(w.Code)
	}
}

func TestDisconnectRevokeFailureKeepsCredential(t *testing.T) {
	a, st, pr := setup(t)
	var c Connection
	json.Unmarshal(do(a, userA, "POST", "/api/v1/banking/connections", enrollJSON).Body.Bytes(), &c)
	pr.revokeErr = ErrProviderUnavailable
	if w := do(a, userA, "DELETE", "/api/v1/banking/connections/"+c.ID, ""); w.Code != 502 || len(st.rows) != 1 {
		t.Fatal("credential dropped without revoke", w.Code)
	}
	pr.revokeErr = errors.Join(ErrProviderAuth) // token already invalid at Teller
	if w := do(a, userA, "DELETE", "/api/v1/banking/connections/"+c.ID, ""); w.Code != 204 || len(st.rows) != 0 {
		t.Fatal(w.Code)
	}
}

func TestEnrollRateLimit(t *testing.T) {
	a, _, _ := setup(t)
	var last int
	for i := 0; i < enrollPerMinute+1; i++ {
		last = do(a, userA, "POST", "/api/v1/banking/connections", enrollJSON).Code
	}
	if last != 429 {
		t.Fatal("not limited", last)
	}
	if w := do(a, userB, "POST", "/api/v1/banking/connections", enrollJSON); w.Code == 429 {
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
