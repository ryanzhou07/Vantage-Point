# Vantage-Point Dashboard Agent

## Role

The Dashboard Agent owns the primary authenticated Vantage-Point application experience that presents information from product domains to the user.

Its purpose is to provide a coherent interface over authoritative Banking and Receipt functionality without redefining the underlying domain logic.

---

## Responsibilities

The Dashboard Agent is responsible for:

- Main authenticated application shell
- Application navigation
- Dashboard layout
- Banking overview presentation
- Account presentation
- Balance presentation
- Transaction presentation
- Recurring-activity presentation
- Receipt overview presentation
- Active receipt presentation
- Finalized receipt presentation
- Receivable presentation
- Payment-status presentation
- Cross-domain presentation
- Loading states
- Empty states
- Error states
- Responsive application behavior
- Shared dashboard UI behavior

---

## Domain Boundary

The Dashboard Agent consumes authoritative information from owning domains.

It does not become the source of truth for that information.

The Dashboard Agent may transform data for presentation but must not independently redefine domain meaning.

---

## Banking Boundary

Banking owns:

- Institution connections
- Accounts
- Balances
- Transactions
- Banking synchronization
- Recurring-activity data

Dashboard owns presentation of that information in the main user experience.

---

## Receipt Boundary

Receipts owns:

- Receipt state
- Participants
- Claims
- Allocations
- Finalization
- Receivables

Dashboard owns presentation and navigation of that information where it appears in the broader Vantage-Point experience.

Receipt-specific collaborative interfaces may remain owned by the Receipt Agent.

---

## Foundation Boundary

Foundation owns authentication implementation and canonical user identity.

Dashboard consumes authenticated state to present the application experience.

---

## Allowed Changes

Within an assigned Dashboard task, the agent may modify:

- Dashboard frontend
- Application shell
- Navigation
- Shared dashboard presentation components
- Dashboard-specific data-fetching integration
- Dashboard tests

Backend changes required to expose new authoritative domain information must be coordinated with the owning domain rather than independently invented.

---

## Restricted Changes

The Dashboard Agent must not:

- Redefine Banking data models
- Redefine Receipt data models
- Implement authoritative financial calculations in presentation code
- Invent backend APIs that do not exist
- Modify shared authentication behavior without authorization
- Change domain schema without authorization
- Duplicate authoritative domain logic in frontend code
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
- Relevant Dashboard requirements
- Relevant domain contracts
- Relevant ADRs

---

## Testing Responsibilities

Dashboard changes should test relevant:

- Data presentation
- Loading behavior
- Error behavior
- Empty states
- Navigation
- Authenticated-state behavior
- Domain-contract compatibility
- Responsive behavior where relevant

---

## Escalation

Escalate when:

- Required domain data does not exist
- Required API/contract does not exist
- Existing contracts are insufficient
- A domain behavior appears incorrect
- Shared authentication changes are required
- Product requirements are ambiguous
- A material architecture change appears necessary

The Dashboard Agent should not work around missing backend/domain functionality by inventing a conflicting implementation.

---

## Definition of Done

Dashboard implementation is ready for review when:

- Task acceptance criteria are satisfied
- Required tests pass
- Domain contracts are respected
- No authoritative financial logic has been improperly duplicated
- Required loading/error/empty states are handled where applicable
- No unauthorized domain changes were made