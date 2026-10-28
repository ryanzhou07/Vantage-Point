package receipts

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"image"
	_ "image/jpeg"
	_ "image/png"
	"io"
	"net/http"
	"regexp"
	"strings"
	"sync"
	"time"

	"github.com/ryanzhou07/Vantage-Point/services/backend/internal/foundation"
)

const MaxImageBytes int64 = 5 * 1024 * 1024

var receiptID = regexp.MustCompile(`^[a-f0-9]{8}-[a-f0-9]{4}-[a-f0-9]{4}-[a-f0-9]{4}-[a-f0-9]{12}$`)

type rateEntry struct {
	start time.Time
	count int
}
type API struct {
	Store *Store
	TTL   time.Duration
	Now   func() time.Time // injected in tests; backend clock decides session expiry
	slots chan struct{}
	mu    sync.Mutex
	rates map[string]rateEntry
}

func NewAPI(s *Store) *API {
	return &API{Store: s, TTL: DefaultSessionTTL, Now: time.Now, slots: make(chan struct{}, 2), rates: map[string]rateEntry{}}
}
func (a *API) allow(id string) bool { return a.allowKey(id, 60) }
func (a *API) allowKey(id string, limit int) bool {
	a.mu.Lock()
	defer a.mu.Unlock()
	now := time.Now()
	e := a.rates[id]
	if now.Sub(e.start) >= time.Minute {
		e = rateEntry{start: now}
	}
	if e.count >= limit {
		return false
	}
	if len(a.rates) >= 4096 {
		for k, v := range a.rates {
			if now.Sub(v.start) >= time.Minute {
				delete(a.rates, k)
			}
		}
		if _, ok := a.rates[id]; !ok && len(a.rates) >= 4096 {
			return false
		}
	}
	e.count++
	a.rates[id] = e
	return true
}
func apiError(w http.ResponseWriter, status int, code, message string) {
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(map[string]any{"error": map[string]string{"code": code, "message": message, "request_id": w.Header().Get("X-Request-ID")}})
}
func storeError(w http.ResponseWriter, e error) {
	status := 503
	msg := "Receipt service is unavailable. Try again shortly."
	switch e.Error() {
	case "RECEIPT_QUOTA":
		status = 429
		msg = "Receipt upload limit reached. Try again later."
	case "RECEIPT_NOT_FOUND":
		status = 404
		msg = "Receipt was not found."
	case "SESSION_NOT_FOUND":
		status = 404
		msg = "No active split session."
	case "SESSION_INVALID":
		status = 401
		msg = "This split link is invalid or has expired."
	case "RECEIPT_CONFLICT":
		status = 409
		msg = "This receipt changed. Reload before saving again."
	case "RECEIPT_INVALID_REVIEW", "RECEIPT_INVALID_ALLOCATION":
		status = 422
		msg = "Check item assignments and receipt amounts before saving."
	case "RECEIPT_NOT_REVIEWABLE", "RECEIPT_RETRY_LIMIT":
		status = 409
		msg = "This operation is unavailable for the current receipt state."
	}
	apiError(w, status, e.Error(), msg)
}
func jsonReply(w http.ResponseWriter, status int, v any) {
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}
func (a *API) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	owner, ok := foundation.UserID(r.Context())
	if !ok {
		apiError(w, 401, "UNAUTHENTICATED", "Sign in to continue.")
		return
	}
	a.serve(w, r, owner)
}

// serve takes the owner from verified context (tests call it directly).
func (a *API) serve(w http.ResponseWriter, r *http.Request, owner string) {
	if !a.allow(owner) {
		w.Header().Set("Retry-After", "60")
		apiError(w, 429, "RATE_LIMITED", "Wait a minute before refreshing again.")
		return
	}
	if a.Store == nil {
		apiError(w, 503, "RECEIPT_SETUP_REQUIRED", "Receipt service is not configured yet.")
		return
	}
	path := strings.TrimPrefix(r.URL.Path, "/api/v1/receipts")
	if path == "" {
		switch r.Method {
		case "POST":
			a.upload(w, r, owner)
		case "GET":
			rows, e := a.Store.list(r.Context(), owner)
			if e != nil {
				storeError(w, e)
				return
			}
			jsonReply(w, 200, map[string]any{"receipts": rows, "limit": 20})
		default:
			apiError(w, 405, "METHOD_NOT_ALLOWED", "Use GET or POST.")
		}
		return
	}
	parts := strings.Split(strings.TrimPrefix(path, "/"), "/")
	if len(parts) > 4 || !receiptID.MatchString(parts[0]) {
		apiError(w, 404, "NOT_FOUND", "Receipt route not found.")
		return
	}
	id := parts[0]
	record, e := a.Store.get(r.Context(), owner, id)
	if e != nil {
		storeError(w, e)
		return
	}
	if len(parts) >= 2 && parts[1] == "session" {
		a.sessionRoute(w, r, owner, id, parts[2:])
		return
	}
	if len(parts) > 2 {
		apiError(w, 404, "NOT_FOUND", "Receipt route not found.")
		return
	}
	if len(parts) == 1 && r.Method == "GET" {
		jsonReply(w, 200, record)
		return
	}
	if len(parts) == 2 && parts[1] == "image" && r.Method == "GET" {
		data, e := a.Store.image(r.Context(), record)
		if e != nil {
			storeError(w, e)
			return
		}
		w.Header().Set("Content-Type", record.ImageMIME)
		w.Header().Set("Content-Disposition", "inline; filename=receipt")
		w.Header().Set("X-Content-Type-Options", "nosniff")
		w.WriteHeader(200)
		_, _ = w.Write(data)
		return
	}
	if r.Method != "POST" || len(parts) != 2 {
		apiError(w, 405, "METHOD_NOT_ALLOWED", "Use the documented receipt operation.")
		return
	}
	switch parts[1] {
	case "review":
		a.save(w, r, owner, id)
	case "retry":
		var in struct {
			Version int64 `json:"version"`
		}
		if !readJSON(w, r, &in) {
			return
		}
		var result Record
		e = a.Store.rpc(r.Context(), "receipt_retry", map[string]any{"p_owner": owner, "p_id": id, "p_version": in.Version}, &result)
		if e != nil {
			storeError(w, e)
			return
		}
		jsonReply(w, 202, result)
	default:
		apiError(w, 404, "NOT_FOUND", "Receipt route not found.")
	}
}
func ValidateImage(data []byte) (string, error) {
	if int64(len(data)) > MaxImageBytes || len(data) == 0 {
		return "", fmt.Errorf("Choose a JPEG or PNG image up to 5 MiB.")
	}
	config, format, e := image.DecodeConfig(bytes.NewReader(data))
	if e != nil || (format != "png" && format != "jpeg") || config.Width < 1 || config.Height < 1 || config.Width > 8000 || config.Height > 8000 || int64(config.Width)*int64(config.Height) > 16000000 {
		return "", fmt.Errorf("Choose a valid JPEG or PNG up to 16 megapixels and 8,000 pixels per side.")
	}
	if _, _, e = image.Decode(bytes.NewReader(data)); e != nil {
		return "", fmt.Errorf("Image could not be decoded. Try another photo.")
	}
	return "image/" + format, nil
}
func (a *API) upload(w http.ResponseWriter, r *http.Request, owner string) {
	select {
	case a.slots <- struct{}{}:
		defer func() { <-a.slots }()
	default:
		apiError(w, 429, "UPLOAD_BUSY", "Another image is uploading. Try shortly.")
		return
	}
	data, e := io.ReadAll(http.MaxBytesReader(w, r.Body, MaxImageBytes))
	if e != nil {
		apiError(w, 413, "IMAGE_TOO_LARGE", "Choose an image up to 5 MiB.")
		return
	}
	mime, e := ValidateImage(data)
	if e != nil {
		apiError(w, 422, "INVALID_IMAGE", e.Error())
		return
	}
	sum := sha256.Sum256(data)
	var record Record
	e = a.Store.rpc(r.Context(), "receipt_create", map[string]any{"p_owner": owner, "p_hash": hex.EncodeToString(sum[:]), "p_mime": mime}, &record)
	if e != nil {
		storeError(w, e)
		return
	}
	if record.Status == "DRAFT" {
		_, e = a.Store.request(r.Context(), "POST", "/storage/v1/object/receipt-images/"+record.ImagePath, mime, data, 65536)
		if e != nil {
			storeError(w, e)
			return
		}
		e = a.Store.rpc(r.Context(), "receipt_enqueue", map[string]string{"p_owner": owner, "p_id": record.ID}, &record)
		if e != nil {
			storeError(w, e)
			return
		}
	}
	jsonReply(w, 202, record)
}
func readJSON(w http.ResponseWriter, r *http.Request, out any) bool {
	if r.Header.Get("Content-Type") != "application/json" {
		apiError(w, 415, "INVALID_REQUEST", "Use application/json.")
		return false
	}
	data, err := io.ReadAll(http.MaxBytesReader(w, r.Body, 131072))
	var object map[string]any
	if err != nil || json.Unmarshal(data, &object) != nil || object == nil || hasNull(object) {
		apiError(w, 400, "INVALID_REQUEST", "Send one complete JSON object; null amounts are not accepted.")
		return false
	}
	if _, ok := object["version"]; !ok {
		apiError(w, 400, "INVALID_REQUEST", "Receipt version is required.")
		return false
	}
	if draft, ok := object["draft"]; ok {
		d, ok := draft.(map[string]any)
		if !ok || !hasFields(d, "merchant", "date", "currency", "items", "subtotal_minor", "tax_minor", "tip_minor", "discount_minor", "total_minor", "participants", "tax_method", "tip_method") {
			apiError(w, 400, "INVALID_REQUEST", "Send all receipt review fields explicitly.")
			return false
		}
		if lines, ok := d["items"].([]any); ok {
			for _, line := range lines {
				item, ok := line.(map[string]any)
				if !ok || !hasFields(item, "id", "name", "quantity", "amount_minor", "participant_id") {
					apiError(w, 400, "INVALID_REQUEST", "Send complete item line fields.")
					return false
				}
			}
		}
		if people, ok := d["participants"].([]any); ok {
			for _, person := range people {
				p, ok := person.(map[string]any)
				if !ok || !hasFields(p, "id", "name", "included") {
					apiError(w, 400, "INVALID_REQUEST", "Send complete participant fields.")
					return false
				}
			}
		}
	}
	dec := json.NewDecoder(bytes.NewReader(data))
	dec.DisallowUnknownFields()
	if dec.Decode(out) != nil {
		apiError(w, 400, "INVALID_REQUEST", "Check the receipt fields and request size.")
		return false
	}
	return true
}
func hasFields(object map[string]any, names ...string) bool {
	for _, name := range names {
		if _, ok := object[name]; !ok {
			return false
		}
	}
	return true
}
func hasNull(value any) bool {
	switch v := value.(type) {
	case nil:
		return true
	case map[string]any:
		for _, entry := range v {
			if hasNull(entry) {
				return true
			}
		}
	case []any:
		for _, entry := range v {
			if hasNull(entry) {
				return true
			}
		}
	}
	return false
}
func (a *API) save(w http.ResponseWriter, r *http.Request, owner, id string) {
	var in struct {
		Version   int64 `json:"version"`
		Draft     Draft `json:"draft"`
		Calculate bool  `json:"calculate"`
	}
	if !readJSON(w, r, &in) {
		return
	}
	if e := ValidateProposal(in.Draft.Proposal); e != nil {
		apiError(w, 422, "INVALID_REVIEW", e.Error())
		return
	}
	if len(in.Draft.Participants) > MaxParticipants {
		apiError(w, 422, "INVALID_REVIEW", "Use at most 40 participants.")
		return
	}
	for _, p := range in.Draft.Participants {
		if !bounded(p.ID, 64) || !bounded(p.Name, 100) {
			apiError(w, 422, "INVALID_REVIEW", "Check participant names.")
			return
		}
	}
	if in.Draft.Date != "" {
		if _, e := time.Parse("2006-01-02", in.Draft.Date); e != nil {
			apiError(w, 422, "INVALID_REVIEW", "Use YYYY-MM-DD for the receipt date, or leave it empty.")
			return
		}
	}
	in.Draft = canonicalizeDraft(id, in.Draft)
	if e := ValidateProposal(in.Draft.Proposal); e != nil {
		apiError(w, 422, "INVALID_REVIEW", e.Error())
		return
	}
	participantIDs := map[string]bool{}
	for _, person := range in.Draft.Participants {
		if participantIDs[person.ID] {
			apiError(w, 422, "INVALID_REVIEW", "Participant IDs must be unique.")
			return
		}
		participantIDs[person.ID] = true
	}
	for _, item := range in.Draft.Items {
		if item.ParticipantID != "" && !participantIDs[item.ParticipantID] {
			apiError(w, 422, "INVALID_REVIEW", "Choose an existing participant for every assignment.")
			return
		}
	}
	var preview *Preview
	if in.Calculate {
		p, e := Calculate(in.Draft)
		if e != nil {
			apiError(w, 422, "REVIEW_REQUIRED", e.Error())
			return
		}
		preview = &p
	}
	var result Record
	e := a.Store.rpc(r.Context(), "receipt_save", map[string]any{"p_owner": owner, "p_id": id, "p_version": in.Version, "p_draft": in.Draft, "p_preview": preview}, &result)
	if e != nil {
		storeError(w, e)
		return
	}
	jsonReply(w, 200, result)
}
