package receipts

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"net/http"
	"time"
)

const (
	DefaultSessionTTL = 5 * time.Minute
	MinSessionTTL     = time.Minute
	MaxSessionTTL     = time.Hour
)

// ParseSessionTTL reads SPLIT_SESSION_TTL (Go duration); empty means the 5-minute default.
func ParseSessionTTL(v string) (time.Duration, error) {
	if v == "" {
		return DefaultSessionTTL, nil
	}
	d, e := time.ParseDuration(v)
	if e != nil || d < MinSessionTTL || d > MaxSessionTTL {
		return 0, errors.New("SPLIT_SESSION_TTL must be between 1m and 60m")
	}
	return d, nil
}

// Session is the owner-visible session record. The raw token is never part of it.
type Session struct {
	ID            string  `json:"id"`
	ReceiptID     string  `json:"receipt_id"`
	Status        string  `json:"status"`
	CreatedAt     string  `json:"created_at"`
	ExpiresAt     string  `json:"expires_at"`
	InvalidatedAt *string `json:"invalidated_at"`
	TokenHash     string  `json:"token_hash,omitempty"`
	PaymentMethod *string `json:"payment_method,omitempty"` // only set by CheckSessionFor
	OwnerID       string  `json:"owner_id,omitempty"`       // server-side only; cleared before any response
}

func newToken() (token, hash string, e error) {
	b := make([]byte, 32)
	if _, e = rand.Read(b); e != nil {
		return
	}
	token = base64.RawURLEncoding.EncodeToString(b)
	return token, hashToken(token), nil
}
func hashToken(t string) string {
	s := sha256.Sum256([]byte(t))
	return hex.EncodeToString(s[:])
}
func stamp(t time.Time) string { return t.UTC().Format(time.RFC3339Nano) }

func (s *Store) sessionOp(ctx context.Context, name string, args map[string]any) (Session, error) {
	var out Session
	e := s.rpc(ctx, name, args, &out)
	return out, e
}

// CheckSession is the participant-token gate (used by the join route). The backend
// clock decides expiry; the stored hash is re-compared in constant time.
func (s *Store) CheckSession(ctx context.Context, token string, now time.Time) (Session, error) {
	return s.checkSession(ctx, token, now, "split_session_check", nil)
}

// CheckSessionFor is CheckSession plus that participant's payment method, in the same single call.
func (s *Store) CheckSessionFor(ctx context.Context, token string, now time.Time, participant string) (Session, error) {
	extra := map[string]any{"p_participant": nil}
	if receiptID.MatchString(participant) {
		extra["p_participant"] = participant
	}
	return s.checkSession(ctx, token, now, "split_session_check_for", extra)
}

func (s *Store) checkSession(ctx context.Context, token string, now time.Time, rpc string, extra map[string]any) (Session, error) {
	h := hashToken(token)
	args := map[string]any{"p_hash": h, "p_now": stamp(now)}
	for k, v := range extra {
		args[k] = v
	}
	out, e := s.sessionOp(ctx, rpc, args)
	if e != nil {
		return Session{}, e
	}
	if subtle.ConstantTimeCompare([]byte(out.TokenHash), []byte(h)) != 1 {
		return Session{}, errors.New("SESSION_INVALID")
	}
	out.TokenHash = ""
	return out, nil
}

// broadcast sends a payload-free Realtime notification; clients refetch authoritative state.
// Best effort and tightly bounded: a Realtime outage never fails the session operation.
func (s *Store) broadcast(ctx context.Context, sessionID string) {
	ctx, cancel := context.WithTimeout(ctx, 2*time.Second)
	defer cancel()
	b, e := json.Marshal(map[string]any{"messages": []map[string]any{{"topic": "split:" + sessionID, "event": "session", "payload": map[string]any{}, "private": false}}})
	if e == nil {
		_, _ = s.request(ctx, "POST", "/realtime/v1/api/broadcast", "application/json", b, 4096)
	}
}

// sessionView adds the backend-decided state: an ACTIVE session past expires_at reads EXPIRED.
func (a *API) sessionView(s Session) map[string]any {
	now := a.Now()
	state := s.Status
	if exp, e := time.Parse(time.RFC3339Nano, s.ExpiresAt); s.Status == "ACTIVE" && (e != nil || !now.Before(exp)) {
		state = "EXPIRED"
	}
	return map[string]any{"session": s, "state": state, "server_time": stamp(now)}
}

// sessionRoute handles /receipts/{id}/session[/close|/extend]; the receipt owner check already ran.
func (a *API) sessionRoute(w http.ResponseWriter, r *http.Request, owner, id string, sub []string) {
	ctx, now := r.Context(), a.Now()
	args := map[string]any{"p_owner": owner, "p_receipt": id}
	switch {
	case len(sub) == 0 && r.Method == "GET":
		s, e := a.Store.sessionOp(ctx, "split_session_status", args)
		a.sessionReply(w, 200, s, e)
	case len(sub) == 0 && r.Method == "POST": // create, also the restart path (new token, new expiry)
		if !a.allowKey("session:"+owner, 5) {
			w.Header().Set("Retry-After", "60")
			apiError(w, 429, "RATE_LIMITED", "Too many session requests. Wait a minute.")
			return
		}
		token, hash, e := newToken()
		if e != nil {
			apiError(w, 503, "RECEIPT_UNAVAILABLE", "Receipt service is unavailable. Try again shortly.")
			return
		}
		args["p_hash"], args["p_now"], args["p_ttl_seconds"] = hash, stamp(now), int(a.TTL.Seconds())
		s, e := a.Store.sessionOp(ctx, "split_session_open", args)
		if e != nil {
			storeError(w, e)
			return
		}
		a.Store.broadcast(ctx, s.ID)
		v := a.sessionView(s)
		v["token"] = token // returned exactly once
		jsonReply(w, 201, v)
	case len(sub) == 1 && sub[0] == "close" && r.Method == "POST":
		args["p_now"] = stamp(now)
		s, e := a.Store.sessionOp(ctx, "split_session_close", args)
		if e == nil {
			a.Store.broadcast(ctx, s.ID)
		}
		a.sessionReply(w, 200, s, e)
	case len(sub) == 1 && sub[0] == "extend" && r.Method == "POST":
		args["p_now"], args["p_ttl_seconds"] = stamp(now), int(a.TTL.Seconds())
		s, e := a.Store.sessionOp(ctx, "split_session_extend", args)
		if e == nil {
			a.Store.broadcast(ctx, s.ID)
		}
		a.sessionReply(w, 200, s, e)
	default:
		apiError(w, 405, "METHOD_NOT_ALLOWED", "Use the documented session operation.")
	}
}
func (a *API) sessionReply(w http.ResponseWriter, status int, s Session, e error) {
	if e != nil {
		storeError(w, e)
		return
	}
	jsonReply(w, status, a.sessionView(s))
}
