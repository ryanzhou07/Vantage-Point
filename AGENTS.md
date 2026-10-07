# Vantage-Point Agent Rules

## 1. Purpose

This file defines how coding agents work inside Vantage-Point.

All agents must follow these rules.

---

# 2. Authority

Order of authority:

1. Project owner
2. Approved `PROJECT.md`
3. Approved `DESIGN.md`
4. This `AGENTS.md`
5. Assigned `VP-###` task
6. Existing implementation

If requirements conflict or an implementation would require violating a higher-authority source, stop and escalate.

Do not silently reinterpret requirements.

---

# 3. Required Reading

Before implementing a task, an agent must understand:

```text
AGENTS.md
+
relevant PROJECT.md section
+
relevant DESIGN.md section
+
assigned VP-### task
```

Agents should not invent product behavior because a requirement appears incomplete.

---

# 4. Task Requirement

Implementation work must correspond to an approved task under:

```text
agents/tasks/
```

A task defines:

- Scope
- Owner
- Dependencies
- Requirements
- Acceptance criteria
- Testing requirements

The task is the authoritative work order.

---

# 5. Scope

Agents implement only the assigned task.

If an agent notices unrelated improvements, it should report them rather than silently expanding scope.

---

# 6. Agent Roles

Vantage-Point uses:

```text
Orchestrator
Foundation
Banking
Receipts
Dashboard
Integration / Review
```

Feature agents own vertical product slices rather than only frontend or backend layers.

---

# 7. Orchestrator

The Orchestrator:

- Converts approved plans into tasks.
- Tracks dependencies.
- Assigns work.
- Maintains task status.
- Routes work for review.
- Identifies blockers.
- Coordinates agents.

The Orchestrator does not independently change:

- Product requirements
- MVP scope
- Major architecture
- Technology choices
- Domain ownership

The Orchestrator does not self-approve implementation work.

---

# 8. Foundation Agent

Owns:

- Supabase authentication integration
- Google OAuth application integration
- Canonical user identity
- Profiles
- Shared authentication middleware
- Authenticated request context
- Shared identity contracts
- Foundation tests

Does not own:

- Banking business logic
- Receipt business logic
- Dashboard business logic

Receipt participants are not normal Vantage-Point users.

---

# 9. Banking Agent

Owns the full Banking vertical slice:

- Teller integration
- Plaid integration where approved
- Provider adapters
- Institution connections
- Provider credentials handling
- Accounts
- Balances
- Transactions
- Pending→posted reconciliation
- Banking synchronization
- Banking worker
- Provider webhooks
- Provider-event idempotency
- Recurring activity
- Banking APIs
- Banking schema/migrations
- Banking UI
- Banking tests

Does not own:

- Authentication identity
- Receipt splitting
- Dashboard-wide aggregation
- Safe-to-spend
- Forecasting
- Rewards optimization
- Automated reimbursement reconciliation

---

# 10. Receipt Agent

Owns the full Receipt vertical slice:

- Receipt upload
- Receipt image handling
- OCR/vision integration
- Receipt review
- Receipt items
- Participants
- Temporary sessions
- QR/share links
- Realtime collaboration
- Claims/allocations
- Allocation concurrency
- Participant submissions
- Owner corrections
- Reconfirmation
- Audit history
- Receipt lifecycle
- Receipt revisions/reset
- Tax/tip allocation
- Finalization
- Receivables
- Manual paid status
- Receipt APIs
- Receipt schema/migrations
- Receipt UI
- Receipt tests

---

# 11. Dashboard Agent

Owns:

- Authenticated application shell
- Navigation
- Dashboard overview
- Cross-domain presentation
- Responsive/mobile presentation
- Loading states
- Empty states
- Error states
- Presentation aggregation

The Dashboard Agent does not redefine Banking or Receipt financial semantics.

---

# 12. Integration / Review Agent

Independently reviews implementation for:

- Acceptance criteria
- Scope
- Architecture
- Domain ownership
- Contracts
- Database safety
- Authentication/authorization
- Security
- Financial precision
- Concurrency
- Idempotency
- Tests
- Regression risk
- CI results

Possible review results:

```text
APPROVED
CHANGES REQUIRED
ESCALATION REQUIRED
```

The Integration Agent does not normally fix the feature itself.

---

# 13. Domain Ownership

Agents should primarily modify their owned domain paths.

Cross-domain changes require explicit review.

Shared resources require extra care.

Examples:

```text
canonical identity
shared contracts
shared middleware
platform infrastructure
database-wide configuration
CI
```

No agent may casually redefine a shared resource to make its own task easier.

---

# 14. Database Rules

One PostgreSQL/Supabase database is used for MVP.

Agents own migrations for their domains.

Foundation owns profile/shared identity schema.

Banking owns Banking schema.

Receipt owns Receipt schema.

Rules:

1. Create migrations for schema changes.
2. Never casually rewrite an already-applied migration.
3. Do not modify another domain's schema without explicit task authorization.
4. Cross-domain foreign keys/contracts require Integration Review.
5. Use exact integer money.
6. Preserve referential integrity.
7. Important uniqueness/idempotency constraints belong near the database where appropriate.
8. Database migrations must pass CI.

There is no separate Database Agent.

---

# 15. Financial Correctness

Authoritative financial values must not use floating point.

Use integer minor units.

Receipt allocation, finalization, tax/tip distribution, and Banking normalization must follow `DESIGN.md`.

Frontend calculations are never authoritative.

---

# 16. Security

Agents must not:

- Commit secrets.
- Log credentials.
- Expose service-role credentials to browsers.
- Trust frontend user IDs.
- Trust receipt IDs as authorization.
- Disable authorization to make tests pass.
- Store raw provider secrets in frontend code.
- Store raw temporary bearer tokens unnecessarily.

External inputs must be validated and bounded.

---

# 17. Traffic Protection

Agents must consider:

- Rate limiting
- Request deduplication
- Pagination
- Caching
- Timeouts
- Retry bounds
- Backpressure
- Worker concurrency

when implementing relevant endpoints.

Expensive provider operations must not be triggered without bounds by ordinary frontend traffic.

---

# 18. Caching

Caches are performance layers, not financial authorities.

Cache entries containing private financial data must be correctly scoped.

A mutation must validate authoritative state regardless of cached frontend/server state.

---

# 19. Provider Boundaries

Teller/Plaid/model-provider structures should remain behind adapters.

Domain code should consume normalized Vantage-Point types.

Tests should use fake/sandbox providers unless a task explicitly requires real-provider integration testing.

---

# 20. Git Workflow

Normal workflow:

```text
VP task
   ↓
dedicated branch/worktree
   ↓
implementation
   ↓
tests
   ↓
pull request
   ↓
CI
   ↓
Integration Review
   ↓
merge
```

Agents do not push directly to `main`.

Agents do not independently merge their own feature work.

Agents do not deploy production unless explicitly authorized by an approved deployment task.

---

# 21. Branches

One implementation task should normally have one dedicated branch/worktree.

Multiple coding agents should not simultaneously work on the same branch.

Branch naming may use:

```text
vp-###-short-description
```

---

# 22. Testing

Agents must provide tests appropriate to their change.

Relevant categories include:

- Unit
- Integration
- Authorization
- Financial correctness
- Concurrency
- Idempotency
- Rate limiting
- Cache isolation
- Responsive/E2E

Do not add meaningless tests solely to increase test counts.

---

# 23. Local Before CI

Agents should run relevant local checks before requesting review.

CI should automate the same fundamental checks.

Agents may not disable valid CI checks simply to produce a green build.

---

# 24. Architecture Changes

If a task requires changing a major approved architecture decision, stop and escalate.

Examples:

- Separate physical database
- New core framework
- New authentication model
- New canonical identity
- Replacing Teller
- Adding major infrastructure
- Creating a new microservice
- Changing money representation

If approved, record the decision under:

```text
agents/decisions/
```

when long-term architectural memory is useful.

---

# 25. Requirements Changes

Agents do not silently change `PROJECT.md`.

If implementation reveals a product requirement problem, report it.

The project owner approves product changes.

---

# 26. Contract Changes

Breaking API/shared contract changes require Integration Review.

An agent must consider consumers before changing shared contracts.

---

# 27. Task Lifecycle

Tasks use:

```text
BACKLOG
   ↓
READY
   ↓
ACTIVE
   ↓
REVIEW
   ↓
COMPLETED
```

Additional state:

```text
BLOCKED
```

The implementation agent reports work ready for review.

It does not unilaterally declare the official task `COMPLETED`.

---

# 28. Review

The Integration / Review Agent evaluates the task against:

```text
PROJECT.md
DESIGN.md
AGENTS.md
VP task acceptance criteria
CI
```

Green CI is necessary where required but does not automatically imply approval.

---

# 29. Completion

A task is complete when:

- Required implementation exists.
- Acceptance criteria are satisfied.
- Required tests pass.
- CI passes.
- Integration Review approves.
- Required documentation/contracts are updated.
- Orchestrator records completion.

---

# 30. Guiding Principle

Agents should prefer:

```text
small
reviewable
tested
domain-owned
financially correct
```

changes over large speculative rewrites.

Build the approved Vantage-Point MVP.

Do not redesign the entire system while implementing one task.