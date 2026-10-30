package receipts

import (
	"bytes"
	"encoding/json"
	"io"
	"net"
	"net/http"
	"strings"
)

// Join is the public, token-gated participant route. Authorization is the share token
// itself (receipt-scoped, expires with the session); participants are never canonical users.
// Posting again is the participant's authoritative refetch. Tokens are never logged.
func (a *API) Join(w http.ResponseWriter, r *http.Request) {
	var in struct {
		Token         string `json:"token"`
		ParticipantID string `json:"participant_id"`
		Name          string `json:"name"`
	}
	if !a.publicRequest(w, r, "join:", 30, &in) || len(in.ParticipantID) > 64 || len(in.Name) > 400 {
		return
	}
	now := a.Now()
	s, e := a.Store.CheckSession(r.Context(), in.Token, now)
	if e != nil {
		storeError(w, e)
		return
	}
	if in.ParticipantID == "" && strings.TrimSpace(in.Name) != "" {
		var added struct {
			ID string `json:"id"`
		}
		if !a.allowKey("joinadd:"+s.ID, 40) {
			apiError(w, 429, "RATE_LIMITED", "Too many people joining. Wait a minute.")
			return
		}
		if e := a.Store.rpc(r.Context(), "split_join_add", map[string]any{"p_hash": hashToken(in.Token), "p_now": stamp(now), "p_name": in.Name}, &added); e != nil {
			storeError(w, e)
			return
		}
		in.ParticipantID = added.ID
		a.Store.broadcast(r.Context(), s.ID)
	}
	rec, e := a.Store.get(r.Context(), s.OwnerID, s.ReceiptID)
	if e != nil {
		storeError(w, e)
		return
	}
	type person struct {
		ID   string `json:"id"`
		Name string `json:"name"`
	}
	type line struct {
		ID        string `json:"id"`
		Name      string `json:"name"`
		Amount    int64  `json:"amount_minor"`
		ClaimedBy string `json:"claimed_by"`
	}
	out := map[string]any{"state": "ACTIVE", "server_time": stamp(now), "session": map[string]string{"id": s.ID, "expires_at": s.ExpiresAt}}
	people, items := []person{}, []line{}
	merchant, currency := "", ""
	if rec.Draft != nil {
		merchant, currency = SanitizeText(rec.Draft.Merchant), rec.Draft.Currency
		for _, p := range rec.Draft.Participants {
			if p.Included && p.Name != "" {
				people = append(people, person{p.ID, SanitizeText(p.Name)})
				if p.ID == in.ParticipantID {
					out["participant"] = person{p.ID, SanitizeText(p.Name)}
				}
			}
		}
		for _, i := range rec.Draft.Items {
			items = append(items, line{i.ID, SanitizeText(i.Name), i.Amount, i.ParticipantID})
		}
	}
	if in.ParticipantID != "" && out["participant"] == nil {
		apiError(w, 404, "PARTICIPANT_NOT_FOUND", "That name is not on this receipt.")
		return
	}
	out["receipt"], out["participants"], out["items"] = map[string]string{"merchant": merchant, "currency": currency}, people, items
	jsonReply(w, 200, out)
}

// Claim lets a joined participant take or release one whole item line. The database
// serializes racing claims; a held item returns 409 SPLIT_CLAIM_CONFLICT.
func (a *API) Claim(w http.ResponseWriter, r *http.Request) {
	var in struct {
		Token         string `json:"token"`
		ParticipantID string `json:"participant_id"`
		ItemID        string `json:"item_id"`
		Release       bool   `json:"release"`
	}
	if !a.publicRequest(w, r, "claim:", 120, &in) {
		return
	}
	if !receiptID.MatchString(in.ParticipantID) || !receiptID.MatchString(in.ItemID) {
		apiError(w, 400, "INVALID_REQUEST", "Send valid participant and item IDs.")
		return
	}
	now := a.Now()
	s, e := a.Store.CheckSession(r.Context(), in.Token, now)
	if e == nil {
		var ignored map[string]any
		e = a.Store.rpc(r.Context(), "split_claim", map[string]any{"p_hash": hashToken(in.Token), "p_now": stamp(now), "p_participant": in.ParticipantID, "p_item": in.ItemID, "p_release": in.Release}, &ignored)
	}
	if e != nil {
		storeError(w, e)
		return
	}
	a.Store.broadcast(r.Context(), s.ID)
	jsonReply(w, 200, map[string]string{"state": "ok"})
}

// publicRequest runs the shared token-route preamble: headers, CORS, POST-only, per-IP
// rate limit, configured store and a strict bounded JSON body with a non-empty token.
func (a *API) publicRequest(w http.ResponseWriter, r *http.Request, prefix string, limit int, in any) bool {
	h := w.Header()
	h.Set("Cache-Control", "no-store")
	h.Set("Referrer-Policy", "no-referrer")
	h.Set("X-Content-Type-Options", "nosniff")
	if a.Origin != "" {
		h.Set("Access-Control-Allow-Origin", a.Origin)
		h.Set("Vary", "Origin")
		h.Set("Access-Control-Allow-Methods", "POST")
		h.Set("Access-Control-Allow-Headers", "Content-Type")
	}
	if r.Method == "OPTIONS" && a.Origin != "" && r.Header.Get("Origin") == a.Origin {
		w.WriteHeader(204)
		return false
	}
	if r.Method != "POST" {
		apiError(w, 405, "METHOD_NOT_ALLOWED", "Use POST.")
		return false
	}
	ip, _, _ := net.SplitHostPort(r.RemoteAddr)
	if !a.allowKey(prefix+ip, limit) {
		h.Set("Retry-After", "60")
		apiError(w, 429, "RATE_LIMITED", "Too many attempts. Wait a minute.")
		return false
	}
	if a.Store == nil {
		apiError(w, 503, "RECEIPT_SETUP_REQUIRED", "Receipt service is not configured yet.")
		return false
	}
	data, e := io.ReadAll(http.MaxBytesReader(w, r.Body, 1024))
	dec := json.NewDecoder(bytes.NewReader(data))
	dec.DisallowUnknownFields()
	var token struct{ Token string }
	_ = json.Unmarshal(data, &token)
	if e != nil || r.Header.Get("Content-Type") != "application/json" || dec.Decode(in) != nil || token.Token == "" || len(token.Token) > 128 {
		apiError(w, 400, "INVALID_REQUEST", "Send the share token.")
		return false
	}
	return true
}
