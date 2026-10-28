package receipts

import (
	"encoding/json"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func join(a *API, body string) (int, map[string]any) {
	r := httptest.NewRequest("POST", "/api/v1/split/session", strings.NewReader(body))
	r.Header.Set("Content-Type", "application/json")
	r.RemoteAddr = "203.0.113.9:1234"
	w := httptest.NewRecorder()
	a.Join(w, r)
	var out map[string]any
	_ = json.Unmarshal(w.Body.Bytes(), &out)
	return w.Code, out
}

func TestJoinExpiryParticipantsAndNoOwnerLeak(t *testing.T) {
	clock := time.Date(2026, 10, 28, 12, 0, 0, 0, time.UTC)
	a, db := sessionAPI(t, &clock)
	_, created := call(a, ownerA, "POST", "/session")
	tok := created["token"].(string)
	code, out := join(a, `{"token":"`+tok+`"}`)
	people, _ := out["participants"].([]any)
	if code != 200 || len(people) != 1 || out["state"] != "ACTIVE" {
		t.Fatal(code, out)
	}
	if b, _ := json.Marshal(out); strings.Contains(string(b), ownerA) || strings.Contains(string(b), "token_hash") {
		t.Fatal("owner identity or hash leaked to participant")
	}
	if code, out = join(a, `{"token":"`+tok+`","participant_id":"p1"}`); code != 200 || out["participant"] == nil {
		t.Fatal(code, out)
	}
	if code, _ = join(a, `{"token":"`+tok+`","participant_id":"p2"}`); code != 404 {
		t.Fatal("excluded participant selectable", code)
	}
	clock = clock.Add(5 * time.Minute) // exactly TTL: backend rejects
	if code, _ = join(a, `{"token":"`+tok+`"}`); code != 401 {
		t.Fatal("join accepted at expiry", code)
	}
	if code, _ = join(a, `{"token":"nope"}`); code != 401 {
		t.Fatal(code)
	}
	for _, b := range db.bodies {
		if strings.Contains(b, tok) {
			t.Fatal("raw token reached storage")
		}
	}
}

func TestJoinRateLimitedAndCORS(t *testing.T) {
	clock := time.Now()
	a, _ := sessionAPI(t, &clock)
	a.Origin = "https://app.example.test"
	for n := 0; n < 30; n++ {
		join(a, `{"token":"x"}`)
	}
	if code, _ := join(a, `{"token":"x"}`); code != 429 {
		t.Fatal("join not limited", code)
	}
	r := httptest.NewRequest("OPTIONS", "/", nil)
	r.Header.Set("Origin", a.Origin)
	w := httptest.NewRecorder()
	a.Join(w, r)
	if w.Code != 204 || w.Header().Get("Access-Control-Allow-Origin") != a.Origin || w.Header().Get("Access-Control-Allow-Credentials") != "" {
		t.Fatal(w.Code, w.Header())
	}
}
