package receipts

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"sync"
	"time"
)

const maxSummaryPage = 20

type shareRow struct {
	ParticipantID string `json:"participant_id"`
	Name          string `json:"name"`
	TotalMinor    int64  `json:"total_minor"`
}

// SummaryRow amounts are saved draft previews, never receivables: amount_kind says so.
type SummaryRow struct {
	ID               string     `json:"id"`
	Merchant         string     `json:"merchant"`
	Date             string     `json:"date"`
	Status           string     `json:"status"`
	Currency         string     `json:"currency"`
	TotalMinor       int64      `json:"total_minor"`
	AmountKind       string     `json:"amount_kind"` // DRAFT_PREVIEW, FINALIZED or NONE
	Shares           []shareRow `json:"shares"`
	SharesTotalMinor int64      `json:"shares_total_minor"`
	CreatedAt        string     `json:"created_at"`
}

func summarize(r Record) SummaryRow {
	row := SummaryRow{ID: r.ID, Status: r.Status, Currency: "USD", AmountKind: "NONE", Shares: []shareRow{}, CreatedAt: r.CreatedAt}
	var p *Proposal
	names := map[string]string{}
	if r.Draft != nil {
		p = &r.Draft.Proposal
		for _, person := range r.Draft.Participants {
			names[person.ID] = person.Name
		}
	} else if r.Proposal != nil {
		p = r.Proposal
	}
	if p != nil {
		row.Merchant, row.Date, row.Currency, row.TotalMinor = p.Merchant, p.Date, p.Currency, p.Total
	}
	if r.Preview != nil {
		row.AmountKind = "DRAFT_PREVIEW"
		if r.Status == "FINALIZED" {
			row.AmountKind = "FINALIZED"
		}
		for _, s := range r.Preview.Shares {
			row.Shares = append(row.Shares, shareRow{s.ParticipantID, names[s.ParticipantID], s.Total})
			row.SharesTotalMinor += s.Total
		}
	}
	return row
}

func encodeCursor(createdAt, id string) string {
	return base64.RawURLEncoding.EncodeToString([]byte(createdAt + "|" + id))
}
func decodeCursor(c string) (string, string, bool) {
	b, e := base64.RawURLEncoding.DecodeString(c)
	parts := strings.Split(string(b), "|")
	if e != nil || len(parts) != 2 || !receiptID.MatchString(parts[1]) {
		return "", "", false
	}
	if _, e = time.Parse(time.RFC3339Nano, parts[0]); e != nil {
		return "", "", false
	}
	return parts[0], parts[1], true
}

// page lists one bounded keyset page of the owner's receipts, newest first.
func (s *Store) page(ctx context.Context, owner string, limit int, cursor string) ([]Record, string, error) {
	q := url.Values{"select": {"id,created_at"}, "owner_id": {"eq." + owner}, "order": {"created_at.desc,id.desc"}, "limit": {strconv.Itoa(limit + 1)}}
	if cursor != "" {
		at, id, _ := decodeCursor(cursor)
		q.Set("or", "(created_at.lt."+at+",and(created_at.eq."+at+",id.lt."+id+"))")
	}
	b, e := s.request(ctx, "GET", "/rest/v1/receipts?"+q.Encode(), "", nil, 65536)
	if e != nil {
		return nil, "", e
	}
	var keys []struct{ ID, CreatedAt string }
	var raw []map[string]string
	if json.Unmarshal(b, &raw) != nil {
		return nil, "", errors.New("RECEIPT_UNAVAILABLE")
	}
	for _, r := range raw {
		keys = append(keys, struct{ ID, CreatedAt string }{r["id"], r["created_at"]})
	}
	next := ""
	if len(keys) > limit {
		keys = keys[:limit]
		next = encodeCursor(keys[limit-1].CreatedAt, keys[limit-1].ID)
	}
	records := make([]Record, len(keys))
	errs := make([]error, len(keys))
	sem := make(chan struct{}, 4)
	var wg sync.WaitGroup
	for i, k := range keys {
		wg.Add(1)
		sem <- struct{}{}
		go func() {
			defer wg.Done()
			defer func() { <-sem }()
			records[i], errs[i] = s.get(ctx, owner, k.ID)
		}()
	}
	wg.Wait()
	for _, e := range errs {
		if e != nil {
			return nil, "", e
		}
	}
	return records, next, nil
}

// summary serves GET /api/v1/receipts/summary?limit=&cursor= for the verified owner only.
func (a *API) summary(w http.ResponseWriter, r *http.Request, owner string) {
	limit := maxSummaryPage
	if v := r.URL.Query().Get("limit"); v != "" {
		n, e := strconv.Atoi(v)
		if e != nil || n < 1 || n > maxSummaryPage {
			apiError(w, 400, "INVALID_REQUEST", "Use a limit from 1 to 20.")
			return
		}
		limit = n
	}
	cursor := r.URL.Query().Get("cursor")
	if cursor != "" {
		if _, _, ok := decodeCursor(cursor); !ok {
			apiError(w, 400, "INVALID_REQUEST", "Cursor is not valid.")
			return
		}
	}
	records, next, e := a.Store.page(r.Context(), owner, limit, cursor)
	if e != nil {
		storeError(w, e)
		return
	}
	rows := make([]SummaryRow, len(records))
	var draftTotal int64
	for i, rec := range records {
		rows[i] = summarize(rec)
		draftTotal += rows[i].SharesTotalMinor
	}
	// outstanding_minor is unpaid finalized receivables only; draft previews never count.
	// One row is fetched only to get the exact SQL total over all unpaid receivables.
	var owed struct {
		Outstanding int64 `json:"outstanding_minor"`
	}
	if e := a.Store.rpc(r.Context(), "receivables_search", map[string]any{"p_owner": owner, "p_status": "OWED", "p_from": nil, "p_to": nil, "p_q": nil, "p_limit": 1, "p_cursor": nil}, &owed); e != nil {
		storeError(w, e)
		return
	}
	jsonReply(w, 200, map[string]any{"receipts": rows, "next_cursor": next, "limit": limit, "outstanding_minor": owed.Outstanding, "page_draft_total_minor": draftTotal})
}
