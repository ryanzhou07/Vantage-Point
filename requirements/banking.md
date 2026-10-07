# Vantage-Point Banking Requirements

## 1. Purpose

The Banking domain gives authenticated Vantage-Point users a unified view of their connected financial institutions, accounts, balances, transactions, and recurring financial activity.

The Banking domain is responsible for acquiring financial data from external providers and converting that data into Vantage-Point's internal financial representation.

For MVP 1, Banking is primarily informational.

Vantage-Point will not initiate bank transfers or move user funds.

---

## 2. Provider Strategy

Vantage-Point will use:

- **Teller — Primary banking provider**
- **Plaid — Secondary provider / fallback**

Initial Banking implementation should prioritize Teller.

Plaid support should not be required before the first Teller-based banking vertical slice can function.

However, the architecture must avoid making Vantage-Point's internal banking model dependent directly on Teller-specific data structures.

Conceptually:

```text
Teller ──────┐
             │
             ▼
       Provider Adapter
             │
             ▼
        Normalization
             │
             ▼
   Vantage-Point Banking Model
             ▲
             │
       Provider Adapter
             ▲
             │
Plaid ───────┘
```

Provider-specific behavior belongs at the integration boundary.

---

## 3. Provider Selection

Teller should be attempted first for supported institution connections.

Plaid may be used when:

- Teller does not support the required institution
- Teller cannot provide required functionality for that institution
- Vantage-Point intentionally routes the connection through Plaid
- A future approved provider-routing strategy selects Plaid

The exact automatic fallback UX does not need to be implemented in the first banking slice.

MVP architecture should make future fallback possible without redesigning the internal banking model.

---

## 4. Authentication Requirement

Only authenticated Vantage-Point users may connect and manage their financial institutions.

Banking must use the canonical authenticated user identity established by Foundation.

The Banking domain must not create a separate banking-user identity system.

Conceptually:

```text
Google
   ↓
Supabase Auth
   ↓
Vantage-Point User
   ↓
Institution Connections
   ↓
Accounts
   ↓
Transactions
```

---

## 5. Institution Connections

An `InstitutionConnection` represents the relationship between a Vantage-Point user and an external financial institution through a supported provider.

Examples:

```text
User
 ↓
Chase Connection
 ↓
├── Checking
├── Savings
└── Credit Card
```

A connection is not the same thing as an account.

One institution connection may contain multiple financial accounts.

A user may have multiple institution connections.

---

## 6. Institution Connection Information

Vantage-Point should maintain enough information to identify and operate a connection.

Conceptually, this may include:

```text
InstitutionConnection

id
user_id
provider
provider_connection_id
institution_name
status
created_at
updated_at
last_successful_sync_at
```

Exact database columns will be determined during database design.

Provider credentials should not be treated as normal public connection fields.

---

## 7. Connection Ownership

Every institution connection must belong to exactly one authenticated Vantage-Point user.

A user must not be able to:

- View another user's institution connection
- Synchronize another user's institution connection
- Access another user's provider credentials
- Access accounts belonging to another user's connection

Ownership must be enforced server-side and/or through appropriate database authorization.

Frontend filtering alone is insufficient.

---

## 8. Provider Credentials

Per-user banking-provider credentials must remain backend-only.

They must never be:

- Returned to the browser unnecessarily
- Included in frontend application state
- Intentionally logged
- Committed to Git
- Exposed through normal API responses

Credentials must be protected appropriately at rest.

Exact credential-storage architecture will be decided separately.

---

## 9. Financial Accounts

An institution connection may contain one or more financial accounts.

MVP should support financial account types returned by the supported provider that are relevant to Vantage-Point, including where available:

- Checking
- Savings
- Credit card

Additional account types may be supported when provider data and approved requirements allow them.

---

## 10. Account Information

Vantage-Point should maintain normalized account information.

Conceptually:

```text
Account

id
institution_connection_id
provider_account_id
name
account_type
account_subtype
currency
current_balance
available_balance
status
created_at
updated_at
last_synced_at
```

This is conceptual rather than the final database schema.

Provider-specific account types should be normalized where practical.

---

## 11. Account Ownership

Account ownership is inherited through its institution connection.

Conceptually:

```text
Vantage-Point User
        ↓
InstitutionConnection
        ↓
Account
```

The application must verify this ownership chain before exposing protected account information.

---

## 12. Balances

MVP should display the most recent balance information available from the banking provider.

Where available, Vantage-Point may distinguish:

- Current balance
- Available balance

The UI should not imply that a balance is live when the underlying provider data may be stale.

The system should retain enough synchronization metadata to determine when account information was last successfully refreshed.

---

## 13. Transactions

Vantage-Point should import and display transactions belonging to connected accounts.

Normalized transaction information may include:

```text
Transaction

id
account_id
provider_transaction_id
amount
currency
description
merchant_name
date
status
category
created_at
updated_at
```

Exact fields will be determined during database design.

---

## 14. Transaction Ownership

Transactions belong to accounts.

Accounts belong to institution connections.

Institution connections belong to Vantage-Point users.

Conceptually:

```text
User
 ↓
InstitutionConnection
 ↓
Account
 ↓
Transaction
```

Authorization should follow this ownership chain.

---

## 15. Financial Precision

Authoritative monetary values must not use floating-point arithmetic.

Vantage-Point should use an exact monetary representation approved during database design.

For example:

```text
$42.19
   ↓
4219 cents
```

or another exact database representation where appropriate.

Currency should be explicitly represented when required.

---

## 16. Transaction Status

Where provider information permits, Vantage-Point should distinguish transaction states such as:

- Pending
- Posted

A pending transaction and its later posted version should not automatically be treated as two unrelated expenses.

The Banking implementation must account for pending-to-posted lifecycle behavior.

---

## 17. Pending-to-Posted Reconciliation

Providers may initially report a pending transaction and later report the final posted transaction.

Possible changes include:

- Provider transaction identifier
- Amount
- Merchant description
- Date
- Status

The Banking domain should attempt to reconcile provider lifecycle information according to the provider's supported identifiers and behavior.

The system must avoid obvious duplicate spending caused by storing both representations as unrelated authoritative transactions.

Exact reconciliation logic may differ by provider.

---

## 18. Synchronization

Banking data must be synchronized from supported providers into Vantage-Point.

Synchronization may occur through:

- Initial connection import
- Explicit refresh where supported
- Background synchronization
- Provider webhooks/events
- Scheduled synchronization where appropriate

The exact scheduling mechanism will be determined during implementation design.

---

## 19. Initial Synchronization

After a financial institution is successfully connected, Vantage-Point should attempt to acquire:

1. Institution/connection information
2. Accounts
3. Balances
4. Transactions required by the MVP
5. Information needed for recurring-activity detection where available

The user should be able to distinguish between:

- Connection established
- Synchronization in progress
- Data available
- Synchronization failure

---

## 20. Idempotency

Bank synchronization and provider-event processing must tolerate duplicate delivery where applicable.

Receiving the same provider event multiple times must not create duplicate authoritative records.

Provider event handling should use stable provider identifiers or another approved idempotency mechanism.

Conceptually:

```text
(provider, provider_event_id)
        ↓
unique processing identity
```

Exact implementation will be determined during database and provider-integration design.

---

## 21. Webhooks / Provider Events

Where Teller or Plaid provides relevant webhook/event functionality, Vantage-Point may use those events to detect banking-data changes.

Provider events must be treated as external input.

Webhook handling must:

- Validate provider authenticity using the provider-supported mechanism
- Tolerate duplicate delivery
- Avoid trusting malformed event data
- Avoid exposing credentials
- Trigger appropriate synchronization or processing

A webhook event does not automatically need to contain the complete authoritative financial record.

---

## 22. Provider Normalization

Teller and Plaid may represent the same financial concept differently.

Provider adapters should translate provider-specific information into Vantage-Point's internal model.

Conceptually:

```text
Teller Transaction
        ↓
Teller Adapter
        ↓
Normalized Transaction

Plaid Transaction
        ↓
Plaid Adapter
        ↓
Normalized Transaction
```

Code outside the provider integration boundary should avoid depending directly on Teller/Plaid response structures.

---

## 23. Duplicate Institution Connections

A user may accidentally attempt to connect the same financial institution more than once.

Vantage-Point should avoid silently creating duplicate financial state when the same provider connection or account is encountered again.

Exact duplicate-connection UX will be determined later.

At minimum, provider identifiers and internal constraints should make duplicate detection possible.

---

## 24. Connection Status

Institution connections should have an explicit status.

Possible conceptual states include:

```text
CONNECTING
ACTIVE
SYNCING
REAUTH_REQUIRED
ERROR
DISCONNECTED
```

Exact status names may change during database/API design.

The important requirement is that connection health must not be represented only through absence of data.

---

## 25. Reauthentication

Financial-provider access may expire or require user action.

Vantage-Point must be capable of representing that a connection requires reauthentication.

The application must not continue presenting stale information as though synchronization is functioning normally.

Detailed provider-specific reauthentication UX may be implemented after the first banking vertical slice if necessary.

---

## 26. Provider Failure

Teller or Plaid may temporarily fail.

A provider failure should not:

- Delete previously synchronized financial data
- Corrupt existing accounts
- Create duplicate accounts
- Create duplicate transactions
- Expose provider credentials

The application should retain previously synchronized information and indicate that newer information may be unavailable or stale.

---

## 27. Stale Data

Vantage-Point should track enough synchronization information to communicate when banking information was last successfully updated.

Conceptually:

```text
Balance: $2,431.22
Last updated: 14 minutes ago
```

Exact frontend wording will be determined by Dashboard requirements.

Banking owns the underlying synchronization timestamp/status.

Dashboard owns its presentation.

---

## 28. Internal Transfers

Transfers between a user's accounts may appear as transactions.

For MVP 1, Vantage-Point does not need a complete transfer-classification engine.

However, the Banking model should avoid assuming every imported transaction represents consumer spending.

Transfer detection and advanced classification may be improved later.

---

## 29. Credit Card Payments

Payments from a checking account to a credit card may appear on both accounts.

MVP does not need to completely solve cross-account spending reconciliation.

However, Banking must not design the transaction model around the assumption that every transaction represents unique spending.

Advanced financial analysis is deferred.

---

## 30. Refunds

Refunds and reversed financial activity may appear in transaction data.

The internal transaction model must support positive and negative financial movement according to an explicitly defined amount-sign convention.

The exact sign convention must be decided during database/contract design and documented consistently.

---

## 31. Recurring Activity

MVP should provide basic recurring transaction/payment information.

Examples may include:

- Subscriptions
- Regular bills
- Other repeating outflows

Recurring detection may use:

- Provider-supplied recurring information where available
- Vantage-Point detection logic
- A combination of both

The exact detection mechanism remains a separate design decision.

Recurring classification should not be represented as guaranteed truth when confidence is uncertain.

---

## 32. Recurring Detection Limitations

Recurring detection may produce:

- False positives
- Missed recurring transactions
- Merchant-name variations
- Variable transaction amounts
- Irregular billing schedules

MVP should present recurring activity as detected information rather than guaranteed contractual obligations.

Advanced recurring-payment intelligence is deferred.

---

## 33. Banking API Boundary

The frontend should interact with Vantage-Point rather than directly using private Teller/Plaid credentials.

Conceptually:

```text
Browser
   ↓
Vantage-Point Backend
   ↓
Banking Domain
   ↓
Teller / Plaid
```

The backend is responsible for provider integration and authoritative access control.

---

## 34. Logging

Banking logs must not intentionally contain:

- Provider access tokens
- Provider secrets
- Banking credentials
- Authentication tokens

Operational logging may contain safe identifiers necessary for debugging when permitted.

Sensitive financial data should not be logged unnecessarily.

---

## 35. Banking MVP User Experience

An authenticated user should eventually be able to:

```text
Sign In with Google
        ↓
Connect Institution
        ↓
Connection Established
        ↓
Accounts Synchronize
        ↓
View Accounts
        ↓
View Balances
        ↓
View Transactions
        ↓
View Basic Recurring Activity
```

---

## 36. MVP Acceptance Requirements

Banking MVP is functionally complete when an authenticated Vantage-Point user can:

- Connect a Teller-supported financial institution
- Have the institution connection associated with their Vantage-Point identity
- Import supported financial accounts
- View normalized account information
- View current balance information
- Import and view transactions
- Distinguish pending/posted state where available
- View basic recurring activity
- See meaningful connection/synchronization state
- Retain existing data during temporary provider failures

And when:

- Teller is the primary provider.
- The internal banking model is not Teller-specific.
- Plaid can be introduced as the secondary provider without replacing the internal model.
- Provider credentials remain backend-only.
- Users cannot access another user's banking information.
- Duplicate synchronization does not create duplicate authoritative records where stable provider identity is available.
- Authoritative money values use exact representations.
- Provider failure does not destroy previously synchronized financial data.

---

## 37. Explicitly Deferred Banking Features

The following are not required for Banking MVP 1:

- Safe-to-spend
- Cash-flow forecasting
- Automated Venmo reconciliation
- Automated Zelle reconciliation
- Automated reimbursement matching
- Credit-card rewards optimization
- Credit-card recommendations
- Automated card routing
- Advanced budgeting
- Money movement
- Bank transfers
- Bill payment
- Complete transfer reconciliation
- Complete credit-card-payment reconciliation
- Advanced transaction categorization
- Advanced recurring-payment prediction
- Multi-provider connection routing optimization

These require separate approved requirements before implementation.