package receipts

import (
	"net/http"
	"strings"
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
		rows := []Receivable{}
		if e := a.Store.rpc(r.Context(), "receivables_list", map[string]string{"p_owner": owner}, &rows); e != nil {
			storeError(w, e)
			return
		}
		jsonReply(w, 200, map[string]any{"receivables": rows, "outstanding_minor": outstanding(rows)})
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
