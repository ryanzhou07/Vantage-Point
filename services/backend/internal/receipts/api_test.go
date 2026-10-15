package receipts

import (
	"bytes"
	"image"
	"image/color"
	"image/png"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func pngFixture() []byte {
	var b bytes.Buffer
	i := image.NewRGBA(image.Rect(0, 0, 8, 8))
	i.Set(0, 0, color.White)
	_ = png.Encode(&b, i)
	return b.Bytes()
}
func TestImagesContentAndLimits(t *testing.T) {
	if mime, e := ValidateImage(pngFixture()); e != nil || mime != "image/png" {
		t.Fatal(mime, e)
	}
	for _, b := range [][]byte{nil, []byte("<svg onload='alert(1)'/>"), []byte("not actually jpeg"), make([]byte, MaxImageBytes+1), pngFixture()[:30]} {
		if _, e := ValidateImage(b); e == nil {
			t.Fatal("invalid image accepted")
		}
	}
}
func TestReceiptRateLimiterBoundAndExpiry(t *testing.T) {
	a := NewAPI(nil)
	for n := 0; n < 60; n++ {
		if !a.allow("owner") {
			t.Fatal("limited early")
		}
	}
	if a.allow("owner") {
		t.Fatal("unbounded polls")
	}
	a.rates["owner"] = rateEntry{start: time.Now().Add(-2 * time.Minute), count: 60}
	if !a.allow("owner") || !a.allow("other-owner") {
		t.Fatal("expiry or isolation broken")
	}
}
func TestReceiptHandlerCannotBypassFoundation(t *testing.T) {
	a := NewAPI(nil)
	for _, method := range []string{"GET", "POST"} {
		r := httptest.NewRequest(method, "/api/v1/receipts", nil)
		r.Header.Set("X-User-ID", "forged")
		w := httptest.NewRecorder()
		a.ServeHTTP(w, r)
		if w.Code != 401 {
			t.Fatal(w.Code)
		}
	}
}
func TestJSONBoundsAndUnknownFields(t *testing.T) {
	for _, body := range []string{`{"version":1,"owner_id":"forged"}`, `{"version":null}`, `{"version":1,"draft":{"total_minor":null}}`, `{"version":1} {}`, strings.Repeat("x", 131073)} {
		w := httptest.NewRecorder()
		r := httptest.NewRequest("POST", "/", strings.NewReader(body))
		r.Header.Set("Content-Type", "application/json")
		var in struct {
			Version int64 `json:"version"`
		}
		if readJSON(w, r, &in) {
			t.Fatal("invalid JSON accepted")
		}
	}
}
