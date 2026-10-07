# Vantage-Point Technical Design

## 1. Design Goals

Vantage-Point should be:

- Financially correct
- Secure
- Mobile-first
- Resistant to traffic spikes and abuse
- Simple enough to build as an MVP
- Easy to test locally
- Compatible with multiple feature agents
- Capable of evolving without premature microservices

Core rule:

> Keep authoritative financial state in the backend/database and treat clients, external providers, retries, and realtime messages as potentially unreliable.

---

# 2. Technology Stack

## Frontend

```text
Next.js
TypeScript
App Router
Tailwind CSS
```

Initial hosting direction:

```text
Vercel
```

## Backend

```text
Go
```

Initial hosting direction:

```text
Google Cloud Run
```

## Database / Platform

```text
Supabase
PostgreSQL
Supabase Auth
Supabase Realtime
Supabase Storage
```

## Banking

```text
Teller → primary
Plaid  → secondary/fallback
```

## Receipt Processing

Multimodal OCR/vision provider behind an internal abstraction.

Exact provider is deferred.

---

# 3. High-Level Architecture

```text
                     Browser
              Next.js / TypeScript
                        │
                      HTTPS
                        │
                        ▼
                Vantage-Point API
                       Go
                        │
       ┌────────────────┼────────────────┐
       │                │                │
 Foundation          Banking          Receipts
                        │                │
                        ▼                ▼
                 Banking Worker    Vision Worker
                        │                │
                        ▼                ▼
                 Teller / Plaid    Vision Provider
                        │                │
                        └───────┬────────┘
                                ▼
                       Supabase/PostgreSQL
```

The backend begins as a modular monolith.

Do not create a large microservice system for MVP.

---

# 4. Runtime Components

Initial runtime components:

```text
1. Next.js Web Application
2. Go Backend API
3. Banking Worker
4. Receipt Vision Worker
5. Supabase/PostgreSQL
```

Workers may initially share infrastructure/binaries if doing so materially simplifies deployment.

---

# 5. Mobile Architecture

Vantage-Point is mobile-first.

The web application must work as a first-class experience on modern mobile browsers.

Critical mobile flows include:

- Dashboard
- Account balances
- Transactions
- Receipt upload
- QR joining
- Item claiming
- Submission
- Reconfirmation
- Receivable management

The temporary receipt experience must not require installing an application.

A future native client may use the same backend:

```text
             Go API
            /      \
      Next.js      Native App
```

A native application is not required for MVP.

---

# 6. Backend Structure

Logical Go structure:

```text
services/backend/
├── cmd/
│   ├── api/
│   ├── bank-worker/
│   └── receipt-worker/
│
└── internal/
    ├── foundation/
    ├── banking/
    ├── receipts/
    ├── dashboard/
    └── platform/
```

`platform` may contain:

- Database
- Authentication integration
- Configuration
- Logging
- Jobs
- Rate limiting
- Cache integration

Feature logic must not be dumped into generic utility packages.

---

# 7. Authentication

Flow:

```text
Google
  ↓
Supabase Auth
  ↓
Session/JWT
  ↓
Go API
  ↓
Token validation
  ↓
ctx.user_id
```

`auth.users.id` is the canonical Vantage-Point user identity.

The browser does not establish identity by submitting a user ID.

---

# 8. Database Strategy

Use one physical PostgreSQL database for MVP.

Do not create separate physical Banking and Receipt databases.

Logical ownership:

```text
Foundation
Banking
Receipts
```

may be represented through naming/module conventions and potentially schemas later.

Dashboard owns no authoritative financial tables.

---

# 9. Database IDs

Internal application IDs use UUIDs.

External provider identifiers remain separate from internal primary keys.

Frontend treats IDs as opaque strings.

---

# 10. Time

Stored timestamps should generally use PostgreSQL:

```text
TIMESTAMPTZ
```

HTTP timestamps use ISO 8601.

Date-only financial fields use:

```text
YYYY-MM-DD
```

---

# 11. Money

Authoritative money is stored as integer minor units.

```text
$42.19 → 4219
```

Recommended PostgreSQL representation:

```text
BIGINT
```

Currency is stored separately.

MVP primarily assumes USD.

Floating point must not be used for authoritative financial calculations.

---

# 12. Percentages

Persisted percentages use basis points where needed.

```text
10000 = 100%
5000  = 50%
2500  = 25%
```

Receipt allocation's authoritative value remains exact money, not percentage.

---

# 13. Foundation Data

Primary application table:

```text
profiles
```

Conceptual fields:

```text
id
display_name
email
avatar_url
created_at
updated_at
```

`profiles.id` references the Supabase authenticated user identity.

One profile exists per authenticated Vantage-Point user.

---

# 14. Banking Data Model

Primary Banking tables:

```text
institution_connections
accounts
transactions
provider_events
recurring_activity
```

## Institution Connections

Conceptual fields:

```text
id
user_id
provider
provider_connection_id
institution_name
institution_provider_id
status
credential_reference
last_successful_sync_at
last_sync_attempt_at
created_at
updated_at
```

Providers:

```text
TELLER
PLAID
```

## Accounts

Conceptual fields:

```text
id
institution_connection_id
provider_account_id
name
official_name
account_type
account_subtype
mask
currency
current_balance_minor
available_balance_minor
status
last_synced_at
created_at
updated_at
```

Types:

```text
CHECKING
SAVINGS
CREDIT_CARD
OTHER
```

## Transactions

Conceptual fields:

```text
id
account_id
provider
provider_transaction_id
pending_provider_transaction_id
status
amount_minor
currency
description
merchant_name
transaction_date
authorized_at
posted_at
category
raw_category
created_at
updated_at
```

Status:

```text
PENDING
POSTED
REMOVED
```

Sign convention:

```text
negative = money out
positive = money in
```

Pending→posted reconciliation must avoid duplicate logical spending.

## Provider Events

Used for provider-event idempotency.

Conceptually:

```text
provider
provider_event_id
event_type
connection_id
received_at
processed_at
status
error_message
```

Important uniqueness:

```text
UNIQUE(provider, provider_event_id)
```

where provider guarantees make this valid.

## Recurring Activity

Conceptual fields:

```text
id
user_id
account_id
merchant_name
description
estimated_amount_minor
currency
frequency
next_expected_date
confidence
source
status
```

---

# 15. Banking Architecture

Normal frontend requests read Vantage-Point state rather than directly requesting Teller/Plaid.

```text
Teller / Plaid
      ↓
Controlled Sync
      ↓
Provider Adapter
      ↓
Normalized Database
      ↓
API
      ↓
Frontend
```

Provider adapters isolate Teller/Plaid structures.

Bank synchronization occurs asynchronously.

---

# 16. Banking Worker

Responsible for:

- Fetching provider data
- Normalization
- Account synchronization
- Balance updates
- Transaction synchronization
- Pending→posted reconciliation
- Recurring activity
- Provider event handling
- Retry behavior
- Synchronization status
- Idempotency

Duplicate sync requests should be coalesced where practical.

---

# 17. Banking Failure

Provider failure should not erase synchronized data.

Instead:

```text
Existing state remains available
+
freshness/error information shown
```

Teller/Plaid failure must not disable unrelated Receipt functionality.

---

# 18. Receipt Data Model

Primary Receipt tables:

```text
receipts
receipt_items
receipt_participants
split_sessions
allocations
participant_submissions
receipt_events
receivables
```

## Receipts

Conceptual fields:

```text
id
owner_user_id
merchant_name
receipt_date
currency
subtotal_minor
tax_minor
tip_minor
discount_minor
total_minor
status
revision
image_reference
processing_status
processing_error
tax_allocation_method
tip_allocation_method
finalized_at
created_at
updated_at
```

## Receipt Items

Conceptual fields:

```text
id
receipt_id
receipt_revision
description
quantity
unit_price_minor
total_price_minor
position
created_at
updated_at
```

## Participants

Conceptual fields:

```text
id
receipt_id
name
is_owner
status
created_at
updated_at
```

Participants are receipt-scoped identities.

They are not canonical users.

## Split Sessions

Conceptual fields:

```text
id
receipt_id
token_hash
status
created_at
expires_at
invalidated_at
```

Raw bearer tokens should not be unnecessarily persisted.

## Allocations

Conceptual fields:

```text
id
receipt_id
receipt_revision
receipt_item_id
participant_id
amount_minor
percentage_bps
status
source
created_at
updated_at
```

`amount_minor` is authoritative.

## Participant Submissions

Conceptual fields:

```text
id
receipt_id
receipt_revision
participant_id
submission_version
status
submitted_at
confirmed_at
created_at
updated_at
```

## Receipt Events

Append-oriented audit history.

Conceptual fields:

```text
id
receipt_id
receipt_revision
event_type
actor_type
actor_user_id
actor_participant_id
target_participant_id
old_value
new_value
reason
created_at
```

JSON may be used for audit before/after values but not as the authoritative receipt state.

## Receivables

Conceptual fields:

```text
id
receipt_id
owner_user_id
participant_id
subtotal_minor
tax_minor
tip_minor
discount_minor
amount_owed_minor
currency
status
created_at
paid_at
updated_at
```

MVP statuses:

```text
OWED
PAID
```

---

# 19. Receipt Revision Strategy

Receipt starts at:

```text
revision = 1
```

If a material correction is required after claiming begins:

```text
Reset
  ↓
revision + 1
  ↓
incompatible old claims/submissions invalidated
```

History is preserved.

Finalized receipts are immutable through normal MVP mutation flows.

---

# 20. Allocation Concurrency

Critical invariant:

```text
SUM(active allocations for item)
<=
item.total_price_minor
```

Allocation mutation must be transactional.

Conceptually:

```text
BEGIN

lock/read relevant item state

calculate active allocation

validate requested amount

write allocation

COMMIT
```

Concurrent conflicting requests should result in one succeeding and another receiving a conflict.

---

# 21. Receipt Finalization

Finalization should execute atomically where practical.

```text
BEGIN

validate receipt state
validate allocations
validate confirmations
calculate tax/tip
calculate exact totals
create receivables
mark receipt FINALIZED
write audit event

COMMIT
```

Failure:

```text
ROLLBACK
```

There must not be a half-finalized receipt.

---

# 22. Receipt Vision

Flow:

```text
Upload
  ↓
Store private image
  ↓
Create receipt
  ↓
Queue vision job
  ↓
Vision Worker
  ↓
OCR / multimodal extraction
  ↓
Write proposal
  ↓
NEEDS_REVIEW
```

Vision output is advisory.

The owner establishes verified receipt truth.

---

# 23. Receipt Image Storage

Receipt images belong in private object storage rather than ordinary relational binary fields.

Initial direction:

```text
Supabase Storage
```

Database stores an image reference.

---

# 24. Receipt Realtime

Initial preferred direction:

```text
Supabase Realtime
```

Correct flow:

```text
Client
  ↓
Go API
  ↓
Validate
  ↓
PostgreSQL commit
  ↓
Realtime notification
  ↓
Other clients
```

Realtime is not financial authority.

Clients must always be capable of refetching authoritative state.

---

# 25. API

Initial API namespace:

```text
/api/v1
```

Example:

```text
GET  /api/v1/me

GET  /api/v1/accounts
GET  /api/v1/transactions

POST /api/v1/receipts
GET  /api/v1/receipts/{id}
POST /api/v1/receipts/{id}/finalize
```

Exact routes are implemented incrementally rather than creating the entire API before development begins.

---

# 26. Authentication Contracts

Owner APIs require a valid Supabase-authenticated session.

Participant APIs use temporary receipt-session authorization.

These are separate security mechanisms.

Knowing a receipt UUID is not authorization.

---

# 27. API Errors

Standard shape:

```json
{
  "error": {
    "code": "ALLOCATION_CONFLICT",
    "message": "The requested portion is no longer available.",
    "request_id": "..."
  }
}
```

Provider-specific internal errors should be translated into Vantage-Point semantics.

Sensitive provider details must not be returned publicly.

---

# 28. HTTP Conventions

Use appropriate status codes such as:

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

---

# 29. Pagination

Large collections use bounded pagination.

Transactions must not accidentally return an entire unlimited history.

Cursor pagination is preferred where appropriate.

---

# 30. Asynchronous Work

Asynchronous processing is appropriate for:

- Bank synchronization
- Provider webhook follow-up
- Receipt OCR/vision
- Potentially expensive recurring detection

Conceptual architecture:

```text
Producer
  ↓
Queue
  ↓
Bounded Worker
```

Exact queue technology is deferred.

---

# 31. Traffic Protection

Traffic protection is a core architecture requirement.

Use a combination of:

- Rate limiting
- Caching
- Request deduplication
- Queues
- Worker concurrency limits
- Request-size limits
- Pagination
- Timeouts
- Bounded retries
- Backoff
- Idempotency

No expensive external operation should scale without bounds directly from incoming HTTP traffic.

---

# 32. Rate Limiting

Conceptual layers:

```text
Internet
   ↓
Edge / Platform Protection
   ↓
Application Rate Limit
   ↓
Identity / Session Limit
   ↓
Domain Limit
   ↓
Expensive Operation Limit
```

Limits may be scoped by:

- User
- IP
- Receipt session
- Receipt
- Banking connection
- Provider
- Operation

Expensive operations receive stricter limits than ordinary cached reads.

Examples requiring strong protection:

- Bank connection
- Forced bank sync
- OCR upload/processing
- Receipt session creation
- Invalid receipt-token attempts
- Finalization attempts

Rate-limited requests return:

```text
429 Too Many Requests
```

Exact numeric thresholds should be determined through implementation/testing rather than guessed in advance.

---

# 33. Caching

Caching may occur at multiple levels:

```text
Browser / Client Cache
        ↓
Next.js
        ↓
Go API
        ↓
Optional Shared Cache
        ↓
PostgreSQL
```

Redis is not required for MVP merely because caching exists.

Introduce a shared cache when actual load/architecture justifies it.

## Cache Classification

Relatively stable:

- Profile
- Institution metadata

Moderately dynamic:

- Accounts
- Transactions
- Recurring activity
- Historical receipts

Highly dynamic:

- Active allocations
- Submission state
- Confirmation state

Critical financial mutations always validate authoritative database state.

Cached financial data must be user-scoped.

Example:

```text
dashboard:{user_id}
```

A cache must never allow one user's financial data to leak to another.

---

# 34. Backpressure

Expensive asynchronous work must use bounded concurrency.

```text
Incoming Work
     ↓
Queue
     ↓
Concurrency Limit
     ↓
Workers
     ↓
External Provider
```

A burst of 500 receipt uploads must not automatically create 500 simultaneous model requests.

Worker autoscaling must still have maximum bounds.

---

# 35. Retry Strategy

Retries must be:

- Bounded
- Idempotent
- Backed off
- Jittered where appropriate

Permanent failures stop retrying.

A failing provider must not create a retry storm.

---

# 36. Request Deduplication

Equivalent expensive work should be coalesced.

Example:

```text
Phone requests sync ───┐
Laptop requests sync ──┼──→ ONE ACTIVE SYNC
Worker requests sync ──┘
```

Similar protection should exist for duplicate receipt-processing work.

---

# 37. Graceful Degradation

Examples:

```text
Teller unavailable
→ show last synchronized state + freshness warning

Vision provider overloaded
→ receipt remains PROCESSING

Realtime unavailable
→ allow authoritative refresh

Cache unavailable
→ database fallback where safe
```

Cache is never the sole authoritative financial store.

---

# 38. Input Protection

Externally controlled inputs need bounds.

Examples:

- Request body size
- Receipt image size
- Image format
- String lengths
- Participant count
- Receipt item count
- Pagination limit
- Active session count

---

# 39. Security

Never expose to the browser:

- Supabase service credentials
- Database passwords
- Teller/Plaid private credentials
- Vision provider secrets

Never intentionally log:

- Authorization headers
- Authentication tokens
- Provider credentials
- Raw receipt bearer tokens

Use HTTPS in deployed environments.

Use appropriate CORS, secure-cookie, SameSite, and CSRF protections according to final session implementation.

---

# 40. Local Development

Required developer tools:

```text
Git
Node.js
pnpm
Go
Docker
Supabase CLI
```

Versions should be pinned by the repository where practical.

Package manager:

```text
pnpm
```

---

# 41. Local Runtime

Typical development:

```text
Next.js       → localhost:3000
Go API        → localhost:8080
Supabase      → local stack
Bank Worker   → local process
Receipt Worker→ local process
```

Exact ports/configuration should be standardized in repository setup.

---

# 42. Local Provider Strategy

Normal development should not require production providers.

Use:

```text
FakeBankProvider
FakeVisionProvider
```

with deterministic fixtures.

Possible explicit provider modes:

```text
fake
sandbox
live
```

`live` must never be the accidental default.

---

# 43. Development Fixtures

Synthetic fixtures may include:

```text
Fake checking account
Fake savings account
Fake credit card
Fake transactions
Fake recurring payments
Fake receipts
Fake participants
```

Real production financial data must not be copied into development fixtures.

---

# 44. Local Mobile Testing

Normal frontend development must test:

- Small mobile
- Large mobile
- Tablet
- Desktop

Important flows should occasionally be tested on a real mobile device.

Receipt collaboration should be testable with:

```text
Owner browser
+
Incognito/second browser
+
optional phone
```

---

# 45. Local Rate-Limit Testing

Application rate limiting must be testable locally.

Tests may configure small thresholds:

```text
limit = 3

request 1 → allowed
request 2 → allowed
request 3 → allowed
request 4 → 429
```

---

# 46. Local Cache Testing

Tests should verify:

- Hit
- Miss
- Expiration
- Invalidation
- User isolation
- Safe fallback

---

# 47. Testing Strategy

Unit tests should not require live internet providers.

Integration tests may use:

- Local PostgreSQL/Supabase
- Local API
- Fake providers

Important future E2E flows:

```text
Google login
→ Dashboard
```

and:

```text
Receipt
→ OCR
→ Review
→ Split
→ Submit
→ Confirm
→ Finalize
```

---

# 48. Financial Correctness Tests

Tests must explicitly cover:

- Integer money
- Tax allocation
- Tip allocation
- Deterministic rounding
- Receipt reconciliation
- Duplicate transactions
- Pending→posted handling
- Finalization
- Allocation concurrency

Example concurrency test:

```text
Item remaining = $10

Request A asks for $10
Request B asks for $10

Expected:
one succeeds
one conflicts
```

---

# 49. CI

CI platform:

```text
GitHub Actions
```

Normal flow:

```text
Task
  ↓
Feature Branch
  ↓
Pull Request
  ↓
CI
  ↓
Integration Review
  ↓
Merge
```

`main` should be protected.

---

# 50. Initial CI Checks

Frontend:

```text
install frozen dependencies
lint
typecheck
tests
production build
```

Backend:

```text
format check
go vet
go test
go build
```

Database:

```text
start clean test database
apply migrations from zero
validate schema
```

Security:

```text
basic secret scanning
dependency scanning as introduced
```

CI must use fake providers rather than production Teller/Plaid/model calls.

---

# 51. CI Expansion

As features are implemented, CI should add tests for:

- Authorization
- RLS
- Financial invariants
- Receipt concurrency
- Provider idempotency
- Vision idempotency
- Rate limiting
- Cache isolation
- Request deduplication
- Worker concurrency
- Retry behavior
- Mobile viewport E2E flows

---

# 52. Deployment Direction

Initial direction:

```text
Next.js
   → Vercel

Go API
   → Google Cloud Run

Workers
   → Google Cloud Run-compatible deployment

Database/Auth/Realtime/Storage
   → Supabase
```

Initial CI does not automatically deploy production.

CD comes later after manual deployment is understood.

---

# 53. Environments

Eventually:

```text
local
staging
production
```

Potential frontend preview environments may also exist.

Production secrets/data must remain isolated from local development and ordinary PR CI.

---

# 54. Architecture Non-Goals

MVP does not require:

- Kubernetes
- Kafka
- Service mesh
- Multiple application databases
- Dedicated API Gateway service
- Event sourcing
- CQRS
- Dedicated user microservice
- Dedicated ledger service
- Separate repository per feature
- Redis without a demonstrated need

---

# 55. Future Ledger

A dedicated ledger may become valuable when Vantage-Point adds:

- Safe-to-spend
- Forecasting
- Obligations
- Cash-flow projection
- Automated reimbursement reconciliation

Do not build it for MVP 1.

---

# 56. Deferred Technical Decisions

Select when implementation requires them:

- Exact Go HTTP framework
- Exact Go database/query library
- Exact queue technology
- Whether Redis is needed
- Exact vision provider
- Exact realtime channel implementation
- Exact image signed-URL strategy
- Exact Teller/Plaid provider identifier constraints
- Exact receipt-session expiration
- Exact rounding remainder ordering
- Exact indexes after real query patterns exist
- Exact observability provider
- Exact production rate-limit values
- Exact caching TTLs

These decisions may not silently change approved product semantics.