package receipts

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"strings"
	"time"
)

const inviteTTL = 7 * 24 * time.Hour

// trips serves /api/v1/trips: collections of a user's receipts. A collection with other members
// is a Trip; only members (verified user id, never a bearer link) can read or claim its receipts.
func (a *API) trips(w http.ResponseWriter, r *http.Request, user string) {
	if !a.allow(user) {
		w.Header().Set("Retry-After", "60")
		apiError(w, 429, "RATE_LIMITED", "Wait a minute before refreshing again.")
		return
	}
	if a.Store == nil {
		apiError(w, 503, "RECEIPT_SETUP_REQUIRED", "Receipt service is not configured yet.")
		return
	}
	ctx, now := r.Context(), a.Now()
	p := strings.Split(strings.Trim(strings.TrimPrefix(r.URL.Path, "/api/v1/trips"), "/"), "/")
	if p[0] == "" {
		p = nil
	}
	valid := func(i int) bool { return len(p) > i && receiptID.MatchString(p[i]) }
	var out json.RawMessage
	var e error
	status := 200
	switch {
	case len(p) == 0 && r.Method == "GET":
		e = a.Store.rpc(ctx, "trip_list", map[string]any{"p_user": user}, &out)
		if e == nil {
			out = wrap("trips", out)
		}
	case len(p) == 0 && r.Method == "POST":
		var in struct {
			Name string `json:"name"`
		}
		if !smallJSON(w, r, &in) {
			return
		}
		status = 201
		e = a.Store.rpc(ctx, "trip_create", map[string]any{"p_owner": user, "p_name": in.Name}, &out)
	case len(p) == 1 && p[0] == "join" && r.Method == "POST":
		var in struct {
			Token string `json:"token"`
		}
		if !smallJSON(w, r, &in) {
			return
		}
		if len(in.Token) < 20 || len(in.Token) > 128 || !a.allowKey("tripjoin:"+user, 10) {
			apiError(w, 400, "TRIP_INVITE_INVALID", "This invite link is invalid or has expired.")
			return
		}
		e = a.Store.rpc(ctx, "trip_join", map[string]any{"p_user": user, "p_hash": hashToken(in.Token), "p_now": stamp(now)}, &out)
	case len(p) == 1 && valid(0) && r.Method == "GET":
		e = a.Store.rpc(ctx, "trip_detail", map[string]any{"p_user": user, "p_id": p[0]}, &out)
	case len(p) == 1 && valid(0) && r.Method == "DELETE":
		e = a.Store.rpc(ctx, "trip_delete", map[string]any{"p_owner": user, "p_id": p[0]}, &out)
	case len(p) == 2 && valid(0) && p[1] == "invite" && r.Method == "POST":
		token, hash, ge := newToken()
		if ge != nil {
			apiError(w, 503, "RECEIPT_UNAVAILABLE", "Try again shortly.")
			return
		}
		exp := now.Add(inviteTTL)
		if e = a.Store.rpc(ctx, "trip_invite", map[string]any{"p_owner": user, "p_id": p[0], "p_hash": hash, "p_expires": stamp(exp)}, &out); e == nil {
			status = 201
			out, _ = json.Marshal(map[string]string{"token": token, "expires_at": stamp(exp)})
		}
	case len(p) == 3 && valid(0) && p[1] == "members" && r.Method == "DELETE":
		target := p[2]
		if target == "me" {
			target = user
		} else if !receiptID.MatchString(target) {
			apiError(w, 404, "NOT_FOUND", "Trip route not found.")
			return
		}
		e = a.Store.rpc(ctx, "trip_remove_member", map[string]any{"p_actor": user, "p_id": p[0], "p_user": target}, &out)
	case len(p) >= 3 && valid(0) && p[1] == "receipts" && valid(2) && len(p) <= 4:
		a.tripReceipt(w, r, user, p, now)
		return
	default:
		apiError(w, 404, "NOT_FOUND", "Trip route not found.")
		return
	}
	if e != nil {
		storeError(w, e)
		return
	}
	w.WriteHeader(status)
	_, _ = w.Write(out)
}

// tripReceipt: GET the member's view of one trip receipt, or POST {item_id,release} to claim as themselves.
func (a *API) tripReceipt(w http.ResponseWriter, r *http.Request, user string, p []string, now time.Time) {
	ctx := r.Context()
	if len(p) == 4 && p[3] == "claim" && r.Method == "POST" {
		var in struct {
			ItemID  string `json:"item_id"`
			Release bool   `json:"release"`
		}
		if !smallJSON(w, r, &in) {
			return
		}
		if !receiptID.MatchString(in.ItemID) {
			apiError(w, 400, "INVALID_REQUEST", "Send a valid item ID.")
			return
		}
		var ignored map[string]any
		if e := a.Store.rpc(ctx, "trip_claim", map[string]any{"p_user": user, "p_trip": p[0], "p_receipt": p[2], "p_item": in.ItemID, "p_release": in.Release}, &ignored); e != nil {
			storeError(w, e)
			return
		}
		jsonReply(w, 200, map[string]string{"state": "ok"})
		return
	}
	if len(p) != 3 || r.Method != "GET" {
		apiError(w, 404, "NOT_FOUND", "Trip route not found.")
		return
	}
	var access struct {
		OwnerID       string `json:"owner_id"`
		ParticipantID string `json:"participant_id"`
	}
	if e := a.Store.rpc(ctx, "trip_receipt_access", map[string]any{"p_user": user, "p_trip": p[0], "p_receipt": p[2]}, &access); e != nil {
		storeError(w, e)
		return
	}
	rec, e := a.Store.get(ctx, access.OwnerID, p[2])
	if e != nil {
		storeError(w, e)
		return
	}
	out := map[string]any{"status": rec.Status, "version": rec.Version, "server_time": stamp(now)}
	shareView(rec, access.ParticipantID, nil, out) // a member who has not claimed yet simply has no participant
	jsonReply(w, 200, out)
}

func wrap(key string, v json.RawMessage) json.RawMessage {
	b, _ := json.Marshal(map[string]json.RawMessage{key: v})
	return b
}

// smallJSON reads one bounded JSON object with no unknown fields.
func smallJSON(w http.ResponseWriter, r *http.Request, out any) bool {
	data, err := io.ReadAll(http.MaxBytesReader(w, r.Body, 4096))
	dec := json.NewDecoder(bytes.NewReader(data))
	dec.DisallowUnknownFields()
	if err != nil || r.Header.Get("Content-Type") != "application/json" || dec.Decode(out) != nil {
		apiError(w, 400, "INVALID_REQUEST", "Check the request fields.")
		return false
	}
	return true
}
