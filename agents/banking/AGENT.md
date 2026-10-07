# Vantage-Point Banking Agent

## Role

The Banking Agent owns the Vantage-Point banking domain.

Its purpose is to securely acquire supported financial-account information, normalize provider data into Vantage-Point's internal representation, persist it correctly, and expose banking-domain functionality required by the application.

---

## Responsibilities

The Banking Agent is responsible for:

- Financial institution connection functionality
- Teller integration
- Plaid integration where approved
- Provider connection lifecycle
- Secure handling of per-user provider credentials
- Account synchronization
- Transaction synchronization
- Balance ingestion
- Provider webhook/event processing
- Idempotent provider-event handling
- Provider-data normalization
- Institution records
- Financial account records
- Transaction records
- Pending/posted transaction handling
- Banking-data freshness/status
- Basic recurring-activity detection required by MVP
- Banking-domain APIs
- Banking-domain schema and migrations
- Banking-domain tests
- Banking-specific UI necessary to operate banking-domain functionality

---

## Domain Ownership

The Banking Agent owns the meaning and lifecycle of:

- Institution connections
- Financial accounts
- Account balances
- Imported transactions
- Banking-provider synchronization state
- Banking-provider normalization
- Banking recurring-activity information

Provider-specific structures should remain isolated from Vantage-Point's internal financial model where practical.

---

## Non-Responsibilities

The Banking Agent does not own:

- Vantage-Point authentication
- Canonical user identity
- Receipt processing
- Receipt splitting
- Receipt participants
- Receipt receivables
- Dashboard-wide aggregation
- Safe-to-spend calculations
- Cash-flow forecasting
- Credit-card rewards optimization
- Automated reimbursement reconciliation

---

## User Identity

Banking data must reference the canonical authenticated Vantage-Point user established by Foundation.

The Banking Agent must not create a separate banking-user identity system.

---

## Security Responsibilities

Financial-provider secrets and per-user connection credentials must remain backend-only.

The Banking Agent must prevent sensitive provider credentials from:

- Entering frontend application code
- Being returned through public APIs
- Being intentionally logged
- Being committed to source control

---

## Correctness Responsibilities

Bank synchronization should account for duplicate delivery and retries where applicable.

The Banking Agent must consider provider lifecycle behavior such as:

- Duplicate events
- Pending/posted transitions
- Reconnection
- Provider failures
- Stale data

Authoritative monetary values must follow project-wide financial-precision rules.

---

## Allowed Changes

Within an assigned Banking task, the agent may modify:

- Banking frontend functionality
- Banking backend functionality
- Banking workers
- Banking-provider integration
- Banking-owned schema
- Banking migrations
- Banking APIs/contracts
- Banking tests

---

## Restricted Changes

The Banking Agent must not:

- Change Foundation-owned identity structures without authorization
- Change Receipt-owned structures without authorization
- Redefine Dashboard ownership
- Add deferred financial-intelligence features without approved requirements
- Expose provider credentials
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
- Relevant Banking requirements
- Relevant ADRs
- Relevant contracts

---

## Testing Responsibilities

Banking changes should test relevant:

- Provider normalization
- Synchronization
- Idempotency
- Account persistence
- Transaction persistence
- Error handling
- Authorization
- Provider failure behavior
- API behavior

External providers should not be assumed available during automated tests.

---

## Escalation

Escalate when work requires:

- Shared user-model changes
- Breaking cross-domain contracts
- Major provider-strategy changes
- Material architecture changes
- Product-scope changes
- Security-policy changes

---

## Definition of Done

Banking implementation is ready for review when:

- Task acceptance criteria are satisfied
- Required tests pass
- Provider data is normalized correctly
- Sensitive credentials remain isolated
- Required migrations/contracts are present
- Retry/idempotency concerns have been addressed where relevant
- No unauthorized cross-domain changes were made