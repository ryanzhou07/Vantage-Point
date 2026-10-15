# Receipt draft contracts — VP-008

Owner UUID is verified by Foundation; callers never supply owner_id. Receipt guests are out of scope. `/dashboard/receipts` consumes the existing authenticated shell. No Settings/nav/callback edits.

## HTTP

Bearer-only, exact WEB_ORIGIN, no credentials/cookies, private `Cache-Control: no-store`. GET `/api/v1/me` behavior stays unchanged. Receipt routes opt into GET/POST and Authorization/Content-Type preflight using Foundation.ProtectMethods; verification/prefilter/canonical identity remain unchanged. Existing three-second request-context budget and 16 validation slots remain; slow uploads may fail safely and can be resumed using the same image. Limits are deliberately conservative for this initial slice.

- `POST /api/v1/receipts`: raw PNG/JPEG bytes, <=5 MiB, decoded content <=16 megapixels and <=8000 per side. Returns 202 Receipt. Exact-byte SHA256 deduplicates **per owner**. A duplicate returns the existing record/job, never enqueues another extraction. Failed upload leaves a resumable DRAFT reservation.
- `GET /api/v1/receipts`: `{receipts:[...],limit:20}` latest 20 by created_at/id. This is a bounded recent-drafts selector, not complete receipt history pagination.
- `GET /api/v1/receipts/{uuid}`: Receipt or 404 for unavailable/other-owner ID.
- `GET /api/v1/receipts/{uuid}/image`: authenticated image bytes, nosniff/no-store, no public/signed URL. Browser may fetch its own private image into a temporary object URL; URL revoked on unmount/replacement.
- `POST /api/v1/receipts/{uuid}/review`: `{version, draft, calculate}`. Draft fields below. `calculate:false` saves incomplete review and clears any prior preview. `calculate:true` validates and atomically saves a server-calculated preview. 409 on stale version; 422 for incomplete/unsupported/irreconciled input. A failed save does not erase client edits.
- `POST /api/v1/receipts/{uuid}/retry`: `{version}`. PROCESSING_FAILED only, max two owner retries, 409 otherwise. Reuses the durable job; no polling-triggered retry.

Error shape follows DESIGN §27, with safe code/message/request_id. 401 invalid auth, 403 origin, 404 receipt, 409 version/state, 413 size, 415 content type, 422 image/review, 429 limits, 503 setup/upstream. No credential/provider body is echoed.

## Types and financial rules

Receipt: id, owner_id, image reference/hash/MIME, status (DRAFT, PROCESSING, NEEDS_REVIEW, PROCESSING_FAILED), positive version, immutable proposal, editable draft, optional preview, safe error_code, retries, timestamps. No FINALIZED status exists in this slice.

Proposal: merchant/date/currency, items[{id,name,quantity,amount_minor,participant_id}], subtotal_minor, tax_minor, tip_minor, discount_minor,total_minor. All amounts are integer minor units (USD initially); amount_minor means **whole line**, not unit price. Quantity is descriptive integer1..1000. Strings bounded; max200 items/40 participants. Each amount and aggregate item subtotal <=100,000,000 cents. Unsupported currency cannot be verified. AI proposal does not establish money truth.

Draft adds participants[{id,name,included}], tax_method and tip_method. IDs remain stable across saves. Owner explicitly chooses each method: equal/proportional; neither defaults. Every line has one included participant. Excluded participants must have no assigned items. Equal weights all explicitly included participants, even those with zero item subtotal. Proportional weights their item subtotal; zero total weight is an error, even for zero tax/tip. All-zero receipts may use equal with included participants.

Nonzero receipt-level discounts block calculation. The owner must review the original extraction/image and explicitly resolve net line amounts/subtotal before setting receipt-level discount to zero. No automatic discount distribution, hidden balancing amount or guessed policy. Item sum==subtotal and subtotal+tax+tip==total exactly before preview.

Largest-remainder distribution conserves each tax/tip cent. Ties sort lexicographically by stable participant ID. Go int64 and bounds keep products below overflow. Preview has shares[{participant_id,items_minor,tax_minor,tip_minor,total_minor}], total_minor,draft:true. Browser formats only; Go calculates. Any subsequent save/edit clears the old verified preview. Result is a saved **draft preview**, never participant-confirmed, finalized or payable.

## Persistence and security

Append-only Receipt migration. public.receipts references profiles.id; JSONB snapshots preserve proposal and owner draft without prematurely introducing guest/allocation/receivable tables. Draft snapshots contain the initial whole-line assignments. Transactional version CAS protects the entire item/participant allocation snapshot, preventing conflicting assignment overwrites. Domain financial validation occurs in the API; browser writes/RPC execution are denied. Authenticated SELECT uses owner RLS. The Go API uses server-only service-role credentials for private Storage and service-only RPCs, always passing the UUID from verified context. This is a privileged boundary requiring independent review. Neither service-role nor Gemini key reaches NEXT_PUBLIC variables/browser/Git/logs.

Private bucket `receipt-images`, owner-only SELECT policy joins to owner-visible receipts. No anonymous/upload/update policies; API validates actual image content before server-side upload. Fixed stored paths use verified UUID/generated receipt UUID; no user-controlled remote URL fetch. Auth deletion cascades receipt/jobs, but Storage object retention/deletion is not implemented by this task; production account-deletion/retention must arrange object cleanup separately.

Durable receipt_vision_jobs are private. Transactional enqueue, primary receipt_id deduplication, SKIP LOCKED claim, 90s fenced leases, global max2 active leases, worker process concurrency1. Crash recovery can repeat an in-flight paid call after lease expiry (providers do not promise exactly-once); stale lease cannot overwrite results. Max3 automatic attempts per stage, jittered exponential backoff, max2 owner retries / max9 total attempts. Permanent access/schema errors stop immediately. Max20 new receipts/owner/day and5 active uploads/processing reservations, serialized per-owner in DB. Reads/mutations additionally60 requests/min/owner per API instance, bounded4096-entry window map; max2 decoded uploads per process. No shared cache or financial authority in polling.

## Provider contract / AWS portability

Extractor.Extract(ctx,imageBytes,mime)->validated Proposal or normalized ProviderError{Code,Retryable,HTTPStatus,Reason,Fields} (only whitelisted safe diagnostic values). No provider types cross the adapter. Worker supplies a60s budget; Gemini request40s, storage10s, output256KiB. Fixed Google endpoint/model `gemini-3.5-flash-lite`, typed responseSchema + responseMimeType application/json, key header (never URL), no redirects/tools/function execution. Image text/output are untrusted and bounded. Access denied maps VISION_ACCESS_REQUIRED; model-not-found maps VISION_MODEL_UNAVAILABLE, never silent fallback. Explicit fake provider requires two opt-ins and is for synthetic dev/tests only.

An AWS Bedrock multimodal or Textract adapter can implement Extractor and map its output/errors into the same Proposal contract, without changing database/UI/calculations. This task does not implement AWS runtime resources, IAM, adapters or paid fallback.
