// Package foundation owns canonical authenticated owner identity.
package foundation

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/url"
	"regexp"
	"strings"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

type identityKey struct{}

// UserID returns identity only after successful server-side verification.
func UserID(ctx context.Context) (string, bool) {
	id, ok := ctx.Value(identityKey{}).(string)
	return id, ok
}

var uuid = regexp.MustCompile(`^[0-9a-fA-F]{8}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{12}$`)

type API struct {
	base, key, origin string
	client            *http.Client
	slots             chan struct{}
}

func New(base, key, origin string) (*API, error) {
	u, e := url.Parse(base)
	if e != nil || u.Host == "" || u.RawQuery != "" || u.Fragment != "" || u.User != nil || (u.Scheme != "https" && !(u.Scheme == "http" && (u.Hostname() == "localhost" || u.Hostname() == "127.0.0.1"))) || key == "" {
		return nil, errors.New("invalid Supabase configuration")
	}
	o, e := url.Parse(origin)
	if e != nil || o.Host == "" || o.Path != "" || o.RawQuery != "" || o.Fragment != "" || o.User != nil || (o.Scheme != "https" && !(o.Scheme == "http" && (o.Hostname() == "localhost" || o.Hostname() == "127.0.0.1"))) {
		return nil, errors.New("invalid browser origin")
	}
	return &API{strings.TrimRight(base, "/"), key, origin, &http.Client{Timeout: 2 * time.Second, CheckRedirect: func(_ *http.Request, _ []*http.Request) error { return http.ErrUseLastResponse }}, make(chan struct{}, 16)}, nil
}
func (a *API) call(ctx context.Context, path, token string, out any) (int, error) {
	req, e := http.NewRequestWithContext(ctx, "GET", a.base+path, nil)
	if e != nil {
		return 0, e
	}
	req.Header.Set("apikey", a.key)
	req.Header.Set("Authorization", "Bearer "+token)
	resp, e := a.client.Do(req)
	if e != nil {
		return 0, e
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		return resp.StatusCode, errors.New("upstream rejected request")
	}
	data, e := io.ReadAll(io.LimitReader(resp.Body, 65537))
	if e != nil || len(data) > 65536 {
		return 0, errors.New("upstream response too large")
	}
	return resp.StatusCode, json.Unmarshal(data, out)
}
func (a *API) verify(ctx context.Context, token string) (string, int) {
	// Unverified parsing is only a rejection prefilter. /user must verify signature
	// and return exactly the same subject before any identity enters context.
	claims := jwt.RegisteredClaims{}
	_, _, e := jwt.NewParser().ParseUnverified(token, &claims)
	if e != nil || !uuid.MatchString(claims.Subject) || claims.Issuer != a.base+"/auth/v1" || claims.ExpiresAt == nil || len(claims.Audience) != 1 || claims.Audience[0] != "authenticated" || jwt.NewValidator(jwt.WithExpirationRequired(), jwt.WithIssuer(a.base+"/auth/v1"), jwt.WithAudience("authenticated")).Validate(&claims) != nil {
		return "", 401
	}
	var user struct {
		ID          string `json:"id"`
		IsAnonymous bool   `json:"is_anonymous"`
	}
	status, e := a.call(ctx, "/auth/v1/user", token, &user)
	if e != nil {
		if status == 401 || status == 403 {
			return "", 401
		}
		return "", 503
	}
	if user.ID != claims.Subject || user.IsAnonymous {
		return "", 401
	}
	return user.ID, 200
}
func fail(w http.ResponseWriter, status int, code, message string, id string) {
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(map[string]any{"error": map[string]string{"code": code, "message": message, "request_id": id}})
}
func (a *API) Protect(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		b := make([]byte, 16)
		_, _ = rand.Read(b)
		id := hex.EncodeToString(b)
		w.Header().Set("Content-Type", "application/json")
		w.Header().Set("Cache-Control", "no-store")
		w.Header().Set("Vary", "Origin")
		w.Header().Set("X-Request-ID", id)
		if origin := r.Header.Get("Origin"); origin != "" {
			if origin != a.origin {
				fail(w, 403, "FORBIDDEN", "Origin is not allowed.", id)
				return
			}
			w.Header().Set("Access-Control-Allow-Origin", a.origin)
		}
		if r.Method == "OPTIONS" {
			w.Header().Set("Access-Control-Allow-Methods", "GET")
			w.Header().Set("Access-Control-Allow-Headers", "Authorization")
			w.WriteHeader(204)
			return
		}
		if r.Method != "GET" {
			fail(w, 405, "METHOD_NOT_ALLOWED", "Use GET.", id)
			return
		}
		if r.URL.RawQuery != "" || r.Header.Get("X-User-ID") != "" {
			fail(w, 400, "INVALID_REQUEST", "User identifiers are not accepted.", id)
			return
		}
		auth := r.Header.Get("Authorization")
		if len(r.Header.Values("Authorization")) != 1 || !strings.HasPrefix(auth, "Bearer ") || len(auth) > 8192 || len(auth) < 8 || strings.ContainsAny(auth[7:], " \t\n\r") {
			fail(w, 401, "UNAUTHENTICATED", "Sign in to continue.", id)
			return
		}
		select {
		case a.slots <- struct{}{}:
			defer func() { <-a.slots }()
		default:
			w.Header().Set("Retry-After", "1")
			fail(w, 429, "RATE_LIMITED", "Try again shortly.", id)
			return
		}
		ctx, cancel := context.WithTimeout(r.Context(), 3*time.Second)
		defer cancel()
		uid, status := a.verify(ctx, auth[7:])
		if status != 200 {
			code, msg := "UNAUTHENTICATED", "Sign in to continue."
			if status == 503 {
				code, msg = "AUTH_UNAVAILABLE", "Authentication is temporarily unavailable."
			}
			fail(w, status, code, msg, id)
			return
		}
		next.ServeHTTP(w, r.WithContext(context.WithValue(ctx, identityKey{}, uid)))
	})
}

type Profile struct {
	ID          string    `json:"id"`
	DisplayName string    `json:"display_name"`
	Email       string    `json:"email"`
	AvatarURL   string    `json:"avatar_url"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

func (a *API) Me(w http.ResponseWriter, r *http.Request) {
	uid, ok := UserID(r.Context())
	if !ok {
		fail(w, 401, "UNAUTHENTICATED", "Sign in to continue.", w.Header().Get("X-Request-ID"))
		return
	}
	var rows []Profile
	_, e := a.call(r.Context(), "/rest/v1/profiles?select=id,display_name,email,avatar_url,created_at,updated_at&id=eq."+uid, r.Header.Get("Authorization")[7:], &rows)
	if e != nil || len(rows) != 1 || rows[0].ID != uid {
		fail(w, 503, "PROFILE_UNAVAILABLE", "Profile is temporarily unavailable.", w.Header().Get("X-Request-ID"))
		return
	}
	_ = json.NewEncoder(w).Encode(map[string]any{"profile": rows[0]})
}
