package receipts

import (
	"encoding/base64"
	"encoding/json"
	"net/http"
	"strings"
	"time"
	"unicode/utf8"
)

// Receivable is the owner's manual-tracking record for one participant's finalized share.
type Receivable struct {
	ID              string  `json:"id"`
	ReceiptID       string  `json:"receipt_id"`
	Merchant        string  `json:"merchant"`
	ReceiptDate     string  `json:"receipt_date"`
	ParticipantID   string  `json:"participant_id"`
	ParticipantName string  `json:"participant_name"`
	AmountOwedMinor int64   `json:"amount_owed_minor"`
	Currency        string  `json:"currency"`
	Status          string  `json:"status"`
	PaymentMethod   *string `json:"payment_method"`
	PaidAt          *string `json:"paid_at"`
	CreatedAt       string  `json:"created_at"`
}

// receivables serves GET /api/v1/receivables and POST /api/v1/receivables/{uuid}/paid for the verified owner.
func (a *API) receivables(w http.ResponseWriter, r *http.Request, owner string) {
	if !a.allow(owner) {
		w.Header().Set("Retry-After", "60")
		apiError(w, 429, "RATE_LIMITED", "Wait a minute before refreshing again.")
		return
	}
	if a.Store == nil {
		apiError(w, 503, "RECEIPT_SETUP_REQUIRED", "Receipt service is not configured yet.")
		return
	}
	parts := strings.Split(strings.Trim(strings.TrimPrefix(r.URL.Path, "/api/v1/receivables"), "/"), "/")
	switch {
	case (parts[0] == "" && r.Method == "GET") || (len(parts) == 1 && parts[0] == "search" && r.Method == "POST"):
		var in receivableSearch
		if r.Method == "POST" && !readJSON(w, r, &in, false) {
			return
		}
		args, msg := receivableArgs(in)
		if msg != "" {
			apiError(w, 400, "INVALID_REQUEST", msg)
			return
		}
		args["p_owner"] = owner
		var out struct {
			Receivables []Receivable `json:"receivables"`
			Next        *struct {
				CreatedAt string `json:"created_at"`
				ID        string `json:"id"`
			} `json:"next"`
			Outstanding int64 `json:"outstanding_minor"`
		}
		if e := a.Store.rpc(r.Context(), "receivables_search", args, &out); e != nil {
			storeError(w, e)
			return
		}
		next := ""
		if out.Next != nil {
			b, _ := json.Marshal(out.Next)
			next = base64.RawURLEncoding.EncodeToString(b)
		}
		if out.Receivables == nil {
			out.Receivables = []Receivable{}
		}
		w.Header().Set("Cache-Control", "private, no-store")
		jsonReply(w, 200, map[string]any{"receivables": out.Receivables, "next_cursor": next, "outstanding_minor": out.Outstanding})
	case len(parts) == 2 && parts[1] == "paid" && receiptID.MatchString(parts[0]) && r.Method == "POST":
		var in struct {
			Paid *bool `json:"paid"`
		}
		if !readJSON(w, r, &in, false) {
			return
		}
		if in.Paid == nil {
			apiError(w, 400, "INVALID_REQUEST", "Send paid as true or false.")
			return
		}
		var out map[string]any
		if e := a.Store.rpc(r.Context(), "receivable_set_paid", map[string]any{"p_owner": owner, "p_id": parts[0], "p_paid": *in.Paid}, &out); e != nil {
			storeError(w, e)
			return
		}
		jsonReply(w, 200, out)
	default:
		apiError(w, 404, "NOT_FOUND", "Receivable route not found.")
	}
}

// receivableSearch is the POST /api/v1/receivables/search body. The shared auth middleware rejects every
// URL query string, so filters travel in the body; empty fields mean "no filter".
type receivableSearch struct {
	Status string `json:"status"`
	From   string `json:"from"`
	To     string `json:"to"`
	Q      string `json:"q"`
	Limit  int    `json:"limit"`
	Cursor string `json:"cursor"`
}

// receivableArgs validates and bounds the filters (VP-023). The cursor is opaque to clients and only
// ever narrows the verified owner's own rows. It returns a client-safe message when input is invalid.
func receivableArgs(in receivableSearch) (map[string]any, string) {
	args := map[string]any{"p_status": nil, "p_from": nil, "p_to": nil, "p_q": nil, "p_limit": 25, "p_cursor": nil}
	if in.Status != "" {
		if in.Status != "OWED" && in.Status != "PAID" {
			return nil, "Status must be OWED or PAID."
		}
		args["p_status"] = in.Status
	}
	var from, to time.Time
	for key, v := range map[string]*string{"p_from": &in.From, "p_to": &in.To} {
		if *v == "" {
			continue
		}
		t, e := time.Parse("2006-01-02", *v)
		if e != nil {
			return nil, "Dates must look like 2026-10-29."
		}
		args[key] = *v
		if key == "p_from" {
			from = t
		} else {
			to = t
		}
	}
	if !from.IsZero() && !to.IsZero() && (to.Before(from) || to.Sub(from) > 5*366*24*time.Hour) {
		return nil, "Choose a date range of at most 5 years, from before to."
	}
	if q := strings.TrimSpace(in.Q); q != "" {
		if n := utf8.RuneCountInString(q); n < 2 || n > 64 {
			return nil, "Search needs 2 to 64 characters."
		}
		args["p_q"] = q
	}
	if in.Limit != 0 {
		if in.Limit < 1 {
			return nil, "Limit must be at least 1."
		}
		args["p_limit"] = min(in.Limit, 50)
	}
	if in.Cursor != "" {
		raw, e := base64.RawURLEncoding.DecodeString(in.Cursor)
		var c struct {
			CreatedAt string `json:"created_at"`
			ID        string `json:"id"`
		}
		if e != nil || len(raw) > 200 || json.Unmarshal(raw, &c) != nil || !receiptID.MatchString(c.ID) {
			return nil, "That page link is not valid."
		}
		if _, e := time.Parse(time.RFC3339Nano, c.CreatedAt); e != nil {
			return nil, "That page link is not valid."
		}
		args["p_cursor"] = c
	}
	return args, ""
}
