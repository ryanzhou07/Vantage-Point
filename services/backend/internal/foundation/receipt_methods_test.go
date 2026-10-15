package foundation

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestReceiptMethodsStillVerifyCanonicalIdentity(t *testing.T) {
	a, s, _ := fixture(t)
	called := false
	handler := a.ProtectMethods(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		id, ok := UserID(r.Context())
		if !ok || id != one {
			t.Error("unverified identity")
		}
		called = true
		w.WriteHeader(202)
	}), "GET", "POST")
	for _, valid := range []bool{false, true} {
		called = false
		r := httptest.NewRequest("POST", "/api/v1/receipts", nil)
		r.Header.Set("Origin", "http://localhost:3000")
		if valid {
			r.Header.Set("Authorization", "Bearer "+token(t, s.URL, one, nil, testKey))
		}
		w := httptest.NewRecorder()
		handler.ServeHTTP(w, r)
		if valid && (w.Code != 202 || !called) {
			t.Fatal("valid owner rejected")
		}
		if !valid && (w.Code != 401 || called) {
			t.Fatal("unauthenticated mutation accepted")
		}
	}
	r := httptest.NewRequest("OPTIONS", "/api/v1/receipts", nil)
	r.Header.Set("Origin", "http://localhost:3000")
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, r)
	if w.Code != 204 || w.Header().Get("Access-Control-Allow-Methods") != "GET, POST" || w.Header().Get("Access-Control-Allow-Headers") != "Authorization, Content-Type" {
		t.Fatal("receipt preflight rejected")
	}
	r = httptest.NewRequest("POST", "/api/v1/receipts", nil)
	r.Header.Set("Origin", "https://other.example")
	w = httptest.NewRecorder()
	handler.ServeHTTP(w, r)
	if w.Code != 403 {
		t.Fatal("cross-origin mutation accepted")
	}
}
