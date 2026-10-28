package receipts

import (
	"bytes"
	"encoding/json"
	"io"
	"net"
	"net/http"
)

// Join is the public, token-gated participant route. Authorization is the share token
// itself (receipt-scoped, expires with the session); participants are never canonical users.
// Posting again is the participant's authoritative refetch. Tokens are never logged.
func (a *API) Join(w http.ResponseWriter, r *http.Request) {
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
		return
	}
	if r.Method != "POST" {
		apiError(w, 405, "METHOD_NOT_ALLOWED", "Use POST.")
		return
	}
	ip, _, _ := net.SplitHostPort(r.RemoteAddr)
	if !a.allowKey("join:"+ip, 30) {
		h.Set("Retry-After", "60")
		apiError(w, 429, "RATE_LIMITED", "Too many attempts. Wait a minute.")
		return
	}
	if a.Store == nil {
		apiError(w, 503, "RECEIPT_SETUP_REQUIRED", "Receipt service is not configured yet.")
		return
	}
	data, e := io.ReadAll(http.MaxBytesReader(w, r.Body, 1024))
	var in struct {
		Token         string `json:"token"`
		ParticipantID string `json:"participant_id"`
	}
	dec := json.NewDecoder(bytes.NewReader(data))
	dec.DisallowUnknownFields()
	if e != nil || r.Header.Get("Content-Type") != "application/json" || dec.Decode(&in) != nil || in.Token == "" || len(in.Token) > 128 || len(in.ParticipantID) > 64 {
		apiError(w, 400, "INVALID_REQUEST", "Send the share token.")
		return
	}
	now := a.Now()
	s, e := a.Store.CheckSession(r.Context(), in.Token, now)
	if e != nil {
		storeError(w, e)
		return
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
	out := map[string]any{"state": "ACTIVE", "server_time": stamp(now), "session": map[string]string{"id": s.ID, "expires_at": s.ExpiresAt}}
	people := []person{}
	merchant, currency := "", ""
	if rec.Draft != nil {
		merchant, currency = rec.Draft.Merchant, rec.Draft.Currency
		for _, p := range rec.Draft.Participants {
			if p.Included && p.Name != "" {
				people = append(people, person{p.ID, p.Name})
				if p.ID == in.ParticipantID {
					out["participant"] = person{p.ID, p.Name}
				}
			}
		}
	}
	if in.ParticipantID != "" && out["participant"] == nil {
		apiError(w, 404, "PARTICIPANT_NOT_FOUND", "That name is not on this receipt.")
		return
	}
	out["receipt"], out["participants"] = map[string]string{"merchant": merchant, "currency": currency}, people
	jsonReply(w, 200, out)
}
