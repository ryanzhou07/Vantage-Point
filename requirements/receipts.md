# Vantage-Point Receipt Requirements

## 1. Purpose

The Receipt domain allows an authenticated Vantage-Point user to upload a receipt, verify extracted receipt data, invite temporary participants, collaboratively assign items, finalize each participant's share, and track money owed back to the receipt owner.

The Receipt domain must support temporary participation without requiring every participant to create a Vantage-Point account.

---

## 2. Receipt Owner

Every receipt must have exactly one authenticated Vantage-Point owner.

The owner must be identified through the canonical authenticated user identity provided by Foundation.

Conceptually:

```text
Vantage-Point User
        ↓
      Receipt
        ↓
Receipt Splitting Session
```

The Receipt domain must not trust a client-supplied `owner_user_id` as proof of ownership.

---

## 3. Receipt Participant Identity

Receipt participants are separate from authenticated Vantage-Point users.

A participant:

- Does not need a Vantage-Point account
- Exists only within the context of a receipt
- Is identified through the receipt's temporary participation flow

Conceptually:

```text
Authenticated Vantage-Point User
        ≠
Temporary Receipt Participant
```

Temporary receipt participants must not automatically become permanent Vantage-Point users.

---

## 4. Receipt Creation

An authenticated user should be able to create a receipt by uploading or capturing a receipt image.

The uploaded receipt must initially belong only to the authenticated owner.

The system should retain enough receipt metadata to support processing, review, splitting, and audit history.

---

## 5. Receipt Image

A receipt may begin from an uploaded image or photo.

The Receipt domain should retain the necessary receipt-image reference so that the owner can compare extracted data with the original receipt during review.

Exact image-storage implementation will be determined separately.

Receipt images must not be publicly accessible by default.

---

## 6. Receipt Processing

After upload, the receipt should enter an asynchronous processing flow.

Conceptually:

```text
Upload Receipt
      ↓
Receipt Created
      ↓
PROCESSING
      ↓
Vision / OCR Processing
      ↓
Extracted Receipt Data
      ↓
NEEDS_REVIEW
```

The frontend must not assume that receipt extraction completes immediately.

---

## 7. Vision / OCR Processing

Vantage-Point may use OCR, multimodal models, or other approved vision-processing services to extract receipt information.

Extraction may include:

- Merchant name
- Receipt date
- Line items
- Quantity
- Item description
- Item price
- Subtotal
- Tax
- Tip
- Discounts
- Total

Not every field must always be detected successfully.

---

## 8. OCR Output Is Not Authoritative

Vision/OCR output must be treated as a proposal rather than final financial truth.

The owner must be given an opportunity to verify and correct extracted data before participants begin claiming items.

Conceptually:

```text
OCR Result
   ↓
Owner Review
   ↓
Verified Receipt
```

Participants must not be asked to split an unverified receipt.

---

## 9. Owner Receipt Review

Before opening a split session, the owner should be able to:

- Correct merchant information
- Add missing items
- Remove incorrect items
- Correct item descriptions
- Correct prices
- Correct quantities
- Correct tax
- Correct tip
- Correct discounts where supported
- Correct receipt total

The exact editing interface belongs to the Receipt domain.

---

## 10. Financial Validation

Before a receipt becomes available for participant claiming, the Receipt domain must perform basic consistency validation.

Examples may include:

- Item prices are valid monetary values
- Quantity values are valid
- Tax is valid
- Tip is valid
- Total is valid
- Required financial fields are not malformed

Where totals do not reconcile, the owner should be warned or prevented from proceeding according to the severity of the inconsistency.

Exact reconciliation tolerance and rounding rules will be defined during financial-model design.

---

## 11. Money Representation

Authoritative receipt monetary values must use exact representations.

Floating-point values must not be used for authoritative receipt totals, item prices, allocations, tax, tip, or receivables.

Approved implementation may use:

- Integer minor units
- Exact database decimal types
- Another explicitly approved exact representation

---

## 12. Receipt Participants

Before starting participant claiming, the owner enters the expected participant names.

Example:

```text
Participants

Ryan
Alex
Chris
Jordan
```

The owner may also be included as a participant in the split when appropriate.

Participants should preferably select from the predefined participant list rather than entering arbitrary free-form identities.

---

## 13. Participant Names

Participant names are lightweight receipt-scoped identity labels.

They are not cryptographically verified identities.

For MVP 1, Vantage-Point assumes a trusted social setting, such as participants splitting a receipt together at a table.

The system should prevent two active participants from accidentally claiming the same participant identity simultaneously where practical.

Exact participant-locking UX may be decided during implementation.

---

## 14. Split Session

A receipt owner must explicitly start a temporary split session before participants can join.

The session should be associated with:

- Receipt
- Temporary access token
- Creation time
- Expiration time
- Session status

Conceptually:

```text
SplitSession

receipt_id
token_hash
created_at
expires_at
status
```

Exact schema will be determined later.

---

## 15. Temporary Session Access

Participants access a receipt through a temporary receipt-specific link.

The owner may share the link through:

- QR code
- Direct share link

The QR code should resolve to the same temporary participant-access flow.

---

## 16. Session Tokens

Temporary session tokens must:

- Be difficult to guess
- Be scoped to the appropriate receipt/session
- Expire
- Not expose internal database identifiers as authorization
- Be stored securely where appropriate

If token values are persisted, secure hashing or another approved token-storage strategy should be considered.

---

## 17. Session Expiration

Receipt split sessions are intended primarily for short collaborative sessions.

Typical expected usage may be approximately:

```text
20–30 minutes
```

However, the implementation should allow enough flexibility for slower groups.

Expiration should prevent continued unauthorized session use.

Session expiration must not delete:

- Receipt
- Existing participant records
- Submitted claims
- Allocation history
- Audit events

---

## 18. Session Extension / Reopening

If participants fail to complete the receipt before session expiry, the owner should be able to recover without recreating the entire receipt.

Potential supported behaviors include:

- Extend the session
- Generate a new temporary session
- Generate a replacement link
- Complete missing allocations manually

Exact UX may be chosen during implementation.

---

## 19. Participant Join Flow

A participant should be able to:

```text
Open temporary link / QR
        ↓
See receipt participation screen
        ↓
Select their predefined name
        ↓
Enter receipt session
        ↓
See current claim state
```

No Vantage-Point signup should be required.

---

## 20. Realtime Shared State

Participants should see receipt allocation state update while the receipt is being split.

Examples include:

```text
Pizza — $20.00

Ryan     50%
Alex     25%
Unclaimed 25%
```

Participants should not need to manually refresh after every change under normal operation.

---

## 21. Realtime Architecture

Realtime browser synchronization may use:

- Supabase Realtime
- WebSockets
- Another approved realtime mechanism

The exact implementation is not yet fixed.

Regardless of transport, the database/backend remains authoritative.

Realtime client state must not be trusted as final financial truth.

---

## 22. Claimable Items

Participants must be able to assign receipt items to themselves.

Supported MVP behavior should include:

- Full-item claim
- Shared-item claim
- Equal split
- Custom percentage split
- Custom amount split where supported by the final allocation model

A single item may be shared among more than two participants.

---

## 23. Full Item Claim

A participant should be able to claim an entire item when no other participant has been allocated any portion of that item.

Example:

```text
Burger — $14.00

Ryan — 100%
```

---

## 24. Shared Item Allocation

Items may be split across multiple participants.

Example:

```text
Pizza — $24.00

Ryan — 50%
Alex — 25%
Chris — 25%
```

The total allocation for an item must never exceed 100%.

---

## 25. Equal Split

The Receipt domain should provide an equal-split convenience action.

Example:

```text
Appetizer — $18.00
3 participants

Ryan  — $6.00
Alex  — $6.00
Chris — $6.00
```

Rounding behavior must be deterministic when exact equal division is not possible.

---

## 26. Allocation Authority

The server/database must authoritatively validate allocation changes.

The frontend must not be allowed to determine independently that an allocation is valid.

Conceptually:

```text
Participant action
      ↓
Server validation
      ↓
Atomic financial update
      ↓
Database
      ↓
Realtime update
```

---

## 27. Allocation Constraint

For each receipt item:

```text
SUM(active allocations) <= item total
```

or the equivalent percentage constraint:

```text
SUM(active allocation percentages) <= 100%
```

depending on the approved internal allocation model.

An allocation that would exceed the available portion must fail.

---

## 28. Concurrent Claims

Multiple participants may attempt to claim the same remaining portion of an item at nearly the same time.

The Receipt domain must prevent the receipt from entering an invalid overallocated state.

Example:

```text
Remaining Pizza = 25%

Ryan attempts +25%
Alex attempts +25%
```

Only valid allocation state may be committed.

Concurrency enforcement must occur server-side and/or through database constraints/transactions.

Client-side checks alone are insufficient.

---

## 29. Allocation Rounding

Any allocation involving percentages or equal division must eventually produce exact monetary amounts.

The system must define deterministic rounding behavior.

For example, when one cent cannot be evenly divided, the remainder must be assigned consistently rather than lost or duplicated.

Exact rounding rules will be determined during financial-model design.

---

## 30. Unclaimed Amounts

A receipt may temporarily contain unclaimed items or partially unclaimed items during the claiming process.

The application should clearly represent remaining unclaimed value.

A receipt cannot be finalized while required receipt value remains unresolved unless the owner explicitly assigns or resolves it.

---

## 31. Participant Submission

Claiming and submission should be separate concepts.

A participant may make allocation selections before submitting.

When satisfied, the participant explicitly submits their selection.

Conceptually:

```text
Select Items
     ↓
Review Selection
     ↓
Submit
```

---

## 32. Submitted Participant State

Once a participant submits:

- Their submitted state must be preserved
- They should not casually modify it without an approved workflow
- The owner can review their submission
- The system must preserve what they originally submitted

This reduces accidental changes after participants believe they are finished.

---

## 33. Owner Review

The receipt owner should be able to review participant submissions before finalization.

The owner should be able to identify:

- Submitted participants
- Participants who have not submitted
- Unclaimed amounts
- Overlap/invalid states
- Each participant's calculated subtotal
- Tax/tip allocation
- Final amount expected from each participant

---

## 34. Owner Corrections

The owner may correct allocations when necessary.

Examples include:

- Participant selected wrong item
- Participant forgot an item
- Shared percentage was wrong
- Owner manually assigns an unclaimed item

Owner corrections must not silently erase the participant's original submission history.

---

## 35. Reconfirmation

If the owner makes a material change affecting a participant after that participant submitted, the affected participant should require reconfirmation.

Conceptually:

```text
Participant Submits
        ↓
Owner Changes Participant Allocation
        ↓
RECONFIRMATION_REQUIRED
        ↓
Participant Reviews Change
        ↓
Participant Confirms
```

Unaffected participants should not necessarily need to reconfirm.

---

## 36. Reconfirmation Access

Reconfirmation may use the existing temporary receipt flow when still active or another approved temporary confirmation link.

Reconfirmation may remain available longer than the original table-side claiming window where appropriate.

Exact expiration duration can be determined later.

---

## 37. Audit History

Important receipt changes must be auditable.

Audit history should preserve relevant information such as:

- Action
- Actor
- Timestamp
- Previous value
- New value
- Affected participant
- Optional reason

Examples include:

- Participant submission
- Owner allocation correction
- Tax/tip change after submission
- Receipt reset
- Finalization
- Manual payment status change

---

## 38. Owner Changes Must Be Attributable

Owner-originated changes should be distinguishable from participant-originated changes.

Conceptually:

```text
Actor Type

OWNER
PARTICIPANT
SYSTEM
```

Exact event model will be determined later.

---

## 39. Tax Allocation

MVP should support at least two owner-selectable tax allocation methods:

### Proportional

Tax is allocated according to each participant's share of the applicable receipt subtotal.

### Even

Tax is divided evenly among participating people.

---

## 40. Tip Allocation

MVP should support at least two owner-selectable tip allocation methods:

### Proportional

Tip is allocated according to each participant's share of the applicable receipt subtotal.

### Even

Tip is divided evenly among participating people.

---

## 41. Tax and Tip Rounding

Tax and tip allocations must use deterministic exact rounding.

The sum of allocated tax must equal the final receipt tax.

The sum of allocated tip must equal the final receipt tip.

No cents may disappear or be created due to rounding.

---

## 42. Discounts

Receipt discounts, coupons, or adjustments may exist.

MVP should preserve detected/corrected discount information where needed for receipt reconciliation.

A fully general discount-allocation engine is not required unless necessary for MVP correctness.

Any unsupported discount behavior should be clearly represented rather than silently producing incorrect participant totals.

---

## 43. Included Gratuity

A receipt may already include gratuity or a service charge.

The Receipt domain must avoid unintentionally charging tip twice when the receipt indicates an included gratuity.

Detailed gratuity classification may initially require owner verification.

---

## 44. Quantity-Based Items

A receipt may contain a quantity greater than one.

Example:

```text
3 × Taco @ $4.00
Total = $12.00
```

The owner must be able to correct quantity information.

The allocation model may treat this as:

- One $12 line item
- Three logical units

depending on the approved implementation.

The implementation must not produce incorrect totals.

---

## 45. Receipt State Machine

Receipt lifecycle must be represented explicitly.

Initial MVP lifecycle:

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

If the owner modifies submitted allocations:

```text
AWAITING_CONFIRMATION
        ↓
Owner Edit
        ↓
RECONFIRMATION_REQUIRED
```

Exact database status names may differ, but equivalent state transitions must be enforced.

---

## 46. State Transition Validation

Receipt states must not be changed arbitrarily.

Examples:

- A receipt should not enter `CLAIMING` before owner review.
- A receipt should not enter `FINALIZED` while unresolved required confirmations remain.
- A finalized receipt should not silently return to active claiming.
- A processing receipt should not accept participant claims.

State transitions must be validated by the Receipt domain.

---

## 47. Receipt Locking After Claiming Begins

Once participant claiming begins, the underlying verified receipt should be treated as locked for normal editing.

This prevents participants from allocating against a financial structure that changes unexpectedly underneath them.

---

## 48. Receipt Reset

If the owner discovers a serious OCR or receipt-structure error after claiming begins, the owner may perform an explicit receipt reset.

Reset is an exceptional action.

A reset may:

- Return the receipt to review
- Invalidate incompatible active claims
- Require participants to re-enter/reconfirm later
- Preserve historical activity

The system must not silently rewrite a claimed receipt.

---

## 49. Reset Auditability

A receipt reset must preserve an audit record indicating:

- Who initiated the reset
- When it occurred
- Previous receipt revision/state
- Resulting receipt revision/state

Claims invalidated by reset should remain historically traceable where practical.

---

## 50. Receipt Revision

The implementation should support the concept that a materially edited receipt may represent a new revision of the receipt structure.

Exact revision implementation is not yet fixed.

The purpose is to distinguish:

```text
Receipt Version Used for Claims
```

from:

```text
Corrected Receipt Version
```

when substantial changes occur.

---

## 51. Finalization Requirements

The owner may finalize a receipt only when the receipt satisfies required validity conditions.

At minimum:

- Receipt has been reviewed
- Required item value has been allocated
- Allocation constraints are valid
- Required participant submissions are resolved
- Required reconfirmations are complete
- Tax/tip allocations are resolved
- Final participant totals can be calculated exactly

---

## 52. Finalized Receipt

Finalization creates the authoritative final split.

Once finalized:

- Final participant totals are frozen
- Active claim links should no longer permit normal modifications
- Final allocations should not silently mutate
- Final result should remain viewable according to authorization rules
- Receivables may be created

Corrections after finalization require an explicit future correction workflow rather than silent mutation.

---

## 53. Final Participant View

After finalization, a participant should be able to view a summary of what they owe.

Conceptually:

```text
Ryan

Burger              $14.00
Pizza share          $6.00
Subtotal             $20.00
Tax                   $1.34
Tip                   $4.00
---------------------------
Amount owed          $25.34
```

This final view acts as a receipt-like explanation of the participant's obligation.

---

## 54. Receivables

Finalization should create receivable records representing money owed back to the receipt owner.

Conceptually:

```text
Receipt
   ↓
Final Participant Amount
   ↓
Receivable
```

A receivable may include:

- Receipt
- Participant
- Owner
- Amount owed
- Payment status
- Created timestamp
- Paid timestamp where applicable

Exact schema will be determined later.

---

## 55. Manual Payment Tracking

MVP payment tracking is manual.

The receipt owner must be able to mark a receivable as paid.

Conceptually:

```text
OWED
 ↓
Owner marks paid
 ↓
PAID
```

---

## 56. Payment Status

MVP should support at least:

- `OWED`
- `PAID`

Additional states such as partial payment may be introduced later if explicitly required.

---

## 57. Payment Automation Deferred

MVP does not automatically verify whether a Venmo, Zelle, bank transfer, or other payment satisfies a receivable.

Automated payment matching is deferred.

The manual paid action is authoritative for MVP.

---

## 58. Participant Payment Accounts

Participants are not required to connect bank accounts, Venmo, Zelle, or any other payment provider for MVP.

Vantage-Point does not initiate reimbursement transfers during MVP 1.

---

## 59. Duplicate Receipt Upload

Users may accidentally upload the same receipt more than once.

A complete duplicate-detection system is not required for initial MVP.

The architecture should not prevent future duplicate-receipt detection.

---

## 60. Participant Disconnects

A participant may close their browser or lose connectivity before submission.

Temporary allocation state should remain recoverable according to the approved persistence strategy.

A participant disconnecting must not corrupt receipt state.

---

## 61. Participant Never Responds

If a participant never submits or reconfirms, the owner must still have a recovery path.

The owner may:

- Contact the participant again
- Extend/recreate temporary access
- Correct/assign the missing portion manually
- Complete the receipt according to approved owner controls

The receipt should not become permanently unusable because one participant disappeared.

---

## 62. Invalid / Leaked Link

Temporary receipt links are bearer-style access mechanisms for a limited receipt session.

If a link is suspected to be compromised, the owner should eventually be able to invalidate or replace the session token.

At minimum, session expiry and receipt-scoped access limit the exposure.

Advanced participant authentication is deferred.

---

## 63. Authorization

Authenticated receipt-owner APIs must verify ownership.

Participant APIs must verify valid receipt-session authorization and participant context.

The system must prevent:

- One owner reading another owner's private receipt
- One owner modifying another owner's receipt
- Expired participant sessions modifying receipt state
- Participants accessing unrelated receipts
- Participants claiming as arbitrary authenticated users through client manipulation

---

## 64. Realtime Authorization

Realtime subscriptions must not create a bypass around normal authorization.

A participant should receive only receipt-session data that they are authorized to access.

An authenticated owner should receive only receipt data they are authorized to access.

---

## 65. Input Validation

External and participant input must be validated.

Examples include:

- Participant name selection
- Allocation percentage
- Allocation amount
- Receipt item identifiers
- Session tokens
- Owner edits
- Tax/tip selection
- Payment-status updates

Client validation may improve UX but does not replace server validation.

---

## 66. Logging

Receipt logs must avoid unnecessary exposure of:

- Temporary session tokens
- Authentication tokens
- Sensitive payment data
- Full receipt images
- Other secrets

Operational logging may use safe identifiers when needed.

---

## 67. Asynchronous Failure

Vision processing may succeed while a later database operation fails, or vice versa.

Receipt processing must be designed to tolerate retries.

Retries must not create duplicate receipt items or duplicate processing outcomes.

Exact job/idempotency strategy will be defined during implementation design.

---

## 68. Processing Failure

If OCR/vision processing fails:

- The receipt must not disappear
- The owner should see a failure state
- The owner should be able to retry where appropriate
- Existing receipt ownership must remain intact

Manual receipt entry may be introduced later if required.

---

## 69. Receipt Domain API Boundary

The browser should communicate with Vantage-Point Receipt APIs/realtime infrastructure.

Conceptually:

```text
Owner / Participant Browser
          ↓
   Receipt Domain API
          ↓
Validation / Authorization
          ↓
      PostgreSQL
          ↓
Realtime Publication
```

Vision providers should remain behind the Vantage-Point backend/worker boundary.

---

## 70. MVP Owner Experience

The owner should be able to complete this flow:

```text
Sign in with Google
        ↓
Upload Receipt
        ↓
Wait for Processing
        ↓
Review / Correct Receipt
        ↓
Enter Participants
        ↓
Start Split Session
        ↓
Share QR / Link
        ↓
Watch Claims Update
        ↓
Review Submissions
        ↓
Correct if Needed
        ↓
Wait for Required Reconfirmations
        ↓
Finalize
        ↓
See Receivables
        ↓
Mark Receivables Paid
```

---

## 71. MVP Participant Experience

A participant should be able to complete this flow:

```text
Scan QR / Open Link
        ↓
Select Name
        ↓
See Receipt
        ↓
Claim / Split Items
        ↓
See Current Shared State
        ↓
Review Personal Share
        ↓
Submit
        ↓
Reconfirm if Owner Changes It
        ↓
See Final Amount Owed
```

No Vantage-Point account should be required.

---

## 72. MVP Acceptance Requirements

Receipt MVP is functionally complete when an authenticated owner can:

- Upload a receipt
- Process it through OCR/vision
- Review and correct extracted information
- Define participants
- Start a temporary split session
- Generate/share participant access
- Observe realtime participant allocations
- Support full-item and shared-item allocations
- Prevent over-allocation
- Receive participant submissions
- Correct participant allocations with audit history
- Require affected participants to reconfirm material changes
- Allocate tax and tip
- Finalize a valid receipt
- Generate final participant amounts
- Generate receivables
- Manually mark receivables paid

And when a temporary participant can:

- Join without a Vantage-Point account
- Select their receipt-scoped identity
- View current receipt allocation state
- Claim or split items
- Submit their selection
- Reconfirm owner changes when required
- View their final amount owed

And when:

- Receipt owners cannot access other owners' receipts.
- Temporary participants cannot access unrelated receipts.
- OCR output is not automatically authoritative.
- Authoritative money uses exact representations.
- Allocation constraints are enforced server-side.
- Concurrent claims cannot produce over-allocation.
- Important changes remain auditable.
- Finalized receipt state cannot silently mutate.
- Session expiration does not delete receipt history.

---

## 73. Explicitly Deferred Receipt Features

The following are not required for Receipt MVP 1:

- Automated Venmo payment verification
- Automated Zelle payment verification
- Automated bank-transfer matching
- Automatic reimbursement reconciliation
- Sending actual payments
- Participant Vantage-Point accounts
- Strong identity verification for participants
- Advanced fraud prevention
- Complex group permissions
- Fully automated discount allocation for every receipt type
- Complex partial-payment workflows
- Multi-currency receipt splitting
- Automatic duplicate receipt detection
- Advanced receipt search
- Social features
- Debt collection workflows
- Automatic reminders
- Credit-card reward optimization
- Safe-to-spend integration

These require separate approved requirements before implementation.