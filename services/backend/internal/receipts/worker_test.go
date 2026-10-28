package receipts

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
)

type testExtractor struct {
	calls atomic.Int32
	err   error
}

func (e *testExtractor) Extract(context.Context, []byte, string) (Proposal, error) {
	e.calls.Add(1)
	return validDraft().Proposal, e.err
}
func TestWorkerOneDeliveryFinishesOrSchedulesBoundedRetry(t *testing.T) {
	for _, retryable := range []bool{false, true} {
		t.Run(map[bool]string{false: "permanent", true: "transient"}[retryable], func(t *testing.T) {
			var claims atomic.Int32
			var finished bool
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				switch r.URL.Path {
				case "/rest/v1/rpc/receipt_claim":
					if claims.Add(1) > 1 {
						_, _ = w.Write([]byte("null"))
						return
					}
					_ = json.NewEncoder(w).Encode(Job{Receipt: Record{ID: "receipt", ImagePath: "owner/image.png", ImageMIME: "image/png"}, Lease: "lease", Attempts: 1})
				case "/storage/v1/object/authenticated/receipt-images/owner/image.png":
					_, _ = w.Write(pngFixture())
				case "/rest/v1/rpc/receipt_finish":
					var body map[string]any
					_ = json.NewDecoder(r.Body).Decode(&body)
					if body["p_id"] != "receipt" || body["p_lease"] != "lease" || body["p_retry"] != retryable || body["p_error"] != "VISION_ACCESS_REQUIRED" || body["p_proposal"] != nil {
						t.Error("unsafe finish payload")
					}
					finished = true
					_, _ = w.Write([]byte("true"))
				default:
					w.WriteHeader(404)
				}
			}))
			defer server.Close()
			store, _ := NewStore(server.URL, "test-service-only")
			provider := &testExtractor{err: &ProviderError{Code: "VISION_ACCESS_REQUIRED", Retryable: retryable}}
			worker := Worker{Store: store, Provider: provider}
			if e := worker.Step(context.Background()); e != nil {
				t.Fatal(e)
			}
			if e := worker.Step(context.Background()); e != nil {
				t.Fatal(e)
			}
			if !finished || provider.calls.Load() != 1 {
				t.Fatal("duplicate/unfinished extraction")
			}
		})
	}
}
func TestWorkerNoConfiguration(t *testing.T) {
	w := Worker{}
	if w.Run(context.Background()) == nil {
		t.Fatal("missing configuration accepted")
	}
}
