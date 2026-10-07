# Vantage-Point Database Design v1

## 1. Purpose

This document defines the initial logical database model for Vantage-Point MVP 1.

The database must support:

- Canonical Vantage-Point users
- Banking connections
- Financial accounts
- Transactions
- Recurring financial activity
- Receipt upload and processing
- Receipt items
- Temporary receipt participants
- Receipt splitting sessions
- Item allocations
- Participant submissions
- Owner corrections
- Reconfirmation
- Audit history
- Finalized receipt totals
- Receivables
- Manual payment tracking

This document defines the **logical data model**.

It does not yet define final SQL migrations.

---

# 2. Database Technology

Vantage-Point will use:

```text
PostgreSQL
    ↓
Supabase
```

PostgreSQL will be the primary authoritative application database.

Supabase may additionally provide:

- Authentication
- Row-Level Security
- Realtime
- Database management
- Storage integration

---

# 3. Database Organization

For MVP 1, Vantage-Point should use **one PostgreSQL database**.

The application should not create separate physical databases for Banking and Receipts.

Conceptually:

```text
Vantage-Point PostgreSQL
│
├── Foundation
│   └── profiles
│
├── Banking
│   ├── institution_connections
│   ├── accounts
│   ├── transactions
│   ├── provider_events
│   └── recurring_activity
│
└── Receipts
    ├── receipts
    ├── receipt_items
    ├── receipt_participants
    ├── split_sessions
    ├── allocations
    ├── participant_submissions
    ├── receipt_events
    └── receivables
```

The Dashboard does not initially require its own authoritative tables.

It reads information from Foundation, Banking, and Receipts.

---

# 4. Schema Strategy

Vantage-Point may initially use the normal Supabase application schema while maintaining domain ownership through naming and migrations.

A future structure may use logical PostgreSQL schemas such as:

```text
auth.*
shared.*
banking.*
receipts.*
```

However, physical schema separation is not required before the first implementation.

The important requirement is **clear domain ownership**.

---

# 5. Primary Key Strategy

Application-owned tables should use globally unique identifiers.

Recommended initial strategy:

```text
UUID
```

Example:

```text
receipt.id
account.id
transaction.id
```

External provider identifiers must never replace internal primary keys.

Provider IDs should be stored separately.

Example:

```text
accounts.id
                = internal Vantage-Point UUID

accounts.provider_account_id
                = Teller/Plaid identifier
```

---

# 6. Timestamp Strategy

Application tables should generally contain:

```text
created_at
updated_at
```

using timezone-aware PostgreSQL timestamps.

Recommended:

```text
TIMESTAMPTZ
```

Operational records may additionally contain domain-specific timestamps such as:

```text
last_synced_at
submitted_at
finalized_at
paid_at
expires_at
```

---

# 7. Money Representation

Vantage-Point must never use floating-point values for authoritative monetary amounts.

Recommended MVP representation:

```text
BIGINT
```

containing currency minor units.

Example:

```text
$42.19
→
4219
```

Field naming should make this clear.

Example:

```text
amount_minor
total_minor
tax_minor
tip_minor
```

Currency should be stored separately.

Example:

```text
currency = "USD"
```

For MVP 1, USD may be the only officially supported currency, but the schema should not unnecessarily prevent future currencies.

---

# 8. Percentage Representation

Receipt allocation percentages must also avoid floating-point uncertainty.

Recommended representation:

```text
basis points
```

where:

```text
10000 = 100%
5000  = 50%
2500  = 25%
```

Example field:

```text
percentage_bps
```

Alternatively, allocation amounts may be the canonical financial representation.

The final allocation implementation can use both:

```text
percentage_bps
amount_minor
```

with server-side validation.

The authoritative final financial amount must always resolve to an exact monetary value.

---

# 9. Canonical User Identity

Supabase Auth provides the canonical authentication identity.

Application-owned profile data will reference the Supabase Auth user ID.

Conceptually:

```text
auth.users
    │
    │ id
    ▼
profiles
```

No Banking or Receipt domain may create a separate application-user table.

---

# 10. profiles

## Purpose

Stores application-level information for authenticated Vantage-Point users.

Authentication secrets remain in Supabase Auth.

## Conceptual Structure

```text
profiles

id
display_name
email
avatar_url
created_at
updated_at
```

## Fields

### id

References:

```text
auth.users.id
```

This is the canonical Vantage-Point user ID.

### display_name

Display name obtained from Google authentication where available.

### email

Authenticated Google email address.

This does not replace Supabase authentication state.

### avatar_url

Optional Google profile image.

### created_at

Profile creation timestamp.

### updated_at

Last profile update timestamp.

---

# 11. Profile Constraints

Each authenticated Supabase user may have at most one profile.

Conceptually:

```text
profiles.id
PRIMARY KEY
REFERENCES auth.users(id)
```

Profile creation must be idempotent.

Repeated login must not create multiple profile records.

---

# 12. Banking Domain Overview

Banking relationships:

```text
profiles
    │
    ▼
institution_connections
    │
    ▼
accounts
    │
    ▼
transactions
```

Additional supporting data:

```text
institution_connections
       │
       └── provider_events

accounts / transactions
       │
       └── recurring_activity
```

---

# 13. institution_connections

## Purpose

Represents a user's connection to a financial institution through Teller or Plaid.

One connection may provide multiple accounts.

## Conceptual Structure

```text
institution_connections

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

---

# 14. institution_connections Fields

### id

Internal Vantage-Point UUID.

### user_id

References:

```text
profiles.id
```

The authenticated owner.

### provider

Expected values:

```text
TELLER
PLAID
```

Additional providers may be added later.

### provider_connection_id

Provider-specific connection identifier.

### institution_name

Human-readable institution name.

Example:

```text
Chase
Bank of America
Wells Fargo
```

### institution_provider_id

Optional provider-specific institution identifier.

### status

Conceptual values:

```text
CONNECTING
ACTIVE
SYNCING
REAUTH_REQUIRED
ERROR
DISCONNECTED
```

### credential_reference

Reference to the protected provider credential material.

The preferred architecture should avoid casually storing raw credentials alongside normal application data.

Exact secret-storage implementation will be determined separately.

### last_successful_sync_at

Timestamp of last successful provider synchronization.

### last_sync_attempt_at

Timestamp of last synchronization attempt.

### created_at

Creation timestamp.

### updated_at

Last update timestamp.

---

# 15. institution_connections Constraints

Every connection belongs to one Vantage-Point user.

Provider connections should have sufficient uniqueness constraints to prevent accidental duplicate ingestion.

Potential constraint:

```text
UNIQUE(provider, provider_connection_id)
```

if provider semantics guarantee global uniqueness.

Otherwise:

```text
UNIQUE(user_id, provider, provider_connection_id)
```

Final constraint should follow Teller/Plaid identifier guarantees.

---

# 16. accounts

## Purpose

Represents financial accounts returned by banking providers.

Examples:

- Checking
- Savings
- Credit card

## Conceptual Structure

```text
accounts

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

---

# 17. accounts Fields

### id

Internal UUID.

### institution_connection_id

References:

```text
institution_connections.id
```

### provider_account_id

Identifier used by Teller/Plaid.

### name

User-facing account name.

Example:

```text
Everyday Checking
```

### official_name

Optional official account title provided by the institution.

### account_type

Normalized Vantage-Point account type.

Initial possible values:

```text
CHECKING
SAVINGS
CREDIT_CARD
OTHER
```

### account_subtype

Optional provider-derived subtype.

### mask

Optional safe account-number suffix.

Example:

```text
1234
```

Never store full account numbers unless explicitly required and properly secured.

### currency

Example:

```text
USD
```

### current_balance_minor

Most recently known current balance.

### available_balance_minor

Most recently known available balance where provided.

May be nullable.

### status

Potential values:

```text
ACTIVE
UNAVAILABLE
CLOSED
```

### last_synced_at

Latest successful account-data synchronization time.

### created_at

Creation timestamp.

### updated_at

Update timestamp.

---

# 18. Account Uniqueness

Within one institution connection:

```text
UNIQUE(
    institution_connection_id,
    provider_account_id
)
```

should normally prevent duplicate provider accounts.

Provider IDs must not be used as internal application IDs.

---

# 19. transactions

## Purpose

Stores normalized banking transactions imported from Teller or Plaid.

## Conceptual Structure

```text
transactions

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

---

# 20. Transaction Amount Convention

Vantage-Point should adopt one consistent internal sign convention.

Recommended convention:

```text
negative = money leaving the account
positive = money entering the account
```

Examples:

```text
Coffee
-525
= -$5.25

Payroll
+175000
= +$1,750.00
```

Provider adapters must translate Teller/Plaid semantics into this Vantage-Point convention.

This prevents provider-specific sign differences from spreading through the application.

---

# 21. transaction Fields

### id

Internal UUID.

### account_id

References:

```text
accounts.id
```

### provider

Source provider.

### provider_transaction_id

Provider's transaction identifier.

### pending_provider_transaction_id

Optional identifier linking posted activity with earlier pending activity where provider support exists.

### status

Expected values:

```text
PENDING
POSTED
REMOVED
```

Exact values may be refined based on provider behavior.

### amount_minor

Transaction amount using the Vantage-Point sign convention.

### currency

Example:

```text
USD
```

### description

Provider-normalized transaction description.

### merchant_name

Merchant if confidently known.

### transaction_date

Primary transaction date.

### authorized_at

Authorization timestamp if available.

### posted_at

Posting timestamp if available.

### category

Normalized Vantage-Point category where available.

### raw_category

Optional provider category/reference information.

### created_at

Database record creation timestamp.

### updated_at

Last normalization/update timestamp.

---

# 22. Transaction Uniqueness

For provider-sourced transactions, duplicate ingestion must be prevented.

Potential:

```text
UNIQUE(provider, provider_transaction_id)
```

or account-scoped equivalent depending on provider guarantees.

Synchronization must use provider IDs and idempotent upserts rather than blindly creating rows on every sync.

---

# 23. Pending-to-Posted Transactions

Pending and posted records must not create obvious duplicated spending.

The data model should support:

```text
Pending Transaction
        ↓
Provider reconciliation
        ↓
Posted Transaction
```

Where possible, provider identifiers should establish the relationship.

Historical information may remain available internally, but current financial presentation must not double-count both forms.

---

# 24. provider_events

## Purpose

Tracks provider webhooks/events and prevents duplicate event processing.

## Conceptual Structure

```text
provider_events

id
provider
provider_event_id
event_type
connection_id
received_at
processed_at
status
error_message
created_at
```

---

# 25. Provider Event Idempotency

Recommended constraint:

```text
UNIQUE(provider, provider_event_id)
```

This allows the same webhook to be safely delivered more than once.

Possible statuses:

```text
RECEIVED
PROCESSING
PROCESSED
FAILED
```

---

# 26. recurring_activity

## Purpose

Stores detected or provider-supplied recurring financial activity.

This is not necessarily guaranteed truth.

## Conceptual Structure

```text
recurring_activity

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
created_at
updated_at
```

---

# 27. recurring_activity Fields

### user_id

Allows efficient user-level recurring queries.

### account_id

Optional relationship to source financial account.

### merchant_name

Normalized recurring merchant.

### estimated_amount_minor

Expected recurring amount when meaningful.

### frequency

Conceptual values:

```text
WEEKLY
MONTHLY
QUARTERLY
ANNUAL
IRREGULAR
UNKNOWN
```

### next_expected_date

Optional prediction.

### confidence

Optional model/provider confidence.

### source

Example:

```text
TELLER
PLAID
VANTAGE_POINT
```

### status

Example:

```text
ACTIVE
INACTIVE
UNCERTAIN
```

---

# 28. Receipt Domain Overview

Receipt relationships:

```text
profiles
    │
    ▼
receipts
    │
    ├── receipt_items
    │
    ├── receipt_participants
    │
    ├── split_sessions
    │
    ├── allocations
    │
    ├── participant_submissions
    │
    ├── receipt_events
    │
    └── receivables
```

---

# 29. receipts

## Purpose

Represents the complete owner-controlled receipt.

## Conceptual Structure

```text
receipts

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

---

# 30. receipts Fields

### id

Internal UUID.

### owner_user_id

References:

```text
profiles.id
```

### merchant_name

Verified merchant name.

### receipt_date

Receipt date where known.

### currency

For MVP:

```text
USD
```

### subtotal_minor

Verified subtotal.

### tax_minor

Verified tax.

### tip_minor

Verified tip.

### discount_minor

Verified discount where applicable.

Recommended convention:

```text
positive value representing total discount magnitude
```

Calculation rules will determine how it is applied.

### total_minor

Verified final receipt total.

### status

Receipt lifecycle state.

### revision

Integer revision number.

Example:

```text
1
2
3
```

Incremented after material reset/revision where required.

### image_reference

Private storage reference for original receipt image.

### processing_status

Vision/OCR job status if useful separately from receipt lifecycle.

### processing_error

Safe processing failure information.

### tax_allocation_method

Expected:

```text
PROPORTIONAL
EVEN
```

### tip_allocation_method

Expected:

```text
PROPORTIONAL
EVEN
```

### finalized_at

Timestamp when authoritative split was finalized.

### created_at

Creation timestamp.

### updated_at

Update timestamp.

---

# 31. Receipt Status

Recommended explicit receipt states:

```text
DRAFT
PROCESSING
NEEDS_REVIEW
READY_TO_SHARE
CLAIMING
AWAITING_CONFIRMATION
RECONFIRMATION_REQUIRED
FINALIZED
```

Additional failure/internal states may be added when justified.

State transitions must be validated by Receipt business logic.

---

# 32. receipt_items

## Purpose

Stores verified receipt line items.

## Conceptual Structure

```text
receipt_items

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

---

# 33. receipt_items Fields

### id

Internal UUID.

### receipt_id

References:

```text
receipts.id
```

### receipt_revision

Identifies which receipt revision the item belongs to.

### description

Verified line-item description.

### quantity

Exact quantity.

Integer quantities are sufficient for many receipts.

If fractional quantities are later needed, this can be revisited.

### unit_price_minor

Unit amount where meaningful.

### total_price_minor

Authoritative item-line total.

### position

Maintains original/display ordering.

### created_at

Creation timestamp.

### updated_at

Update timestamp.

---

# 34. Receipt Item Integrity

Receipt item totals must not be silently changed after active claiming begins.

Material changes after claiming should require:

```text
Receipt Reset / Revision
```

This prevents claims from referencing changing financial data.

---

# 35. receipt_participants

## Purpose

Represents people participating in one receipt split.

These are not Vantage-Point user accounts.

## Conceptual Structure

```text
receipt_participants

id
receipt_id
name
is_owner
status
created_at
updated_at
```

---

# 36. receipt_participants Fields

### id

Internal participant UUID.

### receipt_id

References:

```text
receipts.id
```

### name

Participant display name.

Example:

```text
Alex
Chris
Ryan
```

### is_owner

Whether this participant represents the receipt owner in the split.

This does not replace authenticated ownership.

### status

Conceptual:

```text
INVITED
ACTIVE
SUBMITTED
RECONFIRMATION_REQUIRED
CONFIRMED
```

### created_at

Creation timestamp.

### updated_at

Update timestamp.

---

# 37. Participant Identity Constraint

Participant identity is receipt-scoped.

Conceptually:

```text
receipt_participants.id
```

is the actual internal identity.

Names are presentation labels.

A uniqueness rule may initially prevent identical participant names within the same receipt:

```text
UNIQUE(receipt_id, normalized_name)
```

if that provides the best UX.

This should be reviewed before implementation because real groups may contain duplicate first names.

---

# 38. split_sessions

## Purpose

Controls temporary participant access to a receipt.

## Conceptual Structure

```text
split_sessions

id
receipt_id
token_hash
status
created_at
expires_at
invalidated_at
```

---

# 39. split_sessions Fields

### id

Internal UUID.

### receipt_id

References:

```text
receipts.id
```

### token_hash

Secure hash of the bearer token.

The raw token should not normally be stored.

### status

Possible states:

```text
ACTIVE
EXPIRED
INVALIDATED
COMPLETED
```

### created_at

Session creation time.

### expires_at

Expiration time.

### invalidated_at

Timestamp if owner invalidates/replaces the session.

---

# 40. Session Token Rules

Raw tokens should be generated securely.

Conceptually:

```text
Raw random token
        ↓
sent to participant
        ↓
hash stored in DB
```

When used:

```text
presented token
        ↓
hash
        ↓
compare to token_hash
```

The implementation should not use predictable receipt IDs as authorization tokens.

---

# 41. allocations

## Purpose

Stores participant ownership of portions of receipt items.

This table is central to receipt splitting.

## Conceptual Structure

```text
allocations

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

---

# 42. Allocation Canonical Model

Recommended design:

```text
amount_minor
```

is the authoritative financial value.

`percentage_bps` may be stored to preserve the user's intended split where relevant.

Example:

```text
Pizza = $24.00

Ryan
percentage_bps = 5000
amount_minor   = 1200

Alex
percentage_bps = 2500
amount_minor   = 600

Chris
percentage_bps = 2500
amount_minor   = 600
```

Final monetary totals are calculated from exact amounts.

---

# 43. Allocation Fields

### receipt_id

Redundant with item relationship but useful for validation/querying.

### receipt_revision

Ensures allocations apply to the correct receipt version.

### receipt_item_id

References:

```text
receipt_items.id
```

### participant_id

References:

```text
receipt_participants.id
```

### amount_minor

Exact amount of item allocated.

### percentage_bps

Optional exact split percentage.

### status

Potential:

```text
DRAFT
SUBMITTED
CONFIRMED
INVALIDATED
```

### source

Who caused the allocation:

```text
PARTICIPANT
OWNER
SYSTEM
```

---

# 44. Allocation Constraints

Critical financial invariant:

```text
SUM(active amount_minor)
for one receipt_item
<= receipt_item.total_price_minor
```

A finalized receipt should normally require:

```text
SUM(final allocations)
= receipt_item.total_price_minor
```

for every required item.

Concurrency must be enforced transactionally.

Client-side validation is insufficient.

---

# 45. Allocation Uniqueness

A participant should normally have at most one active allocation per item.

Potential:

```text
UNIQUE(
    receipt_item_id,
    participant_id,
    active_revision
)
```

Exact implementation may use partial indexes or application constraints.

---

# 46. participant_submissions

## Purpose

Preserves explicit participant submission and confirmation state.

Claiming an item is not the same thing as submitting.

## Conceptual Structure

```text
participant_submissions

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

---

# 47. Submission Status

Potential:

```text
DRAFT
SUBMITTED
RECONFIRMATION_REQUIRED
CONFIRMED
INVALIDATED
```

A participant submission must remain historically understandable even when an owner later makes corrections.

---

# 48. Submission Versioning

When an owner changes a participant's financial allocation after submission:

```text
submission_version
```

may increase.

Example:

```text
Participant submitted version 1

Owner correction

Participant required to confirm version 2
```

This gives clear semantic history without silently replacing the earlier submission.

---

# 49. receipt_events

## Purpose

Provides immutable or append-oriented audit history for important receipt changes.

## Conceptual Structure

```text
receipt_events

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

---

# 50. receipt_events Event Types

Examples:

```text
RECEIPT_CREATED
OCR_COMPLETED
OWNER_REVIEW_COMPLETED
SESSION_STARTED
PARTICIPANT_JOINED
PARTICIPANT_SUBMITTED
OWNER_ALLOCATION_CHANGED
RECONFIRMATION_REQUIRED
PARTICIPANT_CONFIRMED
RECEIPT_RESET
RECEIPT_FINALIZED
RECEIVABLE_MARKED_PAID
```

This list may expand.

---

# 51. Audit Actor Model

Actor types:

```text
OWNER
PARTICIPANT
SYSTEM
```

When actor type is `OWNER`:

```text
actor_user_id
```

may reference the authenticated profile.

When actor type is `PARTICIPANT`:

```text
actor_participant_id
```

identifies the receipt participant.

When actor type is `SYSTEM`, neither may be required.

---

# 52. Audit Values

`old_value` and `new_value` may use structured JSON for flexible event history.

Example:

```json
{
  "amount_minor": 1200,
  "percentage_bps": 5000
}
```

Audit JSON must not become the primary source of current application state.

It is historical context.

---

# 53. Receipt Final Totals

The final receipt split should be deterministically derivable from:

```text
final allocations
+
tax allocation
+
tip allocation
+
discount handling
```

However, once finalized, Vantage-Point should preserve final participant financial obligations so they cannot silently drift if calculation code changes later.

This is handled through receivables.

---

# 54. receivables

## Purpose

Stores finalized amounts owed to the receipt owner.

## Conceptual Structure

```text
receivables

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

---

# 55. receivables Fields

### id

Internal UUID.

### receipt_id

Finalized receipt.

### owner_user_id

Authenticated user who is owed the money.

Stored explicitly for efficient ownership queries and integrity.

### participant_id

Participant who owes the amount.

### subtotal_minor

Participant's allocated item subtotal.

### tax_minor

Allocated tax.

### tip_minor

Allocated tip.

### discount_minor

Allocated discount if supported.

### amount_owed_minor

Final authoritative reimbursement amount.

### currency

Example:

```text
USD
```

### status

Initial MVP:

```text
OWED
PAID
```

### created_at

Creation at finalization.

### paid_at

Timestamp when manually marked paid.

### updated_at

Last update.

---

# 56. Receivable Uniqueness

For MVP, each participant should normally receive one receivable per finalized receipt.

Potential:

```text
UNIQUE(receipt_id, participant_id)
```

assuming receipt finalization does not create multiple simultaneous payment obligations for one participant.

---

# 57. Owner as Participant

If the receipt owner participates in the meal, they may appear in:

```text
receipt_participants
```

However, Vantage-Point should not normally create a receivable where the owner owes themselves.

Owner allocations still matter for ensuring the entire receipt is accounted for.

---

# 58. Tax and Tip Allocation

Tax and tip do not necessarily need separate persistent allocation tables in MVP.

They can be deterministically calculated during finalization using:

```text
tax_allocation_method
tip_allocation_method
```

and final participant subtotals.

Final allocated values should then be frozen inside:

```text
receivables.tax_minor
receivables.tip_minor
```

This keeps the MVP model simpler.

If requirements later demand interactive tax/tip allocation editing per person, dedicated tables can be added.

---

# 59. Rounding Strategy

Financial rounding must be deterministic.

Recommended method:

1. Calculate each participant's exact proportional share.
2. Round down to whole minor units.
3. Determine remaining minor units.
4. Distribute remaining units according to a deterministic rule.

Example:

```text
$10.00 / 3

Participant A = $3.34
Participant B = $3.33
Participant C = $3.33

Total = $10.00
```

The deterministic remainder rule should be documented in implementation contracts.

No cents may disappear or be created.

---

# 60. Receipt Revisions

Receipts should begin:

```text
revision = 1
```

If a material reset occurs after claiming:

```text
revision = revision + 1
```

New items and allocations belong to the new revision.

Historical events retain the older revision reference.

This allows the application to understand:

```text
Claims against Receipt v1
```

versus:

```text
Current Receipt v2
```

---

# 61. Reset Behavior

A reset should not physically erase historical claims simply because they are no longer active.

Possible approach:

```text
allocations.status = INVALIDATED
participant_submissions.status = INVALIDATED
```

New revision records then become authoritative.

This preserves auditability.

---

# 62. Finalized Receipt Immutability

After:

```text
receipts.status = FINALIZED
```

normal receipt mutation must stop.

Forbidden normal operations include:

- Editing item prices
- Adding/removing items
- Changing allocations
- Changing tax/tip methodology
- Changing participant final obligations

Future corrections must use an explicit correction workflow.

MVP may simply disallow post-finalization corrections.

---

# 63. Dashboard Data Model

The Dashboard does not need its own authoritative tables for MVP.

It queries/aggregates:

```text
profiles
institution_connections
accounts
transactions
recurring_activity
receipts
receivables
```

Example:

```text
Connected Cash
=
eligible checking/savings account balances
```

Example:

```text
Outstanding Reimbursements
=
SUM(receivables.amount_owed_minor
WHERE status = 'OWED')
```

---

# 64. Dashboard Caching

Persistent dashboard summary tables should not be created initially.

If performance later requires it, Vantage-Point may introduce:

- Materialized views
- Cache entries
- Derived summary tables

only after profiling shows a need.

---

# 65. Ownership Model

Canonical relationships:

```text
profiles
│
├── institution_connections
│       │
│       └── accounts
│               │
│               └── transactions
│
├── recurring_activity
│
└── receipts
        │
        ├── receipt_items
        ├── receipt_participants
        ├── split_sessions
        ├── allocations
        ├── participant_submissions
        ├── receipt_events
        └── receivables
```

---

# 66. Deletion Strategy

Financial and receipt records should not rely heavily on cascading physical deletion without an explicit retention policy.

For MVP:

- Normal application operations should not physically delete finalized financial history.
- Temporary records may be invalidated rather than destroyed when auditability matters.
- Full account deletion behavior remains deferred.

Foreign keys should therefore use deletion behavior deliberately rather than defaulting everything to:

```text
ON DELETE CASCADE
```

---

# 67. Row-Level Security

RLS should protect user-owned data where practical.

Example Foundation rule:

```text
profiles.id = auth.uid()
```

Example receipt-owner rule:

```text
receipts.owner_user_id = auth.uid()
```

Example institution rule:

```text
institution_connections.user_id = auth.uid()
```

However, temporary receipt participants cannot use normal authenticated-user RLS because they are not Supabase users.

Participant access should pass through controlled backend/session authorization.

---

# 68. Participant Access Model

Temporary participants should not receive broad direct database access.

Recommended model:

```text
Participant Browser
        ↓
Temporary Receipt Token
        ↓
Receipt Backend/API
        ↓
Validate Session
        ↓
Perform Authorized Operation
        ↓
PostgreSQL
```

Realtime subscriptions must similarly be restricted to the relevant receipt/session.

---

# 69. Backend Service Access

Backend workers may require elevated database privileges.

Examples:

- Banking synchronization worker
- Receipt vision worker
- Provider webhook processor

These privileges must remain backend-only.

Supabase service credentials must never be shipped to the browser.

---

# 70. Indexing Strategy

Initial indexes should support primary access patterns.

Potential indexes include:

```text
institution_connections(user_id)

accounts(institution_connection_id)

transactions(account_id, transaction_date)

transactions(provider, provider_transaction_id)

recurring_activity(user_id)

receipts(owner_user_id, created_at)

receipt_items(receipt_id)

receipt_participants(receipt_id)

split_sessions(receipt_id)

allocations(receipt_item_id)

allocations(participant_id)

participant_submissions(receipt_id, participant_id)

receipt_events(receipt_id, created_at)

receivables(owner_user_id, status)

receivables(receipt_id)
```

Indexes should not be added blindly.

Final migrations should reflect actual query patterns.

---

# 71. Provider Raw Data

Vantage-Point should avoid requiring normal product code to read provider-specific JSON.

If useful for debugging or future normalization, limited provider source metadata may be retained.

Potential pattern:

```text
provider_metadata JSONB
```

However:

- It must not contain credentials.
- It should not replace normalized columns.
- It should not become the application's primary API.
- Sensitive provider payloads should not be retained without reason.

---

# 72. JSON Usage

JSONB is appropriate for:

- Audit before/after values
- Limited provider metadata
- Processing metadata

JSONB should not be used to avoid designing core relational structures.

Bad:

```text
receipt_data JSONB
```

containing the entire authoritative receipt application model.

Preferred:

```text
receipts
receipt_items
allocations
receivables
```

with explicit relationships.

---

# 73. Database Migrations

All schema changes must occur through version-controlled migrations.

Agents must not manually alter shared production/staging schemas without a migration.

Rules:

1. Create new migration.
2. Apply migration in development.
3. Validate migration.
4. Run tests.
5. Submit for review.
6. Never silently rewrite an already-applied migration.

---

# 74. Domain Migration Ownership

Foundation Agent owns migrations involving:

```text
profiles
shared identity structures
```

Banking Agent owns:

```text
institution_connections
accounts
transactions
provider_events
recurring_activity
```

Receipt Agent owns:

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

Cross-domain migrations require Integration Review.

---

# 75. Database Agent Policy

Vantage-Point does **not** require a dedicated Database Agent.

Database work belongs to the agent that owns the corresponding product domain.

Example:

```text
Banking Agent
    ↓
owns banking migrations

Receipt Agent
    ↓
owns receipt migrations
```

This keeps database ownership aligned with feature ownership.

---

# 76. Cross-Domain Foreign Keys

Approved primary cross-domain relationships include:

```text
institution_connections.user_id
    → profiles.id

receipts.owner_user_id
    → profiles.id

receivables.owner_user_id
    → profiles.id
```

Domain agents must not introduce additional cross-domain relationships casually.

New cross-domain dependencies require review.

---

# 77. Database Constraints vs Application Logic

Important invariants should be enforced as close to the database as practical.

Examples:

Database should enforce:

- Foreign keys
- Unique provider IDs
- Nonnegative values where appropriate
- Valid relationships
- Referential integrity

Application/domain transaction logic should enforce:

- Receipt state transitions
- Concurrent allocation limits
- Reconfirmation workflows
- Provider synchronization workflows
- Finalization requirements

Critical financial invariants should not depend solely on frontend validation.

---

# 78. Concurrency

Receipt allocation operations must execute atomically.

Conceptually:

```text
BEGIN

Lock / validate relevant receipt item

Calculate active allocation total

Verify requested allocation fits

Insert/update allocation

COMMIT
```

Two simultaneous clients must not both successfully consume the same remaining portion.

---

# 79. Idempotency

Database design must support idempotent external operations.

Examples:

```text
Provider webhook
    ↓ duplicate delivery
processed once

Bank transaction sync
    ↓ same transaction returned again
upsert rather than duplicate

Vision job
    ↓ retry
does not create duplicate receipt items
```

Stable external identifiers and unique constraints should be used wherever available.

---

# 80. Security-Sensitive Data

The following must receive special handling:

- Teller credentials
- Plaid credentials
- Supabase service credentials
- Temporary receipt session tokens
- Receipt images
- User financial data

These should never be exposed through generic database APIs without explicit authorization.

---

# 81. Data That Should Not Be Stored

Vantage-Point should not intentionally store:

- Google passwords
- User banking passwords
- Full authentication tokens in ordinary logs
- Plaintext temporary session tokens unnecessarily
- Teller/Plaid secrets in frontend-accessible tables
- Entire bank account numbers unless explicitly required
- Credit-card numbers

---

# 82. Database Error Handling

Database operations should fail explicitly when financial invariants cannot be preserved.

Examples:

```text
Allocation exceeds item total
→ reject transaction

Unknown receipt owner
→ reject creation

Duplicate provider transaction
→ idempotent update / conflict handling

Expired session
→ participant operation rejected
```

The application must not attempt to "fix" corrupted financial state silently.

---

# 83. Initial ER Model

High-level relationship map:

```text
                         auth.users
                             │
                             │ 1:1
                             ▼
                          profiles
                             │
              ┌──────────────┴──────────────┐
              │                             │
              ▼                             ▼
   institution_connections               receipts
              │                             │
              │ 1:N                         ├──────────────┐
              ▼                             │              │
           accounts                         ▼              ▼
              │                      receipt_items   receipt_participants
              │ 1:N                         │              │
              ▼                             │              │
         transactions                       └──────┬───────┘
                                                  │
                                                  ▼
                                             allocations

receipts
   │
   ├── split_sessions
   │
   ├── participant_submissions
   │
   ├── receipt_events
   │
   └── receivables
```

Banking supporting relationship:

```text
institution_connections
        │
        └── provider_events

profiles/accounts
        │
        └── recurring_activity
```

---

# 84. Initial Table Set

The MVP database should begin with approximately these application tables:

## Foundation

```text
profiles
```

## Banking

```text
institution_connections
accounts
transactions
provider_events
recurring_activity
```

## Receipts

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

Total:

```text
14 application tables
```

plus Supabase-managed authentication tables.

This number may change slightly during migration design.

---

# 85. Explicitly Deferred Tables

Do not create tables yet for:

```text
budgets
safe_to_spend
forecast_snapshots
investment_holdings
reward_programs
credit_cards_catalog
card_recommendations
payment_matches
venmo_connections
zelle_connections
financial_goals
notifications
admin_users
```

These belong to future requirements.

---

# 86. Database Design Principles

Vantage-Point database development should follow these principles:

1. One canonical user identity.
2. One PostgreSQL database for MVP.
3. Domain ownership rather than database-per-feature.
4. Exact financial values.
5. Internal IDs separated from provider IDs.
6. Database constraints protect important invariants.
7. External ingestion is idempotent.
8. Receipt financial history is auditable.
9. Finalized financial state does not silently mutate.
10. Temporary participants remain separate from authenticated users.
11. Secrets remain backend-only.
12. Derived Dashboard state does not become a competing source of truth.
13. Migrations are immutable after application.
14. Cross-domain database changes require review.
15. Avoid premature tables for post-MVP functionality.

---

# 87. Decisions Locked by This Design

If approved, the following become Vantage-Point architecture decisions:

```text
Database:
PostgreSQL / Supabase

Physical DB count:
One for MVP

Canonical user:
Supabase auth.users.id

Application profile:
profiles

Internal IDs:
UUID

Money:
integer minor units

Transaction sign:
negative = money out
positive = money in

Primary banking provider:
Teller

Secondary banking provider:
Plaid

Provider normalization:
required

Receipt participants:
receipt-scoped identities

Receipt access:
temporary token/session

Receipt allocation:
exact amount is authoritative

Allocation percentage:
basis points when stored

Receipt history:
audit events

Receipt resets:
revision-based

Final receipt obligations:
frozen into receivables

Dashboard:
no dedicated authoritative data tables
```

---

# 88. Items Still To Decide During Contract / Migration Design

Approval of this document does **not** require deciding every implementation detail.

The following may be finalized later:

```text
Exact PostgreSQL ENUM vs CHECK usage

Exact Supabase RLS syntax

Exact credential encryption/storage solution

Exact Teller identifiers and uniqueness guarantees

Exact Plaid identifiers and uniqueness guarantees

Exact recurring detection model

Exact receipt token expiration duration

Exact realtime authorization implementation

Exact image storage bucket structure

Exact cent-remainder ordering rule

Exact receipt revision migration mechanics

Exact indexes after query patterns are implemented
```

These should be decided before or during the corresponding implementation tasks rather than guessed prematurely.

---

# 89. Database MVP Success Condition

The database design is successful when it can safely support:

```text
Google-authenticated user
        ↓
Profile
        ↓
┌───────────────────────┐
│                       │
Banking                 Receipts
│                       │
Connections             Upload
Accounts                Items
Transactions            Participants
Recurring               Claims
                        Submissions
                        Finalization
                        Receivables
```

without:

- Duplicate user systems
- Floating-point financial state
- Teller-specific application models
- Uncontrolled participant access
- Silent receipt mutation
- Duplicate webhook processing
- Duplicate provider transactions
- Untraceable owner corrections
- Cross-user financial-data access