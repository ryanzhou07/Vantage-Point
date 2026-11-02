package banking

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
	"time"
)

func fakePlaid(t *testing.T, h http.HandlerFunc) *Plaid {
	srv := httptest.NewServer(h)
	t.Cleanup(srv.Close)
	return newPlaid(PlaidConfig{Env: "sandbox", ClientID: "cid-1", Secret: "sek-2"}, srv.URL)
}

func TestPlaidHappyPaths(t *testing.T) {
	p := fakePlaid(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("PLAID-CLIENT-ID") != "cid-1" || r.Header.Get("PLAID-SECRET") != "sek-2" {
			t.Error("auth headers missing")
		}
		var in map[string]any
		json.NewDecoder(r.Body).Decode(&in)
		switch r.URL.Path {
		case "/link/token/create":
			if in["user"].(map[string]any)["client_user_id"] != "u1" {
				t.Error("client_user_id not forwarded")
			}
			w.Write([]byte(`{"link_token":"link-x","expiration":"2026-01-01T00:00:00Z"}`))
		case "/item/public_token/exchange":
			w.Write([]byte(`{"access_token":"access-x","item_id":"item-1"}`))
		case "/item/get":
			w.Write([]byte(`{"item":{"item_id":"item-1","institution_id":"ins_1","institution_name":"First Platypus"}}`))
		case "/item/remove":
			w.Write([]byte(`{"request_id":"r"}`))
		}
	})
	ctx := context.Background()
	if lt, err := p.CreateLinkToken(ctx, "u1"); err != nil || lt.Token != "link-x" {
		t.Fatal(lt, err)
	}
	if ex, err := p.ExchangePublicToken(ctx, "public-x"); err != nil || ex.ItemID != "item-1" || ex.AccessToken != "access-x" {
		t.Fatal(ex, err)
	}
	if it, err := p.GetItem(ctx, "access-x"); err != nil || it.InstitutionName != "First Platypus" {
		t.Fatal(it, err)
	}
	if err := p.RemoveItem(ctx, "access-x"); err != nil {
		t.Fatal(err)
	}
}

func TestPlaidErrorMappingAndRedaction(t *testing.T) {
	for _, tc := range []struct {
		status int
		body   string
		want   error
	}{
		{400, `{"error_type":"INVALID_INPUT","error_code":"INVALID_PUBLIC_TOKEN","error_message":"public-secret-leak"}`, ErrInvalidToken},
		{400, `{"error_type":"ITEM_ERROR","error_code":"ITEM_LOGIN_REQUIRED"}`, ErrLoginRequired},
		{429, `{"error_type":"RATE_LIMIT_EXCEEDED","error_code":"RATE_LIMIT"}`, ErrRateLimited},
		{500, `{"error_type":"API_ERROR","error_code":"INTERNAL_SERVER_ERROR"}`, ErrUnavailable},
		{400, `{"error_type":"INVALID_REQUEST","error_code":"MISSING_FIELDS"}`, ErrRejected},
		{502, `not json`, ErrUnavailable},
	} {
		p := fakePlaid(t, func(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(tc.status); w.Write([]byte(tc.body)) })
		_, err := p.ExchangePublicToken(context.Background(), "public-secret-leak")
		if !errors.Is(err, tc.want) {
			t.Fatalf("%d: got %v want %v", tc.status, err, tc.want)
		}
		if strings.Contains(err.Error(), "leak") || strings.Contains(err.Error(), "sek-2") {
			t.Fatalf("error leaks: %v", err)
		}
	}
}

func TestPlaidRetryBounds(t *testing.T) {
	var n atomic.Int32
	p := fakePlaid(t, func(w http.ResponseWriter, _ *http.Request) { n.Add(1); w.WriteHeader(503) })
	if _, err := p.CreateLinkToken(context.Background(), "u1"); !errors.Is(err, ErrUnavailable) || n.Load() != plaidMaxAttempts {
		t.Fatalf("idempotent call: err=%v attempts=%d", err, n.Load())
	}
	n.Store(0)
	if _, err := p.ExchangePublicToken(context.Background(), "pt"); !errors.Is(err, ErrUnavailable) || n.Load() != 1 {
		t.Fatalf("exchange must not retry: err=%v attempts=%d", err, n.Load())
	}
	n.Store(0)
	p4 := fakePlaid(t, func(w http.ResponseWriter, _ *http.Request) {
		if n.Add(1) < 3 {
			w.WriteHeader(503)
			return
		}
		w.Write([]byte(`{"link_token":"ok"}`))
	})
	if lt, err := p4.CreateLinkToken(context.Background(), "u1"); err != nil || lt.Token != "ok" {
		t.Fatal(lt, err)
	}
}

func TestPlaidRemoveAlreadyGoneAndTimeoutAndSize(t *testing.T) {
	p := fakePlaid(t, func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(400)
		w.Write([]byte(`{"error_type":"INVALID_INPUT","error_code":"INVALID_ACCESS_TOKEN"}`))
	})
	if err := p.RemoveItem(context.Background(), "a"); err != nil {
		t.Fatalf("already removed should succeed: %v", err)
	}
	big := fakePlaid(t, func(w http.ResponseWriter, _ *http.Request) {
		w.Write([]byte(`{"link_token":"` + strings.Repeat("a", plaidMaxBody) + `"}`))
	})
	if _, err := big.CreateLinkToken(context.Background(), "u"); !errors.Is(err, ErrUnavailable) {
		t.Fatalf("oversize body: %v", err)
	}
	release := make(chan struct{})
	defer close(release) // httptest.Close waits for handlers
	slow := fakePlaid(t, func(_ http.ResponseWriter, _ *http.Request) { <-release })
	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()
	if _, err := slow.ExchangePublicToken(ctx, "pt"); !errors.Is(err, ErrUnavailable) {
		t.Fatalf("timeout: %v", err)
	}
}
