# Vantage-Point
## Project Vision & MVP 1 Specification
**Status:** Planning / Pre-Implementation  
**Working Title:** Vantage-Point
**Document Purpose:** Define the overall product direction and establish the initial boundaries, architecture, workflows, and requirements for MVP 1.
---
# 1. Product Vision
Vantage-Point is a personal financial platform designed to combine financial account visibility with collaborative expense management.
Modern personal finances are fragmented across checking accounts, savings accounts, credit cards, recurring expenses, and peer-to-peer reimbursements.
Existing financial applications often focus on historical budgeting. Vantage-Point is intended to eventually provide a more complete financial view by combining:
1. Financial account aggregation
2. Transaction visibility
3. Receipt-based group expense splitting
4. Reimbursement tracking
5. Safe-to-spend calculations
6. Credit card reward optimization
The long-term objective is to create a proactive financial copilot capable of understanding both traditional financial accounts and informal financial obligations between people.
MVP 1 will intentionally implement only a subset of this vision.
---
# 2. MVP 1 Objective
The first usable version of Vantage-Point will prove two major product concepts:
### Financial Account Visibility
A registered Vantage-Point user can:
- Create an account
- Sign in
- Authenticate using supported authentication methods such as Google
- Connect a supported financial institution
- View connected institutions
- View financial accounts
- View current account balances
- View transaction history
- View detected recurring transactions/payments
MVP 1 does **not** attempt to provide advanced financial recommendations or safe-to-spend calculations.
### Collaborative Receipt Splitting
A registered Vantage-Point user can:
- Upload or photograph an itemized receipt
- Have the receipt processed automatically
- Review and correct extracted receipt information
- Add the names of people participating in the expense
- Create a temporary receipt-splitting session
- Share the session through a temporary link and/or QR code
- Allow participants to claim their items without creating Vantage-Point accounts
- Split individual items among multiple participants
- Allocate tax and tip
- Review participant submissions
- Finalize the split
- Track how much each participant owes
- Manually mark reimbursements as paid
---
# 3. MVP Boundaries
MVP 1 is intentionally limited.
The following functionality is **not required for MVP 1**:
- Safe-to-spend calculations
- Cash-flow forecasting
- Automated Venmo reconciliation
- Automated Zelle reconciliation
- Automatic reimbursement matching
- Credit card reward optimization
- Credit card recommendations
- Automated card routing
- Advanced budgeting
- Automated payment collection
- Full participant accounts
- Strong identity verification for receipt participants
These features may be introduced in later phases.
---
# 4. User Identity
Vantage-Point has one primary user identity.
A Vantage-Point user owns both:
- Banking-related data
- Receipt-related data
These should remain separate application domains but share the same Vantage-Point user.
Conceptually:
```text
                     Vantage-Point
                          │
              ┌───────────┴───────────┐
              │                       │
         Banking Domain          Receipt Domain
              │                       │
       Connections                  Receipts
              │                       │
          Accounts                Sessions
              │                       │
       Transactions              Participants
                                      │
                                    Claims
                                      │
                                  Receivables
```
Separate authentication systems should **not** be created for banking and receipt functionality.
---
# 5. Authentication
Authentication should be handled through an established authentication provider such as Supabase Auth.
The application may support Google authentication.
Vantage-Point should maintain an application profile associated with the authenticated identity.
Conceptually:
```text
Google
   ↓
Supabase Auth
   ↓
Vantage-Point User ID
   ↓
User Profile
   ↓
Application Data
```
Vantage-Point should not independently store Google passwords or recreate an authentication system.
The authenticated user's identity should be established by the backend rather than trusted from frontend-provided user identifiers.
---
# 6. Banking Domain
## 6.1 Banking MVP Flow
The expected user flow is:
```text
Create Account
      ↓
Sign In
      ↓
Connect Financial Institution
      ↓
Institution Connection Created
      ↓
Accounts Imported
      ↓
Transactions Imported
      ↓
Balances Displayed
      ↓
Recurring Activity Displayed
```
---
# 7. Financial Institution Connections
Vantage-Point will initially evaluate/use:
- Teller
- Plaid
The external financial provider should be treated as a data source rather than as Vantage-Point's internal data model.
Provider-specific data should be normalized into Vantage-Point's internal representation.
Conceptually:
```text
Teller ─────┐
            │
            ▼
       Sync Worker
            │
            ▼
      Normalization
            │
            ▼
Vantage-Point Financial Model
Plaid ──────┘
```
Application code outside the integration layer should avoid depending directly on Teller/Plaid-specific transaction formats.
---
# 8. Banking Data Model — Conceptual
The initial banking relationship should resemble:
```text
User
 │
 └── InstitutionConnection
          │
          ├── Account
          │      │
          │      └── Transaction
          │
          └── Account
                 │
                 └── Transaction
```
An institution connection and financial account are different concepts.
For example:
```text
User
 ↓
Chase Connection
 ↓
├── Checking Account
├── Savings Account
└── Credit Card
```
Each financial account can contain many transactions.
Exact database schemas will be designed separately.
---
# 9. Banking Credentials and Secrets
Financial-provider credentials must never be exposed through the frontend.
Application-level provider credentials belong in secure backend configuration or secret-management infrastructure.
Examples include:
- Plaid secrets
- Teller credentials/certificates
- Database administrative credentials
- Backend service credentials
Per-user financial connection credentials must also remain backend-only.
Conceptually, an institution connection may eventually contain information such as:
```text
InstitutionConnection
id
user_id
provider
provider_connection_id
encrypted_provider_credentials
status
created_at
last_synced_at
```
Sensitive credentials must:
- Never be returned to the browser
- Never be intentionally logged
- Never be committed to Git
- Be encrypted/protected appropriately at rest
---
# 10. Banking MVP Responsibilities
The banking portion of MVP 1 is primarily informational.
The application should display:
- Connected institutions
- Accounts
- Account types
- Current balances
- Transactions
- Transaction status where available
- Recurring transaction/payment information
Advanced financial calculations are deferred.
---
# 11. Receipt Splitting Domain
Receipt splitting is the second major component of MVP 1.
The primary objective is to make splitting a group expense significantly faster than manually calculating each person's share.
---
# 12. Receipt Owner Flow
The registered Vantage-Point user is the receipt owner.
Expected workflow:
```text
Upload / Photograph Receipt
            ↓
      Receipt Processing
            ↓
       OCR / AI Extraction
            ↓
       Owner Verification
            ↓
       Add Participants
            ↓
      Start Split Session
            ↓
       Generate QR / Link
            ↓
      Participants Claim
            ↓
       Owner Reviews
            ↓
   Participant Confirmation
            ↓
         Finalize
            ↓
    Receivables Created
```
---
# 13. Receipt Processing
Receipt images will be sent through a vision-processing pipeline.
The pipeline should attempt to extract:
- Merchant
- Date
- Individual items
- Item quantities
- Item prices
- Subtotal
- Tax
- Tip
- Discounts where detectable
- Total
AI/OCR results must **not** automatically become authoritative financial information.
The receipt owner must have an opportunity to verify and correct extracted information before sharing begins.
The owner should be able to:
- Add missing items
- Remove incorrectly detected items
- Correct names
- Correct quantities
- Correct prices
- Correct tax
- Correct tip
- Correct total
---
# 14. Receipt Validation
Before the receipt enters the claiming phase, basic consistency checks should occur.
For example:
```text
Item totals
+ tax
+ tip
- discounts
≈ receipt total
```
Differences should be surfaced to the owner for review rather than silently ignored.
---
# 15. Receipt Participants
Receipt participants do not require Vantage-Point accounts.
Before sharing the receipt, the owner enters participant names.
Example:
```text
Participants
Ryan
Alex
John
Sarah
```
When someone enters the temporary receipt session, they select the participant name representing them.
The interface should preferably present the existing participant list rather than requiring exact free-text matching.
Example:
```text
Who are you?
○ Ryan
○ Alex
○ John
○ Sarah
```
Participant identity is intentionally lightweight for MVP 1.
Selecting "John" does not cryptographically prove that the participant is John.
The MVP security assumption is that participants are trusted members of a temporary shared expense session.
---
# 16. Temporary Receipt Sessions
After the receipt has been verified and participants have been added, the owner can create a temporary split session.
The system generates:
- A secure temporary share link
- Optionally a QR code representing that link
The QR code is intended to make in-person splitting particularly easy.
Participants sitting at the same table can scan the QR code and immediately enter the split session.
---
# 17. Session Expiration
Receipt-sharing sessions should be temporary.
Initial target:
**Approximately 20–30 minutes**
The exact duration should remain configurable.
Expiration affects access, not stored receipt data.
Therefore:
```text
Session expires
      ≠
Receipt deleted
```
If the session expires before everyone responds, the owner may:
- Generate another temporary link
- Extend/restart the session
- Manually allocate missing participants' items
Previous valid submissions remain stored.
Once a receipt is finalized, active participant links should become invalid.
---
# 18. Real-Time Receipt State
Receipt claiming is collaborative and should provide near-real-time state updates.
A realtime mechanism such as WebSockets or Supabase Realtime may be used.
Conceptually:
```text
                   Receipt State
                        │
                  Realtime Layer
                        │
          ┌─────────────┼─────────────┐
          ▼             ▼             ▼
        Ryan          Alex          Sarah
```
If an accepted allocation changes, connected participants should receive the updated state without manually refreshing the page.
Realtime communication improves the user experience but does **not** determine correctness.
The server/database remains authoritative.
---
# 19. Item Claims
Participants can select the items they purchased.
An item may belong entirely to one participant:
```text
Burger — $16
Ryan: 100%
```
An item may also be shared:
```text
Pizza — $24
Ryan: 50%
Alex: 25%
John: 25%
```
MVP 1 should support:
- Full-item claims
- Equal splitting
- Custom percentage allocation
A convenience control should allow:
**Split equally**
The system must ensure that total allocations for an item cannot exceed 100%.
---
# 20. Server-Side Validation
The frontend must not be trusted to enforce financial correctness.
For every item:
```text
SUM(accepted allocations) <= 100%
```
must be enforced by backend/database logic.
This prevents concurrent users from accidentally over-allocating an item.
Example race condition:
```text
Pizza has 50% remaining.
Ryan sees 50%.
Alex sees 50%.
Both attempt to claim 50%
at exactly the same time.
```
The server/database must ensure that only valid resulting allocations are accepted.
Realtime updates are responsible for synchronizing screens.
Database transactions/constraints are responsible for correctness.
---
# 21. Participant Submission
Participants may make selections before submitting.
Conceptually:
```text
Ryan selects:
Burger — 100%
Pizza — 50%
[Submit Claim]
```
Submission sends the proposed claim to the backend for validation.
Once submitted, the participant's original submission should be preserved.
Participants should not freely rewrite historical submissions after submission.
---
# 22. Owner Corrections
The receipt owner may correct participant allocations when necessary.
However, owner edits must not silently overwrite historical participant submissions.
The system should preserve:
- Original participant submission
- Owner modification
- Previous allocation
- New allocation
- Timestamp
- Actor responsible for change
- Optional reason/context
An owner modification affecting a participant should require that participant to reconfirm before finalization when practical.
---
# 23. Audit History
Receipt splits should maintain an audit history.
Example:
```text
14:02 — Alex submitted Pizza 50%
14:03 — Ryan submitted Pizza 50%
14:05 — Owner changed Alex Pizza allocation 50% → 25%
14:05 — Alex reconfirmation required
14:07 — Alex confirmed
```
The purpose is not regulatory accounting.
The purpose is to make informal disputes understandable and prevent invisible changes.
---
# 24. Receipt State Machine
Receipt lifecycle should be represented explicitly.
Initial conceptual states:
```text
DRAFT
  ↓
PROCESSING
  ↓
NEEDS_REVIEW
  ↓
READY_TO_SHARE
  ↓
CLAIMING
  ↓
AWAITING_CONFIRMATION
  ↓
FINALIZED
```
Owner changes may cause:
```text
AWAITING_CONFIRMATION
          ↓
       OWNER EDIT
          ↓
RECONFIRMATION_REQUIRED
```
The exact state model may evolve during implementation.
---
# 25. Receipt Reset
Once claiming has started, the underlying receipt should generally be locked.
If the source receipt or OCR extraction is seriously incorrect, the owner may reset the split.
Resetting should:
- Preserve historical/audit information
- Invalidate incompatible claims
- Return the receipt to an editable/review state
- Require a new claiming/confirmation cycle
The system should not silently change receipt items underneath existing participant claims.
---
# 26. Tax and Tip Allocation
The receipt owner chooses the allocation method.
MVP 1 should support at least:
### Proportional
Tax/tip distributed according to each participant's item subtotal.
### Equal
Tax/tip divided equally among included participants.
The chosen method should be visible before finalization.
---
# 27. Finalization
A receipt should only be finalized after the required allocations and confirmations are complete or the owner explicitly resolves missing participants.
Finalization creates a stable snapshot of the split.
Example:
```text
Receipt #123
FINALIZED
Ryan:   $42.19
Alex:   $51.03
John:   $37.50
Sarah:  $53.50
```
The finalized result should not silently change afterward.
Future corrections should create a revision/reopening workflow rather than rewriting historical state invisibly.
---
# 28. Receivables
After finalization, amounts owed to the receipt owner become receivables.
MVP 1 will track payment manually.
Example:
```text
Alex
Owes: $51.03
Status: UNPAID
```
The owner may mark the obligation:
```text
PAID
```
Future versions may automatically detect repayment through Venmo, Zelle, or connected financial transactions.
Automatic payment reconciliation is **not part of MVP 1**.
---
# 29. Dashboard
The Vantage-Point dashboard eventually unifies information from both major domains.
MVP dashboard information may include:
### Banking
- Institutions
- Accounts
- Current balances
- Recent transactions
- Recurring activity
### Receipt Splitting
- Active receipts
- Finalized receipts
- Participants
- Amounts owed
- Payment status
- Receipt history
The frontend is responsible for presentation.
Financial/provider credentials and authoritative financial validation remain backend responsibilities.
---
# 30. MVP System Architecture
Initial architecture should remain intentionally simple.
```text
                    ┌──────────────────┐
                    │   Next.js Web    │
                    │      Client      │
                    └────────┬─────────┘
                             │
                             │ HTTPS
                             ▼
                    ┌──────────────────┐
                    │   Backend API    │
                    │                  │
                    │ Auth             │
                    │ Accounts         │
                    │ Transactions     │
                    │ Receipts         │
                    │ Splits           │
                    │ Dashboard        │
                    └────────┬─────────┘
                             │
               ┌─────────────┼─────────────┐
               │             │             │
               ▼             ▼             ▼
          Sync Worker   Vision Worker   PostgreSQL
               │             │
        ┌──────┴──────┐      ▼
        ▼             ▼    OCR / AI
      Teller         Plaid
```
Realtime receipt functionality may additionally use a realtime communication layer.
```text
PostgreSQL
     │
     ▼
Realtime Service
     │
 ┌───┼────┐
 ▼   ▼    ▼
P1   P2   P3
```
---
# 31. Architecture Principle: Avoid Premature Microservices
MVP 1 should not create independent services merely because the eventual product is large.
The backend can remain modular internally.
Example:
```text
backend/
auth/
accounts/
institutions/
transactions/
receipts/
participants/
claims/
receivables/
dashboard/
```
Specialized worker processes are appropriate where asynchronous processing provides clear value.
Initial candidates:
- Bank synchronization worker
- Receipt vision worker
A dedicated ledger engine is **not currently required for MVP 1**.
---
# 32. Future Ledger Engine
A true ledger/calculation engine may become appropriate when Vantage-Point introduces:
- Safe-to-spend
- Cash-flow projections
- Pending obligations
- Automated receivable reconciliation
- Credit-card payment modeling
- Financial forecasting
That future architecture might resemble:
```text
Financial Events
       ↓
Ledger Engine
       ↓
Derived Financial State
       ↓
├── Safe-to-Spend
├── Obligations
├── Receivables
└── Cash Position
```
This is intentionally deferred.
---
# 33. Future Card Optimization
The long-term product may include a credit-card rewards engine.
Potential functionality:
- Track cards owned by the user
- Maintain reward rules
- Understand merchant/category spending
- Determine optimal card usage
- Detect reward leakage
- Recommend cards based on historical spending
This functionality is **not part of MVP 1**.
---
# 34. Important MVP Failure Cases
The implementation should explicitly consider the following.
### Banking
- Provider sends duplicate webhook/event
- Pending transaction becomes posted
- Pending transaction disappears
- Posted amount differs from pending authorization
- User reconnects the same institution
- Institution connection expires
- Provider temporarily fails
- Data becomes stale
- Internal transfers appear as spending
- Credit-card payments create apparent duplicate spending
- Refunds occur
- Transaction descriptions change
- Recurring-payment detection is uncertain
Bank ingestion should be designed to be idempotent where applicable.
---
# 35. Receipt Failure Cases
The implementation should consider:
- OCR reads incorrect price
- OCR misses item
- OCR duplicates item
- Receipt has multiple quantities
- Multiple participants share an item
- Unequal item split
- Simultaneous claims
- Allocations exceed 100%
- Participant never responds
- Participant closes browser
- Session expires
- Shared link leaks
- Participant chooses wrong identity
- Owner changes allocation
- Owner changes receipt after claims exist
- Tax allocation disagreement
- Tip allocation disagreement
- Discounts/coupons
- Gift cards
- Included gratuity
- Partial reimbursement
- Overpayment
- Receipt uploaded twice
Not every case must be completely solved in MVP 1.
Each should eventually be classified as:
```text
MUST HANDLE IN MVP
EXPLICITLY UNSUPPORTED IN MVP
POST-MVP
```
---
# 36. Core Engineering Principles
The MVP should follow several project-wide principles.
### Backend authority
The backend/database is authoritative for financial state and allocation validity.
### Financial precision
Floating-point arithmetic should not be used for authoritative currency calculations.
Currency should use integer minor units or an appropriate exact decimal representation.
Example:
```text
$42.19 → 4219 cents
```
### Idempotency
Operations such as bank synchronization and webhook processing should tolerate duplicate delivery where applicable.
### Auditability
Important receipt-state changes should be attributable and historically understandable.
### Least privilege
Participants receive only the access required to interact with their temporary receipt session.
### Secret isolation
Financial provider secrets and user financial credentials never enter frontend code.
### Explicit ownership
Every important piece of data should have a clearly defined owner and modification path.
---
# 37. Development Strategy
Vantage-Point should be built incrementally rather than implementing every domain simultaneously.
Recommended planning sequence:
```text
MVP Scope
    ↓
Requirements
    ↓
System Design
    ↓
Data Model
    ↓
API / Event Contracts
    ↓
Security Model
    ↓
Agent Responsibilities
    ↓
Local Development Environment
    ↓
CI
    ↓
First Vertical Slice
    ↓
Architecture Review
    ↓
Parallel Development
    ↓
Staging
    ↓
CD
```
---
# 38. First Vertical Slice
Before aggressively parallelizing development across AI agents, implement one complete path through the architecture.
A useful banking slice:
```text
Authentication
      ↓
Connect/Test Financial Source
      ↓
Sync Account
      ↓
Store Account
      ↓
Store Transactions
      ↓
Backend API
      ↓
Display Transactions
```
A useful receipt slice:
```text
Upload Receipt
      ↓
Vision Processing
      ↓
Extract Items
      ↓
Owner Review
      ↓
Create Split Session
      ↓
Participant Claim
      ↓
Store Allocation
      ↓
Display Result
```
These first slices should validate architectural assumptions before large-scale parallel development begins.
---
# 39. AI Agent Development Strategy
The repository will eventually support multiple specialized development agents.
Potential responsibilities include:
```text
Orchestrator
    │
    ├── Banking / Ingestion Agent
    ├── Receipt Agent
    ├── Frontend Agent
    ├── Database Agent
    └── Integration / Review Agent
```
Exact agent roles are intentionally **not defined in this document**.
Agent design should be performed after MVP boundaries and major system responsibilities are understood.
Agents should eventually coordinate through:
- Explicit file ownership
- Shared contracts
- Database migrations
- Tests
- Git branches
- Pull requests
- CI
- Architecture decisions
rather than relying solely on conversational context.
---
# 40. Current Unresolved Design Decisions
The following decisions remain intentionally open:
- Exact database schema
- Exact API routes
- Exact Go backend organization
- Teller vs. Plaid priority/fallback behavior
- Exact recurring-payment detection mechanism
- Exact realtime technology
- Receipt-session token implementation
- Exact participant reconfirmation UX
- Receipt revision/versioning implementation
- Image-storage architecture
- Notification mechanism
- Data-retention policies
- Detailed authorization model
- CI/CD implementation
- Cloud infrastructure configuration
- Agent ownership model
- Agent orchestration strategy
These should be resolved through subsequent design work rather than guessed during MVP planning.
---
# 41. MVP Success Definition
MVP 1 is successful when a user can perform both of the following end-to-end workflows reliably.
### Banking
```text
Sign Up
→ Sign In
→ Connect Institution
→ View Accounts
→ View Balances
→ View Transactions
→ View Basic Recurring Activity
```
### Receipt Splitting
```text
Sign In
→ Upload Receipt
→ Extract Items
→ Verify Receipt
→ Add Participants
→ Generate Temporary QR/Link
→ Participants Join
→ Participants Claim/Split Items
→ Owner Reviews
→ Participants Confirm
→ Finalize Receipt
→ View Amounts Owed
→ Manually Mark Payments
```
At that point, Vantage-Point has demonstrated the two core concepts required to proceed into more advanced financial intelligence.
---
# 42. Immediate Next Planning Stage
The next stage after this document is **not implementation**.
The next stage is to define how development will be safely divided among humans and AI agents.
That work should establish:
1. Agent roles
2. Agent ownership boundaries
3. Root agent rules
4. Service/module-specific rules
5. Task specification format
6. Git branch strategy
7. Pull request strategy
8. Shared contract ownership
9. Database migration ownership
10. Testing responsibilities
11. Integration/review responsibilities
12. CI authority
13. Deployment authority
Once those rules exist, the project can move toward detailed database design, API contracts, local environment setup, CI, and implementation without allowing parallel agents to independently redefine the architecture.