package receipts

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestGeminiStructuredExtractionAndSecretHeader(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("x-goog-api-key") != "test-only-not-a-real-key" || r.URL.RawQuery != "" {
			t.Error("key missing header or exposed in URL")
		}
		var in map[string]any
		if json.NewDecoder(r.Body).Decode(&in) != nil {
			t.Fatal("bad request")
		}
		config := in["generationConfig"].(map[string]any)
		if config["responseMimeType"] != "application/json" || config["responseSchema"] == nil || config["responseFormat"] != nil || config["responseJsonSchema"] != nil {
			t.Error("missing schema")
		}
		p := validDraft().Proposal
		for n := range p.Items {
			p.Items[n].ID = ""
			p.Items[n].ParticipantID = ""
		}
		b, _ := json.Marshal(p)
		_ = json.NewEncoder(w).Encode(map[string]any{"candidates": []any{map[string]any{"finishReason": "STOP", "content": map[string]any{"parts": []any{map[string]string{"text": string(b)}}}}}})
	}))
	defer server.Close()
	g, _ := NewGemini("test-only-not-a-real-key")
	g.endpoint = server.URL
	g.Client = server.Client()
	p, e := g.Extract(context.Background(), []byte("fake image"), "image/png")
	if e != nil || len(p.Items) != 2 || p.Items[0].ID != "item-001" || p.Items[0].ParticipantID != "" {
		t.Fatalf("%+v %v", p, e)
	}
}
func TestGeminiFailureBounds(t *testing.T) {
	for _, tc := range []struct {
		name, body string
		status     int
		code       string
		retry      bool
	}{{"access", "secret upstream details", 403, "VISION_ACCESS_REQUIRED", false}, {"missing model", "", 404, "VISION_MODEL_UNAVAILABLE", false}, {"overload", "", 429, "VISION_RATE_LIMITED", true}, {"server", "", 500, "VISION_UNAVAILABLE", true}, {"partial", `{"candidates":[{"finishReason":"MAX_TOKENS"}]}`, 200, "VISION_INVALID_OUTPUT", false}, {"malformed", `{`, 200, "VISION_INVALID_OUTPUT", false}, {"oversized", strings.Repeat("x", 262145), 200, "VISION_INVALID_OUTPUT", false}, {"unknown fields", `{"candidates":[{"finishReason":"STOP","content":{"parts":[{"text":"{\"execute\":\"steal secrets\"}"}]}}]}`, 200, "VISION_INVALID_OUTPUT", false}} {
		t.Run(tc.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(tc.status)
				_, _ = io.WriteString(w, tc.body)
			}))
			defer server.Close()
			g, _ := NewGemini("test-only")
			g.endpoint = server.URL
			g.Client = server.Client()
			_, e := g.Extract(context.Background(), nil, "image/png")
			var pe *ProviderError
			if !errors.As(e, &pe) || pe.Code != tc.code || pe.Retryable != tc.retry {
				t.Fatalf("%v", e)
			}
		})
	}
}
func TestGeminiNoKeyAndCancelled(t *testing.T) {
	if _, e := NewGemini(""); e == nil {
		t.Fatal("missing key accepted")
	}
	g, _ := NewGemini("test-only")
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	_, e := g.Extract(ctx, nil, "image/png")
	if e == nil {
		t.Fatal("cancel ignored")
	}
}

func TestGeminiSafeFailureClassification(t *testing.T) {
	cases := []struct {
		status             int
		body, code, reason string
	}{
		{400, `{"error":{"status":"INVALID_ARGUMENT","message":"private upstream details","details":[{"reason":"API_KEY_INVALID"}]}}`, "VISION_ACCESS_REQUIRED", "API_KEY_INVALID"},
		{400, `{"error":{"status":"INVALID_ARGUMENT","message":"private upstream details"}}`, "VISION_REQUEST_REJECTED", "INVALID_ARGUMENT"},
		{429, `{"error":{"status":"RESOURCE_EXHAUSTED","details":[{"reason":"QUOTA_EXCEEDED"}]}}`, "VISION_RATE_LIMITED", "QUOTA_EXCEEDED"},
		{403, `{"error":{"status":"secret-private-message","details":[{"reason":"secret-private-message"}]}}`, "VISION_ACCESS_REQUIRED", "UNSPECIFIED"},
	}
	for _, tc := range cases {
		resp := &http.Response{StatusCode: tc.status, Body: io.NopCloser(strings.NewReader(tc.body))}
		e := geminiFailure(resp)
		if e.Code != tc.code || e.Reason != tc.reason || e.HTTPStatus != tc.status || strings.Contains(e.Error(), "private") {
			t.Fatalf("unsafe/incorrect failure %v", e)
		}
	}
}

func TestGeminiBadRequestFieldPathsAreWhitelisted(t *testing.T) {
	body := `{"error":{"status":"INVALID_ARGUMENT","message":"private","details":[{"fieldViolations":[{"field":"generation_config","description":"Unknown name \"responseFormat\""},{"field":"secret-private-field","description":"private"}]}]}}`
	e := geminiFailure(&http.Response{StatusCode: 400, Body: io.NopCloser(strings.NewReader(body))})
	if e.Reason != "UNSUPPORTED_RESPONSE_FORMAT" || len(e.Fields) != 1 || e.Fields[0] != "generation_config" || strings.Contains(e.Error(), "private") {
		t.Fatalf("%v", e)
	}
}

func TestProviderSchemaAvoidsLargeConstraintStateSpaces(t *testing.T) {
	b, _ := json.Marshal(proposalSchema())
	for _, key := range []string{`"maxItems"`, `"maximum"`, `"minimum"`, `"additionalProperties"`} {
		if strings.Contains(string(b), key) {
			t.Fatal("provider constraints may create excessive schema states; keep strict bounds in Go")
		}
	}
	// Simplified provider schema never relaxes authoritative Go validation.
	p := validDraft().Proposal
	p.Items[0].Amount = MaxMoney + 1
	if ValidateProposal(p) == nil {
		t.Fatal("domain monetary limit relaxed")
	}
}

func TestGeminiModelSchemaUsesVerifiedWireContract(t *testing.T) {
	schema := proposalSchema()
	if schema["type"] != "OBJECT" {
		t.Fatal("wrong typed responseSchema")
	}
	props := schema["properties"].(map[string]any)
	if props["total_minor"].(map[string]any)["type"] != "INTEGER" {
		t.Fatal("money must be integer")
	}
	provider, _ := NewGemini("fake-key-only")
	if !strings.Contains(provider.endpoint, "/models/gemini-3.5-flash-lite:generateContent") {
		t.Fatal("unapproved model")
	}
}
