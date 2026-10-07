# Vantage-Point Task Template

## Task ID

`VP-XXX`

Use a unique sequential task ID.

Example:

`VP-001`

---

## Title

Provide a short, specific task title.

Example:

`Design initial banking database schema`

---

## Status

Choose one:

- `BACKLOG`
- `READY`
- `ACTIVE`
- `REVIEW`
- `BLOCKED`
- `COMPLETED`

The Orchestrator controls the official task status.

---

## Owner

Specify the agent responsible for implementing this task.

Choose one when possible:

- `Foundation Agent`
- `Banking Agent`
- `Receipt Agent`
- `Dashboard Agent`

The Integration / Review Agent should not normally be assigned as the implementation owner.

---

## Purpose

Explain why this task exists.

Describe the problem being solved and how it contributes to Vantage-Point.

Keep this focused on the goal rather than implementation details.

---

## Scope

Define exactly what the assigned agent is allowed and expected to change.

Include relevant:

- Features
- Modules
- Database objects
- APIs
- Frontend components
- Workers
- Tests
- Documentation

Example:

```text
This task includes:

- Define the institution_connections table
- Define the accounts table
- Add required relationships
- Add migration files
- Add schema-level tests
```

---

## Out of Scope

Explicitly identify related work that must not be included in this task.

Example:

```text
This task does NOT include:

- Teller API integration
- Plaid API integration
- Transaction ingestion
- Dashboard UI
- Recurring payment detection
```

Agents must not expand task scope without approval.

---

## Dependencies

List anything that must be completed or decided before this task can be implemented.

Dependencies may include:

- Other tasks
- Architecture decisions
- Contracts
- Database decisions
- Provider decisions
- Foundation functionality

Example:

```text
Depends on:

- VP-001 — Define canonical Vantage-Point user identity
- ADR-002 — PostgreSQL database structure
```

If there are no dependencies, write:

`None`

---

## Requirements

List the exact behavior or implementation requirements the task must satisfy.

Each requirement should be clear enough to verify.

Example:

```text
1. Every institution connection must belong to one Vantage-Point user.

2. One institution connection may contain multiple financial accounts.

3. Provider-specific identifiers must be stored separately from internal IDs.

4. Provider credentials must never be returned through frontend-facing APIs.
```

Do not introduce new product requirements inside a task unless they have already been approved.

---

## Acceptance Criteria

Define objective conditions that determine whether the task is complete.

Use checkboxes.

Example:

```text
- [ ] Required database tables exist.
- [ ] Foreign-key relationships are defined.
- [ ] Migration applies successfully.
- [ ] Migration can be tested in the local environment.
- [ ] No Foundation-owned tables were modified.
- [ ] Required tests pass.
```

Acceptance criteria should describe observable outcomes rather than vague goals.

Bad:

```text
- [ ] Database works well.
```

Better:

```text
- [ ] Attempting to create an account with a nonexistent institution connection fails.
```

---

## Testing Requirements

Specify the testing expected for this task.

Examples:

- Unit tests
- Integration tests
- Migration validation
- API tests
- Authorization tests
- Concurrency tests
- Manual verification

Example:

```text
Required:

- Migration validation
- Foreign-key constraint tests
- Ownership relationship tests
```

If a type of automated test is not appropriate yet, state why rather than silently omitting testing.

---

## Shared Resource Impact

Indicate whether this task affects shared project resources.

### Authentication

`YES / NO`

Details:

### Canonical User Identity

`YES / NO`

Details:

### Shared Contracts

`YES / NO`

Details:

### Shared Database Resources

`YES / NO`

Details:

### Infrastructure

`YES / NO`

Details:

### Other Domains

`YES / NO`

Affected domain(s):

Any `YES` response should be reviewed for cross-domain coordination before implementation.

---

## Database Impact

Choose one:

- `NONE`
- `NEW MIGRATION`
- `MODIFIES DOMAIN SCHEMA`
- `CROSS-DOMAIN CHANGE`

Describe affected database objects.

Example:

```text
New tables:

- institution_connections
- accounts

New relationships:

- institution_connections.user_id -> profiles.id
- accounts.institution_connection_id -> institution_connections.id
```

Applied migrations must not be rewritten.

---

## Contract Impact

Choose one:

- `NONE`
- `NEW CONTRACT`
- `NON-BREAKING CHANGE`
- `BREAKING CHANGE`

Describe any affected:

- API request/response structures
- Events
- Realtime messages
- Shared types
- Domain interfaces

Breaking changes require coordination before implementation.

---

## Security Considerations

Document security-sensitive concerns relevant to the task.

Consider:

- Authentication
- Authorization
- Secrets
- Financial credentials
- Sensitive user data
- Temporary tokens
- Input validation
- Logging
- Least privilege

Example:

```text
Provider access credentials must remain backend-only
and must not appear in logs or API responses.
```

If there are no special security considerations beyond normal project rules, write:

`No additional security considerations identified.`

---

## Financial Correctness Considerations

Complete this section when the task touches financial state.

Consider:

- Money representation
- Rounding
- Idempotency
- Duplicate events
- Transactions
- Allocation constraints
- Concurrency
- Finalized state
- Auditability

Example:

```text
Account balances must use the project's approved exact
currency representation and must not use floating-point
values for authoritative calculations.
```

If the task does not involve financial correctness, write:

`Not applicable.`

---

## Implementation Notes

Filled in by the assigned implementation agent.

Document:

- Important implementation choices
- Files/modules changed
- Migrations created
- Tests added
- Assumptions made within existing requirements
- Known limitations
- Follow-up work discovered

Do not use this section to silently redefine the task.

---

## Blockers

List anything currently preventing completion.

Example:

```text
BLOCKED: Provider credential storage approach has not
yet been approved.
```

If none:

`None`

When a blocker prevents further work, the implementation agent must report it to the Orchestrator.

---

## Implementation Result

Filled in when implementation is ready for review.

### Branch

`branch-name`

### Pull Request

`PR link or identifier`

### Commit

`commit SHA if applicable`

### Implementation Status

Choose:

- `READY FOR REVIEW`
- `BLOCKED`

### Summary

Briefly describe what was implemented.

---

## Review

Filled in by the Integration / Review Agent.

### Review Result

Choose one:

- `APPROVED`
- `CHANGES REQUIRED`
- `ESCALATION REQUIRED`

### Acceptance Criteria Review

Record any failed or questionable acceptance criteria.

### Architecture / Ownership Review

Confirm whether domain boundaries and project rules were respected.

### Database Review

Record migration/schema findings if applicable.

### Contract Review

Record compatibility findings if applicable.

### Security Review

Record security findings if applicable.

### Test Review

Record test and CI findings.

### Review Notes

Provide actionable review feedback.

When requesting changes, identify:

1. What is wrong
2. Why it is wrong
3. Which requirement or project rule is affected
4. What must be corrected

---

## Completion

Filled in by the Orchestrator after required review succeeds.

### Final Status

`COMPLETED`

### Completion Notes

Record any relevant final information.

A task must not be marked `COMPLETED` solely because the implementation agent says the work is finished.