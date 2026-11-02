package banking

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
)

// Connection is the only shape ever returned to clients; it has no credential fields.
type Connection struct {
	ID                   string  `json:"id"`
	Provider             string  `json:"provider"`
	InstitutionName      string  `json:"institution_name"`
	Status               string  `json:"status"`
	LastSyncAttemptAt    *string `json:"last_sync_attempt_at"`
	LastSuccessfulSyncAt *string `json:"last_successful_sync_at"`
	CreatedAt            string  `json:"created_at"`
}

type EnrollInput struct {
	UserID, ID, Provider, ProviderConnectionID, InstitutionName string
	Blob                                                        []byte
	KeyVersion                                                  byte
}

// ConnectionStore is the persistence boundary; every method is scoped by the verified user id.
type ConnectionStore interface {
	// Enroll is idempotent on (user, provider, provider connection); created=false returns the existing row.
	Enroll(ctx context.Context, in EnrollInput) (c Connection, created bool, err error)
	List(ctx context.Context, user string) ([]Connection, error)
	Get(ctx context.Context, user, id string) (Connection, bool, error)
	// Credential returns the encrypted blob and key version, or ok=false if not the user's connection.
	Credential(ctx context.Context, user, id string) (blob []byte, ok bool, err error)
	Delete(ctx context.Context, user, id string) (bool, error)
}

// SupabaseStore calls service-role RPCs over PostgREST. The service key never leaves the server.
type SupabaseStore struct {
	base, key string
	client    *http.Client
}

func NewSupabaseStore(base, key string) (*SupabaseStore, error) {
	u, e := url.Parse(base)
	if e != nil || u.Host == "" || u.User != nil || u.RawQuery != "" || u.Fragment != "" || (u.Path != "" && u.Path != "/") || (u.Scheme != "https" && !(u.Scheme == "http" && (u.Hostname() == "localhost" || u.Hostname() == "127.0.0.1"))) || key == "" {
		return nil, errors.New("banking store requires SUPABASE_URL and SUPABASE_SERVICE_ROLE_KEY")
	}
	return &SupabaseStore{strings.TrimRight(base, "/"), key, &http.Client{Timeout: 10 * time.Second, CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }}}, nil
}

var errStore = errors.New("banking: store unavailable")

func (s *SupabaseStore) rpc(ctx context.Context, name string, in, out any) error {
	b, _ := json.Marshal(in)
	req, err := http.NewRequestWithContext(ctx, "POST", s.base+"/rest/v1/rpc/"+name, bytes.NewReader(b))
	if err != nil {
		return errStore
	}
	req.Header.Set("apikey", s.key)
	req.Header.Set("Authorization", "Bearer "+s.key)
	req.Header.Set("Content-Type", "application/json")
	resp, err := s.client.Do(req)
	if err != nil {
		return errStore // never wrap: transport errors can echo URLs
	}
	defer resp.Body.Close()
	data, err := io.ReadAll(io.LimitReader(resp.Body, 262145))
	if err != nil || resp.StatusCode != 200 || len(data) > 262144 || json.Unmarshal(data, out) != nil {
		return errStore
	}
	return nil
}

func (s *SupabaseStore) Enroll(ctx context.Context, in EnrollInput) (Connection, bool, error) {
	var out struct {
		Created    bool       `json:"created"`
		Connection Connection `json:"connection"`
	}
	err := s.rpc(ctx, "banking_enroll", map[string]any{"p_user": in.UserID, "p_id": in.ID, "p_provider": in.Provider, "p_pconn": in.ProviderConnectionID, "p_name": in.InstitutionName, "p_cred": base64.StdEncoding.EncodeToString(in.Blob), "p_ver": in.KeyVersion}, &out)
	return out.Connection, out.Created, err
}

func (s *SupabaseStore) List(ctx context.Context, user string) ([]Connection, error) {
	var out []Connection
	err := s.rpc(ctx, "banking_connections", map[string]any{"p_user": user}, &out)
	return out, err
}

func (s *SupabaseStore) Get(ctx context.Context, user, id string) (Connection, bool, error) {
	var out []Connection
	if err := s.rpc(ctx, "banking_connections", map[string]any{"p_user": user, "p_id": id}, &out); err != nil || len(out) != 1 {
		return Connection{}, false, err
	}
	return out[0], true, nil
}

func (s *SupabaseStore) Credential(ctx context.Context, user, id string) ([]byte, bool, error) {
	var out *struct {
		Credential string `json:"credential"`
	}
	if err := s.rpc(ctx, "banking_credential", map[string]any{"p_user": user, "p_id": id}, &out); err != nil || out == nil {
		return nil, false, err
	}
	b, err := base64.StdEncoding.DecodeString(out.Credential)
	return b, err == nil, nil
}

func (s *SupabaseStore) Delete(ctx context.Context, user, id string) (bool, error) {
	var ok bool
	err := s.rpc(ctx, "banking_disconnect", map[string]any{"p_user": user, "p_id": id}, &ok)
	return ok, err
}
