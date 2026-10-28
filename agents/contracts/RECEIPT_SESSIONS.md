# Receipt split-session contract — VP-015a

Owner identity comes only from verified Foundation context; receipt IDs never authorize. Non-owner or unknown receipt returns 404. Same headers/errors/limits as RECEIPT_DRAFTS.md.

- `POST /api/v1/receipts/{uuid}/session` -> 201 `{session,state,server_time,token}`. Creates a session (also the restart path); any previous active session is invalidated. `token` (256-bit, base64url) is returned once; only its SHA-256 is stored. Limit 5/min/owner.
- `GET .../session` -> latest session `{session,state,server_time}`; 404 if none.
- `POST .../session/close` -> invalidates the active session; 404 if none.
- `POST .../session/extend` -> active, unexpired session gets `expires_at = now + TTL` (same token); 404 otherwise.

`state` is `ACTIVE | EXPIRED | INVALIDATED`, decided by the backend clock (`expires_at <= now` is EXPIRED). Clients' countdowns are display only.

TTL: `SPLIT_SESSION_TTL` Go duration, default `5m`, allowed 1m-60m, invalid value stops startup.

Participant gate (used by VP-015b): `Store.CheckSession(ctx, token, now)` -> session or `SESSION_INVALID` (401). Valid only while ACTIVE, `now < expires_at`, and receipt not FINALIZED. Expiry affects access only; receipt data is untouched.

Schema: `public.split_sessions` (Receipt-owned), one ACTIVE row per receipt (partial unique index), owner-only SELECT RLS without `token_hash`, mutations only via service-role-only `split_session_*` functions that take the backend clock as `p_now`.
