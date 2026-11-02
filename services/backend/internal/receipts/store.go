package receipts

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
)

type Record struct {
	ID        string    `json:"id"`
	OwnerID   string    `json:"owner_id"`
	ImagePath string    `json:"image_path"`
	ImageHash string    `json:"image_hash"`
	ImageMIME string    `json:"image_mime"`
	Status    string    `json:"status"`
	Version   int64     `json:"version"`
	Proposal  *Proposal `json:"proposal"`
	Draft     *Draft    `json:"draft"`
	Preview   *Preview  `json:"preview"`
	ErrorCode string    `json:"error_code"`
	Retries   int       `json:"retries"`
	CreatedAt string    `json:"created_at"`
}
type Store struct {
	base, key string
	client    *http.Client
}

func NewStore(base, key string) (*Store, error) {
	u, e := url.Parse(base)
	if e != nil || u.Host == "" || u.User != nil || u.RawQuery != "" || u.Fragment != "" || (u.Path != "" && u.Path != "/") || (u.Scheme != "https" && !(u.Scheme == "http" && (u.Hostname() == "localhost" || u.Hostname() == "127.0.0.1"))) || key == "" {
		return nil, errors.New("Receipt server requires SUPABASE_URL and SUPABASE_SERVICE_ROLE_KEY")
	}
	return &Store{strings.TrimRight(base, "/"), key, &http.Client{Timeout: 10 * time.Second, CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }}}, nil
}
func (s *Store) request(ctx context.Context, method, path, mime string, data []byte, max int64) ([]byte, error) {
	req, e := http.NewRequestWithContext(ctx, method, s.base+path, bytes.NewReader(data))
	if e != nil {
		return nil, e
	}
	req.Header.Set("apikey", s.key)
	req.Header.Set("Authorization", "Bearer "+s.key)
	if mime != "" {
		req.Header.Set("Content-Type", mime)
	}
	if method == "POST" && strings.HasPrefix(path, "/storage/") {
		req.Header.Set("x-upsert", "true")
	}
	resp, e := s.client.Do(req)
	if e != nil {
		return nil, errors.New("RECEIPT_UNAVAILABLE")
	}
	defer resp.Body.Close()
	b, e := io.ReadAll(io.LimitReader(resp.Body, max+1))
	if e != nil || int64(len(b)) > max {
		return nil, errors.New("RECEIPT_UNAVAILABLE")
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		var failure struct {
			Message string `json:"message"`
		}
		_ = json.Unmarshal(b, &failure)
		switch failure.Message {
		case "RECEIPT_QUOTA", "RECEIPT_NOT_FOUND", "RECEIPT_CONFLICT", "RECEIPT_NOT_REVIEWABLE", "RECEIPT_RETRY_LIMIT", "RECEIPT_INVALID_REVIEW", "RECEIPT_INVALID_ALLOCATION", "SESSION_NOT_FOUND", "SESSION_INVALID", "SPLIT_NOT_READY", "SPLIT_INVALID_NAME", "SPLIT_FULL", "SPLIT_PARTICIPANT_NOT_FOUND", "SPLIT_ITEM_NOT_FOUND", "SPLIT_CLAIM_CONFLICT":
			return nil, errors.New(failure.Message)
		}
		return nil, errors.New("RECEIPT_UNAVAILABLE")
	}
	return b, nil
}
func (s *Store) rpc(ctx context.Context, name string, in, out any) error {
	b, e := json.Marshal(in)
	if e != nil {
		return e
	}
	limit := int64(262144)
	if name == "receipt_documents" {
		limit = 5242880
	}
	data, e := s.request(ctx, "POST", "/rest/v1/rpc/"+name, "application/json", b, limit)
	if e != nil {
		return e
	}
	return json.Unmarshal(data, out)
}
func (s *Store) get(ctx context.Context, owner, id string) (Record, error) {
	var record Record
	err := s.rpc(ctx, "receipt_document", map[string]string{"p_owner": owner, "p_id": id}, &record)
	if err != nil {
		return Record{}, err
	}
	if record.ID != id {
		return Record{}, errors.New("RECEIPT_NOT_FOUND")
	}
	return record, nil
}
func (s *Store) list(ctx context.Context, owner string) ([]Record, error) {
	var rows []Record
	err := s.rpc(ctx, "receipt_documents", map[string]string{"p_owner": owner}, &rows)
	return rows, err
}
func (s *Store) image(ctx context.Context, r Record) ([]byte, error) {
	return s.request(ctx, "GET", "/storage/v1/object/authenticated/receipt-images/"+r.ImagePath, "", nil, MaxImageBytes)
}
