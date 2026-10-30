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

func outstanding(rows []Receivable) (sum int64) {
	for _, r := range rows {
		if r.Status == "OWED" {
			sum += r.AmountOwedMinor
		}
	}
	return
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
	case parts[0] == "" && r.Method == "GET":
		args, ok := receivableQuery(w, r)
		if !ok {
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

// receivableQuery validates the list filters (VP-023). Unknown parameters are rejected; the cursor is
// opaque to clients and only ever narrows the verified owner's own rows.
func receivableQuery(w http.ResponseWriter, r *http.Request) (map[string]any, bool) {
	bad := func(msg string) (map[string]any, bool) {
		apiError(w, 400, "INVALID_REQUEST", msg)
		return nil, false
	}
	q := r.URL.Query()
	args := map[string]any{"p_status": nil, "p_from": nil, "p_to": nil, "p_q": nil, "p_limit": 25, "p_cursor": nil}
	for key, values := range q {
		if len(values) != 1 {
			return bad("Send each filter once.")
		}
		v := values[0]
		switch key {
		case "status":
			if v != "" && v != "OWED" && v != "PAID" {
				return bad("Status must be OWED or PAID.")
			}
			if v != "" {
				args["p_status"] = v
			}
		case "from", "to":
			if v == "" {
				continue
			}
			if _, e := time.Parse("2006-01-02", v); e != nil {
				return bad("Dates must look like 2026-10-29.")
			}
			args["p_"+key] = v
		case "q":
			v = strings.TrimSpace(v)
			if v == "" {
				continue
			}
			if n := utf8.RuneCountInString(v); n < 2 || n > 64 {
				return bad("Search needs 2 to 64 characters.")
			}
			args["p_q"] = v
		case "limit":
			n := 0
			for _, c := range v {
				if c < '0' || c > '9' || n > 1000 {
					return bad("Limit must be a number.")
				}
				n = n*10 + int(c-'0')
			}
			if n < 1 {
				return bad("Limit must be at least 1.")
			}
			args["p_limit"] = min(n, 50)
		case "cursor":
			if v == "" {
				continue
			}
			raw, e := base64.RawURLEncoding.DecodeString(v)
			var c struct {
				CreatedAt string `json:"created_at"`
				ID        string `json:"id"`
			}
			if e != nil || len(raw) > 200 || json.Unmarshal(raw, &c) != nil || !receiptID.MatchString(c.ID) {
				return bad("That page link is not valid.")
			}
			if _, te := time.Parse(time.RFC3339Nano, c.CreatedAt); te != nil {
				return bad("That page link is not valid.")
			}
			args["p_cursor"] = c
		default:
			return bad("Unknown filter.")
		}
	}
	from, to := "", ""
	from, _ = args["p_from"].(string)
	to, _ = args["p_to"].(string)
	if from != "" && to != "" {
		f, _ := time.Parse("2006-01-02", from)
		t, _ := time.Parse("2006-01-02", to)
		if t.Before(f) || t.Sub(f) > 5*366*24*time.Hour {
			return bad("Choose a date range of at most 5 years, from before to.")
		}
	}
	return args, true
}
