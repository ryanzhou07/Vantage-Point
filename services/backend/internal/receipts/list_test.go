package receipts

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"testing"
)

const ownerWithDebts = "dddddddd-dddd-4ddd-8ddd-dddddddddddd"

type fakeReceipt struct{ id, owner, created string }

// listDB emulates the owner-filtered keyset query and receipt_document, recording the owner filters used.
type listDB struct{ rows []fakeReceipt }

func (f *listDB) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if strings.HasPrefix(r.URL.Path, "/rest/v1/receipts") {
		q := r.URL.Query()
		owner := strings.TrimPrefix(q.Get("owner_id"), "eq.")
		var mine []fakeReceipt
		for _, x := range f.rows {
			if x.owner == owner {
				mine = append(mine, x)
			}
		}
		sort.Slice(mine, func(i, j int) bool { return mine[i].created+mine[i].id > mine[j].created+mine[j].id })
		if m := regexp.MustCompile(`created_at\.lt\.([^,]+),and\(created_at\.eq\.[^,]+,id\.lt\.([^)]+)\)`).FindStringSubmatch(q.Get("or")); m != nil {
			var rest []fakeReceipt
			for _, x := range mine {
				if x.created < m[1] || (x.created == m[1] && x.id < m[2]) {
					rest = append(rest, x)
				}
			}
			mine = rest
		}
		var n int
		_, _ = fmt.Sscan(q.Get("limit"), &n)
		if len(mine) > n {
			mine = mine[:n]
		}
		out := []map[string]string{}
		for _, x := range mine {
			out = append(out, map[string]string{"id": x.id, "created_at": x.created})
		}
		_ = json.NewEncoder(w).Encode(out)
		return
	}
	if strings.HasSuffix(r.URL.Path, "/receivables_search") {
		var who map[string]any
		_ = json.NewDecoder(r.Body).Decode(&who)
		if who["p_status"] != "OWED" || who["p_limit"] != float64(1) {
			http.Error(w, `{"message":"summary must ask only for the unpaid total"}`, 400)
			return
		}
		total := 0
		if who["p_owner"] == ownerWithDebts {
			total = 500 // the paid 300 is excluded by the database
		}
		_, _ = w.Write([]byte(`{"receivables":[],"next":null,"outstanding_minor":` + strconv.Itoa(total) + `}`))
		return
	}
	var in map[string]string
	_ = json.NewDecoder(r.Body).Decode(&in)
	for _, x := range f.rows {
		if x.id == in["p_id"] && x.owner == in["p_owner"] {
			_, _ = fmt.Fprintf(w, `{"id":%q,"status":"NEEDS_REVIEW","created_at":%q,"draft":{"merchant":"M-%s","currency":"USD","total_minor":1000,"participants":[{"id":"p1","name":"Sam","included":true},{"id":"p2","name":"Lee","included":true}]},"preview":{"draft":true,"total_minor":1000,"shares":[{"participant_id":"p1","total_minor":667},{"participant_id":"p2","total_minor":333}]}}`, x.id, x.created, x.id[:4])
			return
		}
	}
	w.WriteHeader(400)
	_, _ = w.Write([]byte(`{"message":"RECEIPT_NOT_FOUND"}`))
}

func summaryCall(a *API, owner, query string) (int, map[string]any) {
	r := httptest.NewRequest("GET", "/api/v1/receipts/summary"+query, nil)
	w := httptest.NewRecorder()
	a.serve(w, r, owner)
	var out map[string]any
	_ = json.Unmarshal(w.Body.Bytes(), &out)
	return w.Code, out
}

func TestSummaryIsolationPaginationAndExactSums(t *testing.T) {
	db := &listDB{rows: []fakeReceipt{
		{"aaaaaaa1-0000-4000-8000-000000000001", ownerA, "2026-10-01T10:00:00+00:00"},
		{"aaaaaaa2-0000-4000-8000-000000000002", ownerA, "2026-10-02T10:00:00+00:00"},
		{"aaaaaaa3-0000-4000-8000-000000000003", ownerA, "2026-10-03T10:00:00+00:00"},
		{"bbbbbbb1-0000-4000-8000-000000000004", ownerB, "2026-10-04T10:00:00+00:00"},
	}}
	srv := httptest.NewServer(db)
	defer srv.Close()
	st, _ := NewStore(srv.URL, "key")
	a := NewAPI(st)
	code, out := summaryCall(a, ownerA, "?limit=2")
	rows := out["receipts"].([]any)
	if code != 200 || len(rows) != 2 || out["next_cursor"] == "" {
		t.Fatal(code, out)
	}
	first := rows[0].(map[string]any)
	if first["id"] != "aaaaaaa3-0000-4000-8000-000000000003" || first["amount_kind"] != "DRAFT_PREVIEW" || first["shares_total_minor"].(float64) != 1000 || first["shares_total_minor"] != first["total_minor"] {
		t.Fatal(first)
	}
	if sh := first["shares"].([]any)[0].(map[string]any); sh["name"] != "Sam" || sh["total_minor"].(float64) != 667 {
		t.Fatal(sh)
	}
	if out["outstanding_minor"].(float64) != 0 || out["page_draft_total_minor"].(float64) != 2000 {
		t.Fatal("draft preview must not count as owed", out)
	}
	_, page2 := summaryCall(a, ownerA, "?limit=2&cursor="+out["next_cursor"].(string))
	if r2 := page2["receipts"].([]any); len(r2) != 1 || r2[0].(map[string]any)["id"] != "aaaaaaa1-0000-4000-8000-000000000001" || page2["next_cursor"] != "" {
		t.Fatal(page2)
	}
	if _, other := summaryCall(a, ownerB, ""); len(other["receipts"].([]any)) != 1 || strings.Contains(fmt.Sprint(other), "aaaaaaa") {
		t.Fatal("cross-user leak", other)
	}
	_, none := summaryCall(a, "cccccccc-cccc-4ccc-8ccc-cccccccccccc", "")
	if len(none["receipts"].([]any)) != 0 {
		t.Fatal(none)
	}
	for _, q := range []string{"?limit=0", "?limit=21", "?limit=x", "?cursor=garbage"} {
		if code, _ := summaryCall(a, ownerA, q); code != 400 {
			t.Fatal(q, code)
		}
	}
}

func TestSummaryUsesPathSegmentsAndCountsOnlyUnpaidReceivables(t *testing.T) {
	db := &listDB{rows: []fakeReceipt{{"aaaaaaa1-0000-4000-8000-000000000001", ownerWithDebts, "2026-10-01T10:00:00+00:00"}}}
	srv := httptest.NewServer(db)
	defer srv.Close()
	st, _ := NewStore(srv.URL, "key")
	a := NewAPI(st)
	r := httptest.NewRequest("GET", "/api/v1/receipts/summary/5", nil) // no query string: Foundation rejects them
	w := httptest.NewRecorder()
	a.serve(w, r, ownerWithDebts)
	var out map[string]any
	_ = json.Unmarshal(w.Body.Bytes(), &out)
	if w.Code != 200 || out["limit"].(float64) != 5 || out["outstanding_minor"].(float64) != 500 {
		t.Fatal(w.Code, out)
	}
	w = httptest.NewRecorder()
	a.serve(w, httptest.NewRequest("GET", "/api/v1/receipts/summary/5/x/y", nil), ownerA)
	if w.Code != 404 {
		t.Fatal("extra segments", w.Code)
	}
}
