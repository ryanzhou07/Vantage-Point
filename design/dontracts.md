# Vantage-Point API and Event Contracts v1

## 1. Purpose

This document defines the initial communication contracts between Vantage-Point frontend, backend domains, workers, and external provider integrations.

The goal is to prevent different agents from independently inventing incompatible APIs.

Contracts describe:

- HTTP API structure
- Authentication expectations
- Request/response conventions
- Error conventions
- Banking endpoints
- Receipt endpoints
- Dashboard aggregation endpoints
- Realtime receipt messages
- Internal asynchronous events
- Provider webhook boundaries

This document defines logical contracts.

Exact Go types and TypeScript types may later be generated or refined from these contracts.

---

## 2. Contract Principles

Vantage-Point contracts should follow these principles:

1. Frontend communicates with Vantage-Point APIs rather than provider APIs directly.
2. Authentication identity comes from the verified session, not client-supplied user IDs.
3. Internal IDs are Vantage-Point IDs.
4. Provider-specific identifiers should not leak into unrelated frontend contracts unless needed.
5. Monetary values must use exact integer minor units.
6. Currency must be explicit where appropriate.
7. API errors should use a consistent structure.
8. Domain boundaries should remain clear.
9. Realtime messages should reflect authoritative backend state.
10. Contracts should be versioned deliberately when breaking changes occur.

---

# 3. API Base

Initial HTTP API convention:

```text
/api/v1
```

Examples:

```text
GET  /api/v1/accounts
GET  /api/v1/transactions
POST /api/v1/receipts
```

Versioning from the start gives Vantage-Point a controlled path for future breaking API changes.

---

# 4. Authentication

Protected owner APIs require a valid authenticated Vantage-Point session.

Conceptually:

```text
Browser
  ↓
Supabase session token
  ↓
Vantage-Point API
  ↓
Verify token
  ↓
ctx.user_id
```

The authenticated user must be derived from the verified token.

Endpoints must not use request fields such as:

```text
user_id
owner_user_id
```

as proof of identity.

---

# 5. Participant Authentication

Receipt participants are not authenticated Vantage-Point users.

Participant endpoints use temporary receipt-session authorization.

Conceptually:

```text
Participant Browser
      ↓
Receipt session token
      ↓
Receipt API
      ↓
Validate session
      ↓
Participant-scoped authorization
```

Participant authorization and authenticated-owner authorization are separate mechanisms.

---

# 6. Content Type

HTTP APIs should primarily use:

```text
application/json
```

Receipt image upload may use:

```text
multipart/form-data
```

or a signed-upload flow.

Exact upload mechanism can be selected during implementation.

---

# 7. ID Format

Vantage-Point IDs are represented as UUID strings.

Example:

```json
{
  "id": "f0ee55ca-bf27-4b31-b307-8ea233437ed9"
}
```

Frontend code should treat IDs as opaque strings.

---

# 8. Money Contract

All authoritative monetary values exposed by APIs should use integer minor units.

Example:

```json
{
  "amount_minor": 4219,
  "currency": "USD"
}
```

Meaning:

```text
$42.19
```

Do not send authoritative money as:

```json
{
  "amount": 42.19
}
```

---

# 9. Percentage Contract

When receipt percentage allocations are represented:

```json
{
  "percentage_bps": 5000
}
```

means:

```text
50.00%
```

Range:

```text
0 → 10000
```

where applicable.

---

# 10. Timestamp Contract

API timestamps should use ISO 8601 UTC-compatible strings.

Example:

```json
{
  "created_at": "2026-10-07T18:24:00Z"
}
```

Frontend may convert timestamps to local display time.

---

# 11. Date Contract

Date-only financial values should use:

```text
YYYY-MM-DD
```

Example:

```json
{
  "transaction_date": "2026-10-07"
}
```

---

# 12. Standard Success Envelope

Simple resource responses may return the resource directly.

Example:

```json
{
  "id": "...",
  "name": "Checking"
}
```

Collection endpoints may use:

```json
{
  "data": [],
  "next_cursor": null
}
```

Vantage-Point should avoid unnecessary deeply nested response wrappers.

---

# 13. Error Contract

All application API errors should use a consistent structure.

Recommended:

```json
{
  "error": {
    "code": "RECEIPT_NOT_FOUND",
    "message": "Receipt was not found.",
    "request_id": "..."
  }
}
```

Optional additional details:

```json
{
  "error": {
    "code": "VALIDATION_ERROR",
    "message": "The request was invalid.",
    "request_id": "...",
    "details": {
      "field": "amount_minor"
    }
  }
}
```

Sensitive internal information must not appear in public error responses.

---

# 14. HTTP Status Conventions

Recommended use:

```text
200 OK
201 Created
202 Accepted
204 No Content

400 Bad Request
401 Unauthorized
403 Forbidden
404 Not Found
409 Conflict
422 Unprocessable Entity
429 Too Many Requests

500 Internal Server Error
502 Bad Gateway
503 Service Unavailable
```

Domain-specific error codes should provide more precise meaning.

---

# 15. Pagination

Large collections should support cursor-based pagination where practical.

Conceptual request:

```text
GET /api/v1/transactions?cursor=abc123&limit=50
```

Response:

```json
{
  "data": [],
  "next_cursor": "def456"
}
```

Default and maximum limits can be defined later.

---

# 16. Idempotency

APIs that may be retried and could create duplicate financial state should support idempotent behavior.

Examples:

- Receipt creation
- Provider event processing
- Finalization
- Certain worker operations

Public API idempotency keys may be introduced where needed.

Conceptually:

```text
Idempotency-Key: unique-client-generated-value
```

Exact use should be task-specific rather than added blindly to every endpoint.

---

# 17. Foundation API

The Foundation domain provides authenticated-user context and basic profile information.

Initial endpoint:

```text
GET /api/v1/me
```

---

# 18. GET /me

Returns the authenticated user's Vantage-Point profile.

Example response:

```json
{
  "id": "user-uuid",
  "display_name": "Ryan",
  "email": "ryan@example.com",
  "avatar_url": "https://...",
  "created_at": "2026-10-07T18:24:00Z"
}
```

The API derives the user from authentication.

No user ID is supplied by the client.

---

# 19. Banking API Overview

Initial Banking endpoints:

```text
GET    /api/v1/banking/connections
POST   /api/v1/banking/connections
GET    /api/v1/banking/connections/{connection_id}
DELETE /api/v1/banking/connections/{connection_id}

GET    /api/v1/accounts
GET    /api/v1/accounts/{account_id}

GET    /api/v1/transactions
GET    /api/v1/accounts/{account_id}/transactions

GET    /api/v1/recurring
```

Provider-specific connection bootstrap endpoints may also be required.

---

# 20. Banking Connection Response

Conceptual response:

```json
{
  "id": "connection-uuid",
  "provider": "TELLER",
  "institution_name": "Chase",
  "status": "ACTIVE",
  "last_successful_sync_at": "2026-10-07T18:00:00Z",
  "created_at": "2026-10-01T14:00:00Z"
}
```

Provider credentials must never be included.

---

# 21. Create Banking Connection

Conceptual:

```text
POST /api/v1/banking/connections
```

Request may contain provider-specific connection completion data produced by approved Teller/Plaid flows.

Example conceptual request:

```json
{
  "provider": "TELLER",
  "provider_connection_token": "temporary-provider-result"
}
```

The exact Teller/Plaid bootstrap payload should be defined from current provider SDK requirements during implementation.

Long-lived provider credentials must be handled server-side.

---

# 22. Banking Provider Boundary

Provider-specific structures must be isolated.

Conceptually:

```text
Teller response
   ↓
Teller adapter
   ↓
Vantage-Point Banking types
```

and:

```text
Plaid response
   ↓
Plaid adapter
   ↓
Vantage-Point Banking types
```

Frontend APIs should expose normalized Vantage-Point resources.

---

# 23. Account Contract

Example account:

```json
{
  "id": "account-uuid",
  "connection_id": "connection-uuid",
  "institution_name": "Chase",
  "name": "Total Checking",
  "account_type": "CHECKING",
  "account_subtype": null,
  "mask": "1234",
  "currency": "USD",
  "current_balance_minor": 243122,
  "available_balance_minor": 230000,
  "status": "ACTIVE",
  "last_synced_at": "2026-10-07T18:00:00Z"
}
```

---

# 24. Account Types

Initial normalized values:

```text
CHECKING
SAVINGS
CREDIT_CARD
OTHER
```

Provider values should be translated into this internal vocabulary where possible.

---

# 25. Transaction Contract

Example:

```json
{
  "id": "transaction-uuid",
  "account_id": "account-uuid",
  "status": "POSTED",
  "amount_minor": -5421,
  "currency": "USD",
  "description": "WHOLE FOODS MARKET",
  "merchant_name": "Whole Foods",
  "transaction_date": "2026-10-06",
  "authorized_at": "2026-10-06T17:30:00Z",
  "posted_at": "2026-10-07T09:00:00Z",
  "category": "GROCERIES"
}
```

Internal convention:

```text
negative = money out
positive = money in
```

---

# 26. Transaction Query

Conceptual:

```text
GET /api/v1/transactions
```

Potential query parameters:

```text
account_id
status
start_date
end_date
cursor
limit
```

Example:

```text
GET /api/v1/transactions?account_id=...&limit=50
```

---

# 27. Recurring Activity Contract

Example:

```json
{
  "id": "recurring-uuid",
  "account_id": "account-uuid",
  "merchant_name": "Spotify",
  "description": "Spotify Premium",
  "estimated_amount_minor": -1199,
  "currency": "USD",
  "frequency": "MONTHLY",
  "next_expected_date": "2026-11-02",
  "confidence": 0.91,
  "source": "VANTAGE_POINT",
  "status": "ACTIVE"
}
```

Confidence may be omitted if the selected detection system does not provide a meaningful value.

---

# 28. Receipt API Overview

Owner endpoints:

```text
POST   /api/v1/receipts
GET    /api/v1/receipts
GET    /api/v1/receipts/{receipt_id}
PATCH  /api/v1/receipts/{receipt_id}

POST   /api/v1/receipts/{receipt_id}/review
POST   /api/v1/receipts/{receipt_id}/participants
POST   /api/v1/receipts/{receipt_id}/sessions
POST   /api/v1/receipts/{receipt_id}/finalize
POST   /api/v1/receipts/{receipt_id}/reset
```

Participant and allocation endpoints are defined separately.

---

# 29. Create Receipt

Conceptual:

```text
POST /api/v1/receipts
```

Image may be uploaded in the same operation or through a prior signed-upload flow.

Initial response should not wait for OCR completion.

Example:

```json
{
  "id": "receipt-uuid",
  "status": "PROCESSING",
  "processing_status": "QUEUED",
  "created_at": "2026-10-07T18:24:00Z"
}
```

HTTP status may be:

```text
201 Created
```

or:

```text
202 Accepted
```

depending on final upload flow.

---

# 30. Receipt Contract

Example owner-facing receipt:

```json
{
  "id": "receipt-uuid",
  "merchant_name": "Restaurant Example",
  "receipt_date": "2026-10-07",
  "currency": "USD",
  "subtotal_minor": 10000,
  "tax_minor": 662,
  "tip_minor": 2000,
  "discount_minor": 0,
  "total_minor": 12662,
  "status": "NEEDS_REVIEW",
  "revision": 1,
  "tax_allocation_method": "PROPORTIONAL",
  "tip_allocation_method": "PROPORTIONAL",
  "items": [],
  "participants": [],
  "created_at": "...",
  "updated_at": "..."
}
```

---

# 31. Receipt Item Contract

Example:

```json
{
  "id": "item-uuid",
  "description": "Pizza",
  "quantity": 1,
  "unit_price_minor": 2400,
  "total_price_minor": 2400,
  "position": 1
}
```

---

# 32. Owner Receipt Editing

Before claiming begins:

```text
PATCH /api/v1/receipts/{receipt_id}
```

may allow verified receipt fields to be corrected.

Conceptual request:

```json
{
  "merchant_name": "Joe's Pizza",
  "tax_minor": 662,
  "tip_minor": 2000
}
```

Receipt item editing may use nested input or dedicated item endpoints.

The final implementation should favor clear validation over overly generic patch operations.

---

# 33. Receipt Review Completion

Conceptual:

```text
POST /api/v1/receipts/{receipt_id}/review
```

Meaning:

```text
Owner confirms receipt financial structure is ready.
```

Successful transition:

```text
NEEDS_REVIEW
→
READY_TO_SHARE
```

The backend validates receipt consistency before transitioning.

---

# 34. Receipt Participant Contract

Example:

```json
{
  "id": "participant-uuid",
  "name": "Alex",
  "is_owner": false,
  "status": "INVITED"
}
```

Participants are receipt-scoped.

No global user ID is required.

---

# 35. Add Participants

Conceptual:

```text
POST /api/v1/receipts/{receipt_id}/participants
```

Request:

```json
{
  "participants": [
    {
      "name": "Alex"
    },
    {
      "name": "Chris"
    }
  ]
}
```

The owner may also be represented as a participant.

---

# 36. Start Split Session

Conceptual:

```text
POST /api/v1/receipts/{receipt_id}/sessions
```

Response:

```json
{
  "session_id": "session-uuid",
  "participant_url": "https://...",
  "expires_at": "2026-10-07T19:00:00Z"
}
```

The raw access token may exist only inside the generated URL.

The database stores its secure hash.

---

# 37. Participant Join

Conceptual endpoint:

```text
POST /api/v1/receipt-sessions/{session_token}/join
```

Request:

```json
{
  "participant_id": "participant-uuid"
}
```

Successful result establishes a participant-scoped session/context.

The exact mechanism may use a secure participant session cookie or temporary token.

---

# 38. Participant Receipt View

Conceptual:

```text
GET /api/v1/participant/receipts/{receipt_id}
```

Participant authorization determines receipt access.

Response should expose only information needed for participation.

It must not expose:

- Owner banking information
- Other private user data
- Provider credentials
- Internal security metadata

---

# 39. Allocation Contract

Example:

```json
{
  "id": "allocation-uuid",
  "receipt_item_id": "item-uuid",
  "participant_id": "participant-uuid",
  "amount_minor": 1200,
  "percentage_bps": 5000,
  "status": "DRAFT",
  "source": "PARTICIPANT"
}
```

---

# 40. Create or Update Allocation

Conceptual:

```text
PUT /api/v1/participant/items/{item_id}/allocation
```

Example:

```json
{
  "amount_minor": 1200,
  "percentage_bps": 5000
}
```

The backend must atomically verify available item capacity.

---

# 41. Allocation Conflict

If another participant consumes the available amount first, the API should reject the stale request.

Recommended:

```text
409 Conflict
```

Example:

```json
{
  "error": {
    "code": "ALLOCATION_CONFLICT",
    "message": "The requested portion is no longer available.",
    "request_id": "..."
  }
}
```

Client then refreshes/reconciles with authoritative state.

---

# 42. Equal Split Action

Equal split may be expressed as a domain action instead of forcing the frontend to calculate allocations independently.

Conceptual owner or participant request:

```text
POST /api/v1/receipts/{receipt_id}/items/{item_id}/equal-split
```

Request:

```json
{
  "participant_ids": [
    "participant-a",
    "participant-b",
    "participant-c"
  ]
}
```

Backend applies deterministic cent rounding.

---

# 43. Participant Submission

Conceptual:

```text
POST /api/v1/participant/receipts/{receipt_id}/submit
```

Response:

```json
{
  "participant_id": "...",
  "status": "SUBMITTED",
  "submission_version": 1,
  "submitted_at": "..."
}
```

Submission locks the participant's normal editing flow.

---

# 44. Owner Correction

Owner corrections should be explicit domain operations.

Conceptual:

```text
PATCH /api/v1/receipts/{receipt_id}/allocations/{allocation_id}
```

or a dedicated correction endpoint.

Request may contain:

```json
{
  "amount_minor": 1500,
  "reason": "Participant selected the wrong share."
}
```

Backend should:

1. Validate owner access.
2. Apply correction.
3. Record receipt audit event.
4. Mark affected participant `RECONFIRMATION_REQUIRED` where appropriate.

---

# 45. Reconfirmation

Conceptual participant action:

```text
POST /api/v1/participant/receipts/{receipt_id}/confirm
```

Response:

```json
{
  "participant_id": "...",
  "status": "CONFIRMED",
  "submission_version": 2,
  "confirmed_at": "..."
}
```

Backend verifies that confirmation corresponds to the latest required submission version.

---

# 46. Receipt Reset

Conceptual:

```text
POST /api/v1/receipts/{receipt_id}/reset
```

Request:

```json
{
  "reason": "OCR missed two receipt items."
}
```

Backend should:

- Validate owner.
- Increment receipt revision.
- Invalidate incompatible active allocations/submissions.
- Record audit event.
- Return receipt to appropriate review state.

---

# 47. Receipt Finalization

Conceptual:

```text
POST /api/v1/receipts/{receipt_id}/finalize
```

The backend must verify:

- Receipt state permits finalization.
- All required item value is allocated.
- Allocations are financially valid.
- Required confirmations are complete.
- Tax/tip can be resolved.
- Final participant amounts can be calculated exactly.

Finalization creates immutable receivable obligations.

---

# 48. Finalization Response

Example:

```json
{
  "receipt_id": "receipt-uuid",
  "status": "FINALIZED",
  "finalized_at": "...",
  "receivables": [
    {
      "id": "receivable-1",
      "participant_id": "participant-a",
      "amount_owed_minor": 2534,
      "currency": "USD",
      "status": "OWED"
    }
  ]
}
```

---

# 49. Receivable Contract

Example:

```json
{
  "id": "receivable-uuid",
  "receipt_id": "receipt-uuid",
  "participant": {
    "id": "participant-uuid",
    "name": "Alex"
  },
  "subtotal_minor": 2000,
  "tax_minor": 134,
  "tip_minor": 400,
  "discount_minor": 0,
  "amount_owed_minor": 2534,
  "currency": "USD",
  "status": "OWED",
  "created_at": "...",
  "paid_at": null
}
```

---

# 50. Mark Receivable Paid

Conceptual:

```text
POST /api/v1/receivables/{receivable_id}/mark-paid
```

Response:

```json
{
  "id": "receivable-uuid",
  "status": "PAID",
  "paid_at": "2026-10-07T20:00:00Z"
}
```

Only the authenticated receipt owner may perform this action.

---

# 51. Dashboard API

The Dashboard may initially use domain endpoints directly.

However, an overview endpoint can reduce frontend request fan-out.

Potential:

```text
GET /api/v1/dashboard
```

---

# 52. Dashboard Response

Conceptual:

```json
{
  "banking": {
    "connected_cash_minor": 842031,
    "currency": "USD",
    "connections_needing_attention": 0
  },
  "receipts": {
    "active_receipt_count": 2,
    "receipts_needing_action": 1,
    "outstanding_receivables_minor": 12644,
    "currency": "USD"
  }
}
```

This endpoint provides derived presentation information.

It must not become an independent source of truth.

---

# 53. Realtime Receipt Requirements

Receipt allocation changes should be observable by connected participants.

Realtime transport may use Supabase Realtime or another approved mechanism.

Realtime payloads should identify:

- Receipt
- Revision
- Event type
- Updated resource/version
- Server timestamp

---

# 54. Realtime Event Envelope

Recommended logical structure:

```json
{
  "event": "receipt.allocation.updated",
  "receipt_id": "receipt-uuid",
  "receipt_revision": 1,
  "occurred_at": "2026-10-07T18:30:00Z",
  "data": {}
}
```

Realtime event data should reflect committed backend state.

---

# 55. Receipt Realtime Event Types

Initial useful events may include:

```text
receipt.updated
receipt.item.updated

receipt.participant.joined
receipt.participant.updated

receipt.allocation.created
receipt.allocation.updated
receipt.allocation.removed

receipt.participant.submitted
receipt.participant.reconfirmation_required
receipt.participant.confirmed

receipt.session.updated

receipt.finalized
```

Not every database write needs to produce a public realtime event.

---

# 56. Realtime Allocation Event

Example:

```json
{
  "event": "receipt.allocation.updated",
  "receipt_id": "receipt-uuid",
  "receipt_revision": 1,
  "occurred_at": "...",
  "data": {
    "item_id": "item-uuid",
    "allocations": [
      {
        "participant_id": "participant-a",
        "amount_minor": 1200,
        "percentage_bps": 5000
      }
    ],
    "remaining_minor": 1200
  }
}
```

This allows clients to reconcile to current authoritative state.

---

# 57. Realtime Security

Realtime access must be authorization-scoped.

An authenticated owner may subscribe to their own receipt.

A participant may subscribe only to the receipt/session they are authorized to access.

Knowing a receipt UUID must not be sufficient to subscribe.

---

# 58. Internal Event Philosophy

Internal background work should use events/jobs where asynchronous processing is appropriate.

Examples:

```text
bank.connection.created
bank.sync.requested
bank.sync.completed

receipt.uploaded
receipt.vision.requested
receipt.vision.completed
```

These are internal workflow signals, not necessarily public client events.

---

# 59. Internal Event Envelope

Recommended:

```json
{
  "event_id": "event-uuid",
  "event_type": "receipt.vision.requested",
  "occurred_at": "...",
  "resource_id": "receipt-uuid",
  "resource_version": 1,
  "payload": {}
}
```

Consumers should be designed to tolerate duplicate delivery.

---

# 60. Receipt Vision Workflow

Conceptually:

```text
Receipt API
   ↓
receipt.vision.requested
   ↓
Vision Worker
   ↓
OCR / Model
   ↓
Database
   ↓
receipt.vision.completed
```

A retry must not create duplicate receipt items.

---

# 61. Banking Sync Workflow

Conceptually:

```text
Connection Created
      ↓
bank.sync.requested
      ↓
Banking Worker
      ↓
Teller / Plaid
      ↓
Normalize
      ↓
PostgreSQL
      ↓
bank.sync.completed
```

The sync worker must be idempotent.

---

# 62. Banking Internal Events

Potential internal events:

```text
bank.connection.created

bank.sync.requested
bank.sync.started
bank.sync.completed
bank.sync.failed

bank.connection.reauth_required

bank.provider.event.received
```

These events may initially be represented by queue jobs rather than a dedicated event-bus system.

---

# 63. Receipt Internal Events

Potential:

```text
receipt.created
receipt.vision.requested
receipt.vision.completed
receipt.vision.failed

receipt.session.started

receipt.participant.submitted
receipt.reconfirmation.required

receipt.finalized

receivable.created
receivable.paid
```

Only events useful for decoupling or asynchronous processing should be implemented.

Do not create an event system merely because an event name is listed here.

---

# 64. Provider Webhooks

External provider webhooks terminate at Vantage-Point backend endpoints.

Conceptually:

```text
POST /api/v1/webhooks/teller
POST /api/v1/webhooks/plaid
```

These endpoints are not authenticated using normal user sessions.

They must use provider-supported authenticity verification.

---

# 65. Teller Webhook Boundary

Teller payloads should be accepted and validated only in the Teller integration boundary.

Conceptually:

```text
Teller
 ↓
Teller webhook handler
 ↓
Validate authenticity
 ↓
Record provider event
 ↓
Trigger synchronization
```

Teller payload structures should not propagate directly into unrelated domain code.

---

# 66. Plaid Webhook Boundary

Similarly:

```text
Plaid
 ↓
Plaid webhook handler
 ↓
Validate authenticity
 ↓
Record provider event
 ↓
Trigger synchronization
```

The internal Banking model remains provider-neutral.

---

# 67. Webhook Response Behavior

Webhook endpoints should respond quickly after safe validation/recording.

Expensive synchronization work should generally occur asynchronously.

Conceptually:

```text
Webhook Received
      ↓
Validate
      ↓
Persist event
      ↓
Queue work
      ↓
200 OK
```

The provider should not have to wait for a full account synchronization request.

---

# 68. Contract Ownership

Foundation Agent owns:

```text
/me
authentication context
shared auth errors
```

Banking Agent owns:

```text
banking connections
accounts
transactions
recurring activity
bank webhooks
bank sync events
```

Receipt Agent owns:

```text
receipts
items
participants
sessions
allocations
submissions
finalization
receivables
receipt realtime
receipt worker events
```

Dashboard Agent owns:

```text
dashboard presentation contracts
```

but may not redefine underlying Banking/Receipt resource semantics.

---

# 69. Shared Contract Changes

Breaking shared-contract changes require integration review.

Examples:

```text
Changing amount_minor to decimal string

Renaming account status values

Changing receipt allocation semantics

Changing authentication identity format
```

Agents must not independently introduce incompatible versions.

---

# 70. Type Generation

Long term, shared API types should preferably be generated from one contract source.

Possible future approach:

```text
OpenAPI
   ↓
TypeScript types
Go types / clients
```

For MVP setup, hand-written contracts may be acceptable while the API stabilizes.

Vantage-Point should avoid maintaining multiple contradictory definitions manually.

---

# 71. OpenAPI

A future `openapi.yaml` may become the machine-readable HTTP contract source.

Recommended future structure:

```text
contracts/
├── openapi.yaml
├── events/
└── realtime/
```

Do not create a massive OpenAPI specification before the first routes are implemented.

Start with the approved logical contract in this document and formalize machine-readable contracts incrementally.

---

# 72. API Security Principles

Every endpoint should answer:

```text
Who is making this request?

What resource are they trying to access?

Are they authorized to perform this action?
```

Authentication alone is not sufficient.

Resource ownership must also be validated.

---

# 73. Owner Resource Pattern

Example:

```text
GET /api/v1/receipts/{receipt_id}
```

Backend flow:

```text
Authenticate user
      ↓
Load receipt
      ↓
Verify receipt.owner_user_id == ctx.user_id
      ↓
Return receipt
```

A frontend-provided owner ID is unnecessary.

---

# 74. Banking Resource Pattern

Example:

```text
GET /api/v1/accounts/{account_id}
```

Flow:

```text
Authenticate user
      ↓
Load account
      ↓
Load/verify connection ownership
      ↓
Return account
```

---

# 75. Participant Resource Pattern

Participant flow:

```text
Validate receipt session
      ↓
Validate participant identity/context
      ↓
Validate receipt revision/state
      ↓
Perform participant operation
```

Participant APIs must not gain access to unrelated authenticated-user resources.

---

# 76. Optimistic Concurrency

Where stale client state could cause financial conflicts, clients should be prepared for server rejection.

Receipt allocations are the most important example.

Optional future mechanisms include:

```text
resource_version
ETag
If-Match
```

MVP may initially rely on transactional backend validation plus `409 Conflict`.

---

# 77. Client State Rules

Frontend state is a cache/presentation representation.

It is not authoritative.

After mutations:

```text
request
 ↓
server validation
 ↓
database commit
 ↓
response / realtime event
 ↓
frontend updates
```

Clients should reconcile to server state after conflict.

---

# 78. Validation Responsibility

Frontend:

- Immediate UX feedback
- Basic formatting
- Prevent obvious invalid input

Backend:

- Authentication
- Authorization
- Financial validation
- State transitions
- Allocation constraints
- Finalization
- Idempotency

Database:

- Referential integrity
- Uniqueness
- Core constraints

---

# 79. API Logging

API logs may include:

- Request ID
- Route
- HTTP method
- Status code
- Safe resource ID
- Safe authenticated user ID where appropriate
- Latency

Logs must not intentionally include:

- Supabase session tokens
- Teller credentials
- Plaid credentials
- Receipt bearer tokens
- Raw banking secrets

---

# 80. Request IDs

Every API request should eventually have a unique request/correlation identifier.

Example:

```text
request_id
```

This allows errors and distributed worker activity to be traced without exposing sensitive details.

---

# 81. Contract Error Codes

Potential shared codes:

```text
AUTHENTICATION_REQUIRED
FORBIDDEN
RESOURCE_NOT_FOUND
VALIDATION_ERROR
CONFLICT
RATE_LIMITED
INTERNAL_ERROR
DEPENDENCY_UNAVAILABLE
```

Banking examples:

```text
BANK_CONNECTION_NOT_FOUND
BANK_CONNECTION_REAUTH_REQUIRED
BANK_PROVIDER_UNAVAILABLE
ACCOUNT_NOT_FOUND
```

Receipt examples:

```text
RECEIPT_NOT_FOUND
INVALID_RECEIPT_STATE
SESSION_EXPIRED
PARTICIPANT_NOT_FOUND
ALLOCATION_CONFLICT
RECONFIRMATION_REQUIRED
RECEIPT_NOT_READY_TO_FINALIZE
```

---

# 82. External Dependency Errors

If Teller, Plaid, Supabase, or the vision provider fails, the API should translate that failure into Vantage-Point semantics.

Bad:

```text
TellerError 42942 API_NODE_X...
```

Better:

```json
{
  "error": {
    "code": "BANK_PROVIDER_UNAVAILABLE",
    "message": "Banking information is temporarily unavailable.",
    "request_id": "..."
  }
}
```

Detailed provider errors may remain in secure internal logs.

---

# 83. Contract Versioning

Initial version:

```text
v1
```

Minor backward-compatible additions do not require a new API version.

Breaking changes should require deliberate migration/version planning.

Do not create `v2` for every internal change.

---

# 84. Contract Testing

Important contracts should eventually have automated tests.

Examples:

- Unauthorized `/me` fails.
- User cannot request another user's receipt.
- User cannot request another user's account.
- Allocation conflict returns expected error.
- Finalize rejects unresolved receipt.
- Teller/Plaid adapters produce normalized account structures.
- Money remains integer minor units.

---

# 85. MVP Contract Surface

The initial API does not need every endpoint in this document on day one.

Implementation should proceed in vertical slices.

Example first Foundation slice:

```text
GET /api/v1/me
```

Example first Banking slice:

```text
Connect Teller
GET /accounts
GET /transactions
```

Example first Receipt slice:

```text
POST /receipts
GET /receipts/{id}
OCR processing
owner review
```

Additional endpoints should be implemented as their tasks become active.

---

# 86. Explicitly Deferred Contracts

Do not create APIs yet for:

```text
safe-to-spend
budgets
forecasts
investments
card rewards
card recommendations
Venmo matching
Zelle matching
payment initiation
financial advice
administration
```

These are outside MVP 1.

---

# 87. Contract Summary

Core flow:

```text
Frontend
   ↓
/api/v1
   ↓
Authentication / Participant Authorization
   ↓
Domain
   ├── Foundation
   ├── Banking
   ├── Receipts
   └── Dashboard
   ↓
PostgreSQL
```

External systems:

```text
Teller / Plaid
      ↓
Banking Adapters
      ↓
Banking Domain
```

and:

```text
Vision Provider
      ↓
Vision Worker
      ↓
Receipt Domain
```

Realtime:

```text
Receipt Domain
      ↓
Committed Database State
      ↓
Realtime Events
      ↓
Participants / Owner
```

---

# 88. Decisions Locked by This Design

If approved:

```text
API base:
    /api/v1

Owner authentication:
    Supabase authenticated session

Participant authentication:
    Temporary receipt-session authorization

Money:
    Integer minor units

Percentages:
    Basis points

Errors:
    Shared structured error format

Banking API:
    Provider-neutral

Receipt API:
    State/action oriented

Allocation conflicts:
    Server rejection / 409

Receipt realtime:
    Server-authoritative event updates

Provider webhooks:
    Teller/Plaid-specific boundary handlers

Async work:
    Internal idempotent jobs/events

Dashboard:
    Reads domain contracts rather than owning financial state
```

---

# 89. Remaining Implementation Decisions

The following can be finalized during implementation:

```text
Exact Teller connection bootstrap payload

Exact Plaid connection bootstrap payload

Exact receipt image upload transport

Exact pagination cursor encoding

Exact session cookie/token mechanics

Exact realtime transport

Exact queue implementation

Exact OpenAPI generation tooling

Exact API router/framework choices in Go
```

These decisions should not change the approved domain semantics.