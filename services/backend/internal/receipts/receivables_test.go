package receipts

import (
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
