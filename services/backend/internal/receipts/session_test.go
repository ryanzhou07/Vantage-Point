package receipts

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

const (
	rid      = "11111111-1111-4111-8111-111111111111"
	ownerA   = "aaaaaaaa-aaaa-4aaa-8aaa-aaaaaaaaaaaa"
	ownerB   = "bbbbbbbb-bbbb-4bbb-8bbb-bbbbbbbbbbbb"
	itemFree = "22222222-2222-4222-8222-222222222222"
	itemHeld = "33333333-3333-4333-8333-333333333333"
)

// fakeDB emulates the split_session_* RPC semantics and records every request body.
type fakeDB struct {
	bodies []string
	hash   string
	exp    time.Time
	active bool
}

func (f *fakeDB) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	b, _ := io.ReadAll(r.Body)
	f.bodies = append(f.bodies, string(b))
	var in map[string]any
	_ = json.Unmarshal(b, &in)
	fail := func(m string) { w.WriteHeader(400); _, _ = w.Write([]byte(`{"message":"` + m + `"}`)) }
	name := r.URL.Path[strings.LastIndex(r.URL.Path, "/")+1:]
	if name == "receipt_document" {
		if in["p_owner"] != ownerA {
			fail("RECEIPT_NOT_FOUND")
			return
		}
		_, _ = w.Write([]byte(`{"id":"` + rid + `","draft":{"merchant":"Cafe","currency":"USD","participants":[{"id":"p1","name":"Sam","included":true},{"id":"p2","name":"Out","included":false}]}}`))
		return
	}
	row := func() string {
		st := "INVALIDATED"
		if f.active {
			st = "ACTIVE"
		}
		return `{"id":"s1","receipt_id":"` + rid + `","status":"` + st + `","created_at":"x","expires_at":"` + f.exp.Format(time.RFC3339Nano) + `","invalidated_at":null,"owner_id":"` + ownerA + `","token_hash":"` + f.hash + `"}`
	}
	now, _ := time.Parse(time.RFC3339Nano, asString(in["p_now"]))
	switch name {
	case "split_session_open":
		f.hash, f.active = asString(in["p_hash"]), true
		f.exp = now.Add(time.Duration(in["p_ttl_seconds"].(float64)) * time.Second)
	case "split_session_close":
		if !f.active {
			fail("SESSION_NOT_FOUND")
			return
		}
		f.active = false
	case "split_join_add":
		_, _ = w.Write([]byte(`{"id":"p1","name":"` + asString(in["p_name"]) + `"}`))
		return
	case "split_participant_method":
		_, _ = w.Write([]byte(`"VENMO"`))
		return
	case "split_claim":
		if in["p_item"] == itemHeld {
			fail("SPLIT_CLAIM_CONFLICT")
			return
		}
		_, _ = w.Write([]byte(`{}`))
		return
	case "split_session_check":
		if !f.active || in["p_hash"] != f.hash || !now.Before(f.exp) {
			fail("SESSION_INVALID")
			return
		}
	}
	_, _ = w.Write([]byte(row()))
}
func asString(v any) string { s, _ := v.(string); return s }

func sessionAPI(t *testing.T, clock *time.Time) (*API, *fakeDB) {
	db := &fakeDB{}
	srv := httptest.NewServer(db)
	t.Cleanup(srv.Close)
	st, e := NewStore(srv.URL, "service-key-for-test")
	if e != nil {
		t.Fatal(e)
	}
	a := NewAPI(st)
	a.Now = func() time.Time { return *clock }
	return a, db
}
func call(a *API, owner, method, path string) (int, map[string]any) {
	r := httptest.NewRequest(method, "/api/v1/receipts/"+rid+path, nil)
	w := httptest.NewRecorder()
	a.serve(w, r, owner)
	var out map[string]any
	_ = json.Unmarshal(w.Body.Bytes(), &out)
	return w.Code, out
}

func TestSessionExpiryTokenReplacementAndNoRawToken(t *testing.T) {
	clock := time.Date(2026, 10, 28, 12, 0, 0, 0, time.UTC)
	a, db := sessionAPI(t, &clock)
	code, out := call(a, ownerA, "POST", "/session")
	token, _ := out["token"].(string)
	if code != 201 || token == "" || out["state"] != "ACTIVE" {
		t.Fatal(code, out)
	}
	ctx := context.Background()
	if _, e := a.Store.CheckSession(ctx, token, clock.Add(299*time.Second)); e != nil {
		t.Fatal("rejected before expiry", e)
	}
	for _, at := range []time.Duration{300 * time.Second, time.Hour} {
		if _, e := a.Store.CheckSession(ctx, token, clock.Add(at)); e == nil || e.Error() != "SESSION_INVALID" {
			t.Fatal("accepted at/after expiry", at, e)
		}
	}
	clock = clock.Add(301 * time.Second)
	if _, out = call(a, ownerA, "GET", "/session"); out["state"] != "EXPIRED" {
		t.Fatal("status must read EXPIRED", out)
	}
	_, out2 := call(a, ownerA, "POST", "/session")
	if out2["token"] == token {
		t.Fatal("token reused")
	}
	if _, e := a.Store.CheckSession(ctx, token, clock); e == nil {
		t.Fatal("previous token still valid")
	}
	for _, b := range db.bodies {
		if strings.Contains(b, token) || strings.Contains(b, out2["token"].(string)) {
			t.Fatal("raw token sent to storage")
		}
	}
	if !strings.Contains(strings.Join(db.bodies, ""), hashToken(token)) {
		t.Fatal("hash not stored")
	}
}

func TestSessionOwnerOnlyAndClose(t *testing.T) {
	clock := time.Now()
	a, _ := sessionAPI(t, &clock)
	call(a, ownerA, "POST", "/session")
	for _, c := range [][2]string{{"POST", "/session"}, {"GET", "/session"}, {"POST", "/session/close"}, {"POST", "/session/extend"}} {
		if code, _ := call(a, ownerB, c[0], c[1]); code != 404 {
			t.Fatal("non-owner reached session", c, code)
		}
	}
	if code, out := call(a, ownerA, "POST", "/session/close"); code != 200 || out["state"] != "INVALIDATED" {
		t.Fatal(code, out)
	}
	if code, _ := call(a, ownerA, "POST", "/session/close"); code != 404 {
		t.Fatal(code)
	}
}

func TestSessionCreationRateLimitedAndTTLConfig(t *testing.T) {
	clock := time.Now()
	a, _ := sessionAPI(t, &clock)
	for n := 0; n < 5; n++ {
		if code, _ := call(a, ownerA, "POST", "/session"); code != 201 {
			t.Fatal(code)
		}
	}
	if code, _ := call(a, ownerA, "POST", "/session"); code != 429 {
		t.Fatal("creation not limited", code)
	}
	for v, ok := range map[string]bool{"": true, "5m": true, "1m": true, "60m": true, "30s": false, "2h": false, "junk": false} {
		if d, e := ParseSessionTTL(v); (e == nil) != ok || (v == "" && d != 5*time.Minute) {
			t.Fatal(v, d, e)
		}
	}
}
