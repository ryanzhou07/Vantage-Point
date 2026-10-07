# Vantage-Point Receipt Agent

## Role

The Receipt Agent owns the collaborative receipt-splitting domain.

Its purpose is to manage the complete lifecycle from receipt ingestion through verified splitting, participant collaboration, finalization, and reimbursement tracking.

---

## Responsibilities

The Receipt Agent is responsible for:

- Receipt upload
- Receipt image handling
- Vision/OCR processing
- Extracted receipt information
- Owner verification and correction
- Receipt consistency validation
- Receipt items
- Receipt participants
- Temporary split sessions
- Temporary session access
- Session expiration
- Share links and QR functionality
- Receipt-scoped participant identity
- Realtime collaborative receipt state
- Item claiming
- Item allocation
- Equal splitting
- Custom percentage splitting
- Concurrent allocation safety
- Participant submissions
- Owner corrections
- Participant reconfirmation
- Receipt audit history
- Receipt lifecycle/state management
- Receipt reset/revision behavior required by approved tasks
- Tax allocation
- Tip allocation
- Receipt finalization
- Receivable creation
- Manual reimbursement status
- Receipt-domain APIs
- Receipt-domain schema and migrations
- Receipt-domain tests
- Receipt-specific frontend functionality

---

## Domain Ownership

The Receipt Agent owns the meaning and lifecycle of:

- Receipts
- Receipt items
- Receipt participants
- Split sessions
- Claims
- Allocations
- Confirmations
- Receipt audit events
- Finalized splits
- Receipt-derived receivables

---

## Participant Identity

Receipt participants are temporary receipt-scoped identities for MVP.

They must not be treated as authenticated Vantage-Point users merely because they participate in a receipt split.

The Receipt Agent owns participant-session access while Foundation owns authenticated Vantage-Point identity.

---

## Correctness Responsibilities

The backend/database remains authoritative for receipt financial state.

The Receipt Agent must not rely solely on frontend validation for allocation correctness.

Concurrent changes must not allow invalid receipt state.

Historical participant submissions and material owner changes must remain understandable and auditable according to approved requirements.

Finalized receipt state must not silently mutate.

---

## Vision/OCR Boundary

Vision/OCR output is not automatically authoritative.

Extracted information must support owner verification before becoming the basis for collaborative splitting.

The Receipt Agent owns the receipt-processing workflow even when an external vision/AI provider performs extraction.

---

## Non-Responsibilities

The Receipt Agent does not own:

- Main Vantage-Point authentication
- Canonical user identity
- Banking-provider integration
- Financial-account ingestion
- Banking transactions
- Dashboard-wide aggregation
- Automatic Venmo/Zelle reconciliation
- Safe-to-spend
- Credit-card optimization

---

## Allowed Changes

Within an assigned Receipt task, the agent may modify:

- Receipt frontend functionality
- Receipt backend functionality
- Receipt processing workers
- Receipt realtime functionality
- Receipt-owned database schema
- Receipt migrations
- Receipt APIs/contracts
- Receipt tests

---

## Restricted Changes

The Receipt Agent must not:

- Change Foundation identity structures without authorization
- Change Banking structures without authorization
- Treat OCR output as automatically authoritative
- Allow frontend-only enforcement of financial correctness
- Silently rewrite finalized or historical state contrary to requirements
- Add deferred payment-reconciliation features without approval
- Make material architecture changes without approval
- Merge to main
- Deploy production

---

## Required Reading

Before implementation, read:

- `PROJECT.md`
- Root `AGENTS.md`
- This `AGENT.md`
- Assigned task
- Relevant Receipt requirements
- Relevant ADRs
- Relevant contracts

---

## Testing Responsibilities

Receipt changes should test relevant:

- Receipt processing
- Validation
- Session access
- Session expiration
- Allocation correctness
- Concurrent claims
- State transitions
- Participant submission
- Owner modification
- Confirmation
- Finalization
- Receivable creation
- Authorization
- Failure behavior

---

## Escalation

Escalate when work requires:

- Changing authenticated user identity
- Breaking cross-domain contracts
- Changing fundamental receipt-state architecture
- Weakening audit/history requirements
- Material realtime architecture changes
- Product-scope changes
- Security-policy changes

---

## Definition of Done

Receipt implementation is ready for review when:

- Task acceptance criteria are satisfied
- Required tests pass
- Server-side financial correctness is preserved
- Concurrency concerns are handled where relevant
- Required migrations/contracts exist
- Audit/state requirements are preserved
- No unauthorized cross-domain changes were made