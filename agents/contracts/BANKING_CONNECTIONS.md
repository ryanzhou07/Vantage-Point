# Banking connections API (VP-018c)

All routes need a verified Foundation session; the user id comes from it, never the body (unknown body fields are rejected). Responses are `Cache-Control: no-store`. Errors: `{"error":{"code":"..."}}`.

| Route | Result |
|---|---|
| `POST /api/v1/banking/link-token` | `200 {"link_token","expiration"}` for Plaid Link. Rate limited. |
| `POST /api/v1/banking/connections` body `{"public_token"}` | Backend exchanges it, encrypts the access token (AES-256-GCM, AAD user+connection), stores it. `201` new, `200` existing item. Institution name comes from Plaid, not the browser. |
| `GET /api/v1/banking/connections` | `200 {"connections":[Connection]}` |
| `GET /api/v1/banking/connections/{id}` | `200 Connection` or `404` |
| `DELETE /api/v1/banking/connections/{id}` | Removes the Item at Plaid, then the row: `204`. Provider failure: `502` and the credential is kept so retry works. |

`Connection`: `id, provider ("plaid"), institution_name, status (ACTIVE|NEEDS_REAUTH|ERROR), last_sync_attempt_at, last_successful_sync_at, created_at`. No token or credential field is ever returned.

Codes: `400 INVALID_REQUEST|INVALID_TOKEN`, `401 UNAUTHENTICATED`, `404 NOT_FOUND`, `405`, `429 RATE_LIMITED` (10 provider calls/min/user, per instance), `502 PROVIDER_UNAVAILABLE`, `503 UNAVAILABLE`.

Database: `public.institution_connections`, RLS owner read, `encrypted_credential` excluded from browser grants, writes only via service-role functions `banking_enroll/connections/credential/disconnect`. If persisting fails after the exchange, the backend removes the fresh Item at Plaid (best effort).
