package receipts

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

const tripID = "11111111-1111-4111-8111-111111111111"

type tripDB struct{ bodies []string }

func (f *tripDB) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	b, _ := io.ReadAll(r.Body)
	f.bodies = append(f.bodies, r.URL.Path+" "+string(b))
	switch r.URL.Path[strings.LastIndex(r.URL.Path, "/")+1:] {
	case "trip_list":
		_, _ = w.Write([]byte(`[{"id":"` + tripID + `","name":"Trip A"}]`))
	case "trip_create":
		_, _ = w.Write([]byte(`{"id":"` + tripID + `","name":"Trip A"}`))
	case "trip_join":
		if !strings.Contains(string(b), hashToken("valid-invite-token-0123456789")) {
			w.WriteHeader(400)
			_, _ = w.Write([]byte(`{"message":"TRIP_INVITE_INVALID"}`))
			return
		}
		_, _ = w.Write([]byte(`{"id":"` + tripID + `","name":"Trip A"}`))
	case "trip_receipt_access":
		_, _ = w.Write([]byte(`{"owner_id":"` + ownerA + `","participant_id":null}`))
	case "receipt_document":
		_, _ = w.Write([]byte(`{"id":"` + rid + `","status":"NEEDS_REVIEW","version":3,"draft":{"merchant":"Cafe","currency":"USD","participants":[{"id":"p1","name":"Sam","included":true}],"items":[{"id":"i1","name":"Soup","amount_minor":500,"participant_ids":["p1"]}]}}`))
	default:
		_, _ = w.Write([]byte(`{"ok":true}`))
	}
}

func tripCall(t *testing.T, method, path, body string) (int, map[string]any, *tripDB) {
	db := &tripDB{}
	srv := httptest.NewServer(db)
	t.Cleanup(srv.Close)
	st, _ := NewStore(srv.URL, "service-key-for-test")
	a := NewAPI(st)
	a.Now = func() time.Time { return time.Date(2026, 11, 22, 12, 0, 0, 0, time.UTC) }
	r := httptest.NewRequest(method, "/api/v1/trips"+path, strings.NewReader(body))
	r.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	a.trips(w, r, ownerA)
	var out map[string]any
	_ = json.Unmarshal(w.Body.Bytes(), &out)
	return w.Code, out, db
}

func TestTripsUserFromContextAndInviteHashed(t *testing.T) {
	if code, out, db := tripCall(t, "POST", "", `{"name":"Trip A"}`); code != 201 || out["name"] != "Trip A" || !strings.Contains(db.bodies[0], `"p_owner":"`+ownerA+`"`) {
		t.Fatal(code, out, db.bodies)
	}
	if _, _, db := tripCall(t, "POST", "", `{"name":"x","owner_id":"evil"}`); len(db.bodies) != 0 {
		t.Fatal("unknown body field reached the database")
	}
	code, out, db := tripCall(t, "POST", "/"+tripID+"/invite", `{}`)
	token, _ := out["token"].(string)
	if code != 201 || len(token) < 20 || strings.Contains(strings.Join(db.bodies, ""), token) || !strings.Contains(db.bodies[0], hashToken(token)) {
		t.Fatal("invite must store only the hash", code, out)
	}
	if code, out, _ = tripCall(t, "POST", "/join", `{"token":"valid-invite-token-0123456789"}`); code != 200 || out["id"] != tripID {
		t.Fatal(code, out)
	}
	if code, _, _ = tripCall(t, "POST", "/join", `{"token":"some-other-invite-token-abcdef"}`); code != 400 {
		t.Fatal("bad invite accepted", code)
	}
	if code, _, _ = tripCall(t, "POST", "/join", `{"token":"short"}`); code != 400 {
		t.Fatal(code)
	}
}

func TestTripReceiptMemberViewAndClaim(t *testing.T) {
	code, out, _ := tripCall(t, "GET", "/"+tripID+"/receipts/"+rid, "")
	items, _ := out["items"].([]any)
	if code != 200 || len(items) != 1 || out["status"] != "NEEDS_REVIEW" || out["participant"] != nil {
		t.Fatal(code, out)
	}
	if b, _ := json.Marshal(out); strings.Contains(string(b), ownerA) {
		t.Fatal("owner identity leaked to member view")
	}
	code, _, db := tripCall(t, "POST", "/"+tripID+"/receipts/"+rid+"/claim", `{"item_id":"`+itemHeld+`","release":false}`)
	if code != 200 || !strings.Contains(db.bodies[0], `"p_user":"`+ownerA+`"`) {
		t.Fatal(code, db.bodies)
	}
	if code, _, _ = tripCall(t, "POST", "/"+tripID+"/receipts/"+rid+"/claim", `{"item_id":"nope"}`); code != 400 {
		t.Fatal(code)
	}
	if code, _, _ = tripCall(t, "DELETE", "/"+tripID+"/members/not-a-uuid", ""); code != 404 {
		t.Fatal(code)
	}
}
