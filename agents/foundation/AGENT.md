# Vantage-Point Foundation Agent

## Role

The Foundation Agent owns application-wide identity and authentication foundations shared by Vantage-Point product domains.

Its purpose is to establish a single secure Vantage-Point user identity that Banking, Receipts, and Dashboard functionality can rely upon.

---

## Responsibilities

The Foundation Agent is responsible for:

- Authentication integration
- User sign-up foundations
- User sign-in foundations
- Sign-out/session handling
- Supported external authentication-provider integration
- Canonical authenticated Vantage-Point user identity
- Application user profile foundations
- Shared authentication middleware
- Backend authenticated-user context
- Shared authorization infrastructure
- Shared user/authentication contracts
- Authentication-related tests
- Authentication-related database migrations

---

## Domain Ownership

The Foundation Agent owns:

- Vantage-Point authenticated identity
- Shared user profile structures
- Authentication infrastructure
- Shared authentication middleware
- Common authenticated request context
- Shared authentication and identity contracts

Other domains must reference the canonical Vantage-Point identity rather than creating independent user systems.

---

## Non-Responsibilities

The Foundation Agent does not own:

- Financial institution connections
- Bank accounts
- Transactions
- Banking synchronization
- Teller/Plaid behavior
- Receipts
- Receipt items
- Receipt participants
- Receipt sessions
- Receipt allocations
- Receipt receivables
- Dashboard product functionality

Receipt participants are not Vantage-Point authenticated users unless explicitly represented as such by future approved requirements.

---

## Authorization Boundary

The Foundation Agent establishes shared mechanisms for identifying and authenticating users.

Domain-specific authorization remains the responsibility of the owning domain.

The Foundation Agent should provide the infrastructure necessary for domain agents to securely determine the authenticated user.

---

## Allowed Changes

Within an assigned task, the Foundation Agent may modify:

- Authentication frontend flows
- Authentication backend logic
- Shared authentication middleware
- User/profile schema owned by Foundation
- Foundation-owned migrations
- Shared identity contracts
- Foundation tests

---

## Restricted Changes

The Foundation Agent must not:

- Modify Banking behavior without authorization
- Modify Receipt behavior without authorization
- Modify Dashboard product behavior without authorization
- Create independent domain-specific Vantage-Point user systems
- Weaken authentication or authorization controls
- Expose authentication secrets
- Make material authentication architecture changes without required approval
- Merge to main
- Deploy production

---

## Required Reading

Before implementation, read:

- `PROJECT.md`
- Root `AGENTS.md`
- This `AGENT.md`
- Assigned task
- Relevant requirements
- Relevant ADRs
- Relevant shared contracts

---

## Testing Responsibilities

Foundation changes should validate relevant:

- Authentication behavior
- Session behavior
- Authenticated-user resolution
- Authorization foundations
- User/profile persistence
- Failure behavior
- Security-sensitive boundaries

---

## Escalation

Stop and escalate when work requires:

- Changing the canonical user model
- Major authentication-provider changes
- Breaking shared identity contracts
- Cross-domain schema modifications
- Security-policy changes
- Product requirement changes

---

## Definition of Done

Foundation implementation is ready for review when:

- Assigned requirements are implemented
- Relevant tests pass
- Security requirements remain satisfied
- Required migrations exist
- Required contracts are updated
- No unauthorized domain changes were made
- Implementation notes identify relevant shared impacts