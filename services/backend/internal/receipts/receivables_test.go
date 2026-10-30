package receipts

import (
	"encoding/base64"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

// VP-020: the paid toggle has no receipt version, but review/finalize/retry still require one.
func TestMarkPaidNeedsPaidNotVersion(t *testing.T) {
	clock := time.Now()
	a, db := sessionAPI(t, &clock)
	for body, want := range map[string]int{`{"paid":true}`: 200, `{"paid":false}`: 200, `{}`: 400, `{"paid":"yes"}`: 400, `{"paid":null}`: 400} {
		r := httptest.NewRequest("POST", "/api/v1/receivables/"+rid+"/paid", strings.NewReader(body))
		r.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		a.receivables(w, r, ownerA)
		if w.Code != want {
			t.Errorf("paid body %s: got %d, want %d (%s)", body, w.Code, want, w.Body.String())
		}
	}
	if !strings.Contains(strings.Join(db.bodies, ""), `"p_paid":true`) {
		t.Fatal("paid flag not forwarded to the database")
	}
	for _, route := range []string{"review", "finalize", "retry"} {
		r := httptest.NewRequest("POST", "/api/v1/receipts/"+rid+"/"+route, strings.NewReader(`{}`))
		r.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		a.serve(w, r, ownerA)
		if w.Code != 400 || !strings.Contains(w.Body.String(), "version is required") {
			t.Errorf("%s without version: %d %s", route, w.Code, w.Body.String())
		}
	}
}

// VP-023: filters are validated and bounded before they reach the database.
func TestReceivableListQueryValidation(t *testing.T) {
	clock := time.Now()
	a, db := sessionAPI(t, &clock)
	cursor := base64.RawURLEncoding.EncodeToString([]byte(`{"created_at":"2026-10-29T10:00:00Z","id":"` + rid + `"}`))
	for query, want := range map[string]int{
		"": 200, "?q=ab": 200, "?q=a": 400, "?q=" + strings.Repeat("x", 65): 400, "?status=PAID&from=2026-01-01&to=2026-02-01": 200,
		"?status=BOGUS": 400, "?from=2026-13-01": 400, "?from=2026-02-01&to=2026-01-01": 400, "?from=2019-01-01&to=2026-01-01": 400,
		"?limit=0": 400, "?limit=x": 400, "?limit=999": 200, "?cursor=garbage": 400, "?cursor=" + cursor: 200,
		"?foo=1": 400, "?status=OWED&status=PAID": 400,
	} {
		w := httptest.NewRecorder()
		a.receivables(w, httptest.NewRequest("GET", "/api/v1/receivables"+query, nil), ownerA)
		if w.Code != want {
			t.Errorf("query %q: got %d, want %d (%s)", query, w.Code, want, w.Body.String())
		}
	}
	all := strings.Join(db.bodies, "")
	if !strings.Contains(all, `"p_limit":50`) || strings.Contains(all, `"p_limit":999`) || !strings.Contains(all, `"p_owner":"`+ownerA+`"`) {
		t.Fatalf("limit not clamped or owner not forwarded: %s", all)
	}
}

// VP-021: only VENMO or ZELLE, only with a valid share token.
func TestPaymentMethodValidationAndToken(t *testing.T) {
	clock := time.Now()
	a, db := sessionAPI(t, &clock)
	_, created := call(a, ownerA, "POST", "/session")
	tok := created["token"].(string)
	for body, want := range map[string]int{
		`{"token":"` + tok + `","participant_id":"` + rid + `","method":"ZELLE"}`:  200,
		`{"token":"` + tok + `","participant_id":"` + rid + `","method":"PAYPAL"}`: 400,
		`{"token":"` + tok + `","participant_id":"nope","method":"VENMO"}`:         400,
		`{"token":"wrong","participant_id":"` + rid + `","method":"VENMO"}`:        401,
	} {
		w := httptest.NewRecorder()
		r := httptest.NewRequest("POST", "/api/v1/split/payment-method", strings.NewReader(body))
		r.Header.Set("Content-Type", "application/json")
		a.PaymentMethod(w, r)
		if w.Code != want {
			t.Errorf("%s: got %d, want %d", body, w.Code, want)
		}
	}
	if !strings.Contains(strings.Join(db.bodies, ""), `"p_method":"ZELLE"`) {
		t.Fatal("method not forwarded")
	}
}
