# Vantage-Point Agent Governance

## 1. Purpose

This file defines the project-wide rules governing all AI development agents working on Vantage-Point.

Every agent must follow this file in addition to its own `AGENT.md`.

An individual agent's instructions may further restrict its behavior but may not override or weaken the rules in this file.

---

## 2. Authority Hierarchy

The authority hierarchy is:

1. Project owner
2. Approved project requirements and architectural decisions
3. Root `AGENTS.md`
4. Assigned task specification
5. Domain-specific `AGENT.md`
6. Existing implementation

Agents must not reinterpret lower-level information in a way that conflicts with higher-level authority.

When a conflict cannot be resolved from the repository, the issue must be escalated.

---

## 3. Source of Truth

Agents must use repository artifacts as the persistent source of truth.

Important sources include:

- `PROJECT.md`
- Approved requirements
- Architecture Decision Records
- Shared contracts
- Task specifications
- Database migrations
- Tests
- Source code

Agents must not rely on conversational memory when an authoritative repository artifact exists.

---

## 4. Task Requirement

Agents perform implementation work only through defined tasks.

A task must establish:

- Objective
- Owner
- Scope
- Dependencies
- Requirements
- Acceptance criteria
- Testing requirements
- Relevant shared-resource impact

Agents must remain within the assigned task scope.

If additional work is discovered, it must be reported to the Orchestrator rather than silently expanding the task.

---

## 5. Domain Ownership

Vantage-Point uses domain-oriented development ownership.

Primary domains are:

- Foundation
- Banking
- Receipts
- Dashboard

Each domain agent may work across frontend, backend, database, and tests when those changes belong to its domain and task.

Ownership is based on product responsibility rather than technical layer.

An agent must not modify another domain's behavior without explicit authorization.

---

## 6. Shared Resources

Some resources affect multiple domains and therefore require additional coordination.

Shared resources may include:

- User identity
- Authentication infrastructure
- Shared authorization infrastructure
- Shared API contracts
- Shared types
- Common libraries
- Application-wide configuration
- Database-wide conventions
- CI/CD infrastructure
- Deployment infrastructure

A feature agent must not make a breaking shared-resource change without coordination through the Orchestrator and appropriate review.

---

## 7. Database Rules

Database changes must be performed through migrations.

Agents must not silently modify an existing applied migration.

Database objects should have clear domain ownership.

Cross-domain schema changes require coordination.

Authoritative currency calculations must not use floating-point arithmetic.

Financial values must use an exact representation such as integer minor units or another approved exact representation.

Database constraints and transactions should enforce correctness where application-level validation alone is insufficient.

---

## 8. Contract Rules

Interfaces shared between components or domains must have an explicit source of truth.

Agents must not independently create incompatible assumptions about shared APIs, events, or data structures.

Breaking contract changes require coordination with affected domains.

---

## 9. Security Rules

Agents must follow least-privilege principles.

Secrets and credentials must never be:

- Committed to source control
- Exposed to frontend code when backend-only
- Intentionally logged
- Embedded directly in source code

Financial-provider credentials and per-user financial connection credentials must remain backend-only.

Agents must not weaken authentication, authorization, validation, or security controls merely to simplify implementation.

---

## 10. Git Rules

Development work must occur outside the protected main branch.

Tasks should use isolated branches or worktrees where practical.

Agents must not push directly to `main`.

Agents must not rewrite unrelated work.

Agents must keep changes within task scope.

---

## 11. Testing Rules

Agents are responsible for testing work within their domain.

Agents must not:

- Remove valid tests simply to make a task pass
- Disable failing tests without authorization
- Ignore failures affecting their task
- Claim successful completion when required validation has not passed

Relevant tests must accompany behavior changes where appropriate.

---

## 12. Review Rules

Feature agents do not approve their own work.

Implementation readiness and integration approval are separate states.

Completed implementation must be routed through the defined review process.

The Integration / Review Agent may approve work or request changes but does not redefine product requirements.

---

## 13. Architecture Changes

Agents may make routine implementation decisions within established architecture.

Material architectural changes require explicit review.

When a significant architectural change is necessary:

1. Stop the affected work when appropriate.
2. Report the issue to the Orchestrator.
3. Document the proposed decision through the project's architecture-decision process.
4. Obtain required approval.
5. Resume implementation only after the decision is resolved.

Agents must not silently redefine Vantage-Point architecture.

---

## 14. Requirements

Agents implement approved requirements.

Agents must not:

- Add major product functionality without approval
- Remove requirements because implementation is difficult
- Redefine MVP scope
- Treat an implementation assumption as an approved requirement

Requirement ambiguity must be escalated.

---

## 15. Merge Authority

No feature agent may independently merge its own work into the protected main branch.

Passing tests or Integration Review does not itself grant merge authority.

Merge policy is controlled separately by the project owner and repository configuration.

---

## 16. Deployment Authority

Feature agents may not independently deploy Vantage-Point to production.

Production deployment authority is controlled separately from feature implementation.

Agents must not assume that merged code is automatically authorized for production deployment.

---

## 17. Escalation

Agents must escalate when work requires:

- Changing approved product requirements
- Material architecture changes
- Breaking shared contracts
- Unauthorized cross-domain modifications
- Major technology changes
- Security-policy changes
- Unclear ownership
- Conflicting requirements
- Work outside assigned scope

Agents should not guess when the decision materially affects other domains or the product architecture.

---

## 18. Completion Principle

A feature agent declaring implementation complete means only that the implementation is ready for review.

A task becomes officially completed only after the required review and project workflow have been satisfied.