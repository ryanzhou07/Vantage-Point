# Vantage-Point Dashboard Requirements

## 1. Purpose

The Dashboard is the primary authenticated interface for Vantage-Point.

It brings information from the Foundation, Banking, and Receipt domains into one coherent financial overview.

The Dashboard is primarily a **presentation and interaction layer**.

It must not duplicate authoritative Banking or Receipt business logic.

---

## 2. Authentication

Only authenticated Vantage-Point users may access the main Dashboard.

For MVP 1, authentication requires a valid Google-authenticated Vantage-Point session.

Conceptually:

```text
Google Authentication
        ↓
Supabase Auth
        ↓
Vantage-Point
        ↓
Dashboard
```

If the user's session is invalid or expired, protected Dashboard information must not be displayed.

---

## 3. Dashboard Responsibilities

The Dashboard should provide access to:

- Connected financial institutions
- Financial accounts
- Account balances
- Recent transactions
- Recurring financial activity
- Receipt splitting
- Active receipts
- Finalized receipts
- Outstanding receivables
- Receivable payment status

The Dashboard presents information owned by other domains.

---

## 4. Main Navigation

MVP should provide clear navigation between major Vantage-Point areas.

Conceptually:

```text
Vantage-Point

├── Overview
├── Accounts
├── Transactions
├── Receipts
└── Settings / Profile
```

Exact labels and visual navigation design may change during frontend implementation.

---

## 5. Overview

The Overview page should provide a concise summary of the user's current financial information.

It may include:

```text
Overview

Total Account Balance

Accounts
├── Checking
├── Savings
└── Credit Cards

Recent Transactions

Recurring Activity

Receipts
├── Active Splits
└── Recent Receipts

Money Owed to You
```

The Overview should summarize existing domain data rather than creating a separate source of financial truth.

---

## 6. Account Summary

The Dashboard should show the user's connected financial accounts.

Each account summary should display appropriate available information such as:

- Institution
- Account name
- Account type
- Current balance
- Available balance where available
- Synchronization status/freshness

Sensitive account identifiers should be masked where appropriate.

---

## 7. Total Balance

The Dashboard may calculate a summarized account balance from supported connected accounts.

Any aggregate must clearly define which account types are included.

For MVP, the Dashboard must not present an ambiguous number as the user's true net worth.

For example:

```text
Connected Cash Balance
$8,420.31
```

is preferable to:

```text
Net Worth
$8,420.31
```

when investments, debts, or other assets are not represented.

---

## 8. Credit Card Balances

Credit-card balances must not be treated as positive cash.

The Dashboard should visually distinguish:

- Cash/depository balances
- Credit balances or amounts owed

The aggregation model must not simply add all account balances together without considering account type.

---

## 9. Account Detail

A user should be able to select an account and view more information.

The account view may include:

- Account name
- Institution
- Account type
- Current balance
- Available balance
- Recent transactions
- Last synchronization time
- Connection status

Banking remains authoritative for this information.

---

## 10. Transactions

The Dashboard should provide a transaction view containing imported Banking transactions.

Useful information may include:

- Merchant/description
- Amount
- Date
- Account
- Pending/posted status
- Category where available

The Dashboard should not independently modify authoritative Banking transaction records.

---

## 11. Transaction Sign Presentation

The Dashboard must consistently represent incoming and outgoing money according to the Banking domain's approved amount-sign convention.

Presentation may translate the internal representation into human-friendly language or visual treatment.

The Dashboard must not define a conflicting financial sign convention.

---

## 12. Pending Transactions

Pending transactions should be visually distinguishable from posted transactions where Banking provides that information.

The UI must not imply that pending activity has the same finality as posted activity.

---

## 13. Banking Data Freshness

The Dashboard should communicate when Banking data may be stale.

Examples may include:

```text
Updated 8 minutes ago
```

or:

```text
Bank connection needs attention
```

The Dashboard consumes synchronization information provided by Banking.

It must not invent synchronization state.

---

## 14. Banking Errors

If a financial connection is experiencing an error, the Dashboard should provide understandable feedback.

Examples include:

- Synchronization temporarily unavailable
- Connection requires attention
- Reauthentication required
- Provider temporarily unavailable

Existing synchronized data should remain viewable where safe and appropriate.

---

## 15. Connect Institution

The Dashboard should provide an obvious way for authenticated users to begin connecting a financial institution.

For MVP:

```text
Dashboard
   ↓
Connect Bank
   ↓
Banking Domain
   ↓
Teller
```

Plaid may later be presented as a secondary/fallback connection path according to Banking requirements.

The Dashboard does not own provider integration logic.

---

## 16. Recurring Activity

The Dashboard should present recurring financial activity identified by Banking.

Examples may include:

```text
Recurring

Spotify          $11.99
Netflix          $22.99
Internet         $64.99
Gym              $30.00
```

Recurring detection is owned by Banking.

Dashboard presentation must not imply certainty beyond the confidence/information supplied by Banking.

---

## 17. Receipt Area

The Dashboard should provide access to the user's Receipt functionality.

The user should be able to:

- Start a new receipt
- View receipts being processed
- View receipts requiring review
- View active splits
- View receipts awaiting confirmation
- View finalized receipts
- View related receivables

---

## 18. Create Receipt

The Dashboard should provide an obvious action for starting the receipt workflow.

Conceptually:

```text
+ Split Receipt
      ↓
Upload / Capture Receipt
      ↓
Receipt Domain
```

Once the Receipt workflow begins, receipt-specific interfaces remain owned by the Receipt domain.

---

## 19. Receipt Status

The Dashboard should present meaningful receipt lifecycle information.

Examples:

```text
Needs Review
Claiming
Waiting for Confirmation
Finalized
```

The Dashboard should consume authoritative Receipt state rather than creating its own independent receipt statuses.

---

## 20. Active Receipts

The user should be able to quickly find receipts requiring action.

Examples include:

- Receipt processing
- Receipt needs owner review
- Participants currently claiming
- Participant submission missing
- Reconfirmation required
- Receipt ready for finalization

Actionable receipts should be distinguishable from completed receipt history.

---

## 21. Finalized Receipts

Finalized receipts should remain accessible to the owner.

A finalized receipt summary may include:

- Merchant
- Date
- Total
- Participants
- Final participant amounts
- Receivables
- Payment status

The Dashboard must not silently modify finalized receipt data.

---

## 22. Receivables

The Dashboard should present money owed to the authenticated receipt owner.

Conceptually:

```text
Money Owed to You

Alex       $25.34    OWED
Chris      $18.72    PAID
Jordan     $31.15    OWED
```

Receivable amounts and statuses are authoritative Receipt-domain information.

---

## 23. Outstanding Receivable Summary

The Dashboard may calculate a summary such as:

```text
Outstanding Reimbursements

$56.49
```

This value must be derived from current authoritative receivable data.

Paid receivables must not remain included in the outstanding amount.

---

## 24. Mark Paid

The owner should be able to initiate the MVP manual "mark paid" action from an appropriate Dashboard or Receipt interface.

The actual payment-status update belongs to the Receipt domain.

The Dashboard should call the appropriate Receipt operation rather than independently changing local state as the source of truth.

---

## 25. Loading States

Dashboard areas dependent on asynchronous data should have appropriate loading states.

The UI should distinguish:

```text
Loading
```

from:

```text
No Data
```

and:

```text
Error
```

These states must not be presented as equivalent.

---

## 26. Empty States

The Dashboard should provide useful empty states.

Examples:

No connected accounts:

```text
No accounts connected yet.

[ Connect Bank ]
```

No receipts:

```text
No receipts yet.

[ Split a Receipt ]
```

No outstanding receivables:

```text
No outstanding reimbursements.
```

---

## 27. Error States

A failure in one Dashboard domain should not unnecessarily make the entire Dashboard unusable.

For example:

```text
Banking unavailable
        +
Receipt data available
```

should still allow receipt functionality to operate when technically possible.

---

## 28. Cross-Domain Failure Isolation

The Dashboard should avoid assuming that all domains are simultaneously healthy.

Conceptually:

```text
Dashboard
├── Banking ✓
├── Receipts ✓
└── Recurring ✕
```

A localized failure should be represented as a localized failure where possible.

---

## 29. Responsive Design

The Dashboard must work on both desktop and mobile-sized interfaces.

This is particularly important because receipt splitting may commonly occur from phones.

Exact breakpoints and component designs will be determined during frontend implementation.

---

## 30. Financial Formatting

Currency values must be displayed consistently.

For MVP 1, USD may be the primary supported currency.

Presentation formatting must not alter authoritative stored monetary values.

Examples:

```text
$1,234.56
-$42.18
```

Exact negative-value presentation may follow the approved UI design.

---

## 31. Financial Calculation Authority

The Dashboard may perform simple presentation-level aggregation when explicitly defined.

It must not independently implement complex financial logic owned by Banking or Receipts.

For example:

Allowed:

```text
Sum outstanding receivables
```

Not allowed:

```text
Invent independent transaction reconciliation logic
```

or:

```text
Recalculate finalized receipt allocations differently
```

---

## 32. Data Ownership

The Dashboard does not own authoritative Banking or Receipt records.

Conceptually:

```text
Foundation
    ↓
Identity

Banking
    ↓
Accounts / Transactions / Recurring

Receipts
    ↓
Receipts / Receivables

        ↓
    Dashboard
        ↓
Presentation
```

---

## 33. API Usage

The Dashboard should consume defined Vantage-Point domain APIs/contracts.

It should not:

- Query provider secrets
- Call Teller using private backend credentials
- Call Plaid using private backend credentials
- Reimplement Receipt authorization
- Circumvent backend ownership checks

---

## 34. User Privacy

Dashboard APIs and interfaces must display only data the authenticated user is authorized to access.

Client-side routing or hiding UI elements is not sufficient authorization.

Backend/database authorization remains required.

---

## 35. Profile

The Dashboard should provide basic access to the authenticated user's profile information.

MVP profile presentation may include:

- Display name
- Google-associated email
- Profile image where available

Advanced profile management is not required.

---

## 36. Sign Out

The Dashboard must provide an accessible sign-out action.

Sign-out behavior is owned by Foundation.

After sign-out, protected Dashboard data must no longer remain accessible through normal application navigation.

---

## 37. MVP Dashboard Experience

The intended high-level experience is:

```text
                VANTAGE-POINT

┌─────────────────────────────────────┐
│ Overview                            │
│                                     │
│ Connected Cash       $8,420.31      │
│ Outstanding Owed       $126.44      │
├─────────────────────────────────────┤
│ Accounts                            │
│ Checking             $3,201.40      │
│ Savings              $5,218.91      │
│ Credit Card            $842.13 owed │
├─────────────────────────────────────┤
│ Recent Transactions                 │
│ Grocery Store          -$54.21      │
│ Payroll             +$1,420.00      │
├─────────────────────────────────────┤
│ Recurring                           │
│ Spotify               $11.99        │
│ Internet              $64.99        │
├─────────────────────────────────────┤
│ Receipts                            │
│ Dinner — Claiming                   │
│ Lunch — Needs Review                │
├─────────────────────────────────────┤
│ Money Owed to You                   │
│ Alex                  $25.34        │
│ Jordan                $31.15        │
└─────────────────────────────────────┘
```

This is conceptual and does not prescribe final visual design.

---

## 38. MVP Acceptance Requirements

Dashboard MVP is functionally complete when an authenticated user can:

- Enter the protected application
- Navigate between major MVP areas
- Begin connecting a financial institution
- View connected accounts
- View balances
- View recent transactions
- Distinguish pending transactions where available
- See banking-data freshness/status
- View recurring activity
- Start a receipt split
- Find active receipts
- Find finalized receipts
- View outstanding receivables
- View payment status
- Mark an eligible receivable paid through the Receipt domain
- Access basic profile functionality
- Sign out

And when:

- Dashboard data is scoped to the authenticated user.
- Banking remains authoritative for banking information.
- Receipts remains authoritative for receipt information.
- Foundation remains authoritative for authentication identity.
- Credit balances are not treated as positive cash.
- Loading, empty, and error states are distinguishable.
- A localized domain failure does not unnecessarily destroy the entire Dashboard.
- The interface is usable on desktop and mobile.

---

## 39. Explicitly Deferred Dashboard Features

The following are not required for Dashboard MVP 1:

- Safe-to-spend
- Net-worth calculation
- Investment portfolio analysis
- Budgeting system
- Cash-flow forecasting
- Credit-card optimization
- Credit-card recommendations
- Reward tracking
- Automated reimbursement matching
- Advanced spending analytics
- Financial goal tracking
- AI financial recommendations
- Complex customizable widgets
- User-defined dashboard layouts
- Multi-currency aggregation
- Administrative dashboard

These require separate approved requirements.