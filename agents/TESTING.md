# Required test guidelines for VP-002 and VP-003

Approved task acceptance criteria remain authoritative. These guidelines specify evidence and execution; they do not change product or architecture requirements.

## Common checks

- Use the repository-pinned Node, pnpm, Go, and Supabase versions. Record actual versions. Do not upgrade a runtime without authorization. On 2026-10-11 the owner explicitly authorized Go 1.26.9 with full retesting to remediate reported standard-library advisories.
- From the worktree root run frozen dependency installation, `pnpm lint`, `pnpm typecheck`, the test command introduced by VP-002, and `pnpm build`.
- From `services/backend` check that `gofmt -l` on tracked Go source is empty, then run `go vet ./...`, `go test -race ./...`, and `go build ./...`. Formatting drift must produce a failing check.
- Use a disposable local database, apply all migrations from zero, and run schema assertions. Never reset a remote database. Commands must fail on migration/assertion errors.
- Run secret scanning with redacted findings and configured dependency scanning. Never commit a test secret or print credentials/tokens in evidence.
- Automated tests use deterministic local/fake providers and synthetic data. No production providers or user financial data.
- Do not add tests that merely mirror implementation. No-test scaffold results must be identified as no coverage, not presented as behavioral verification. Once tests exist, their failures must fail the command and CI.
- Run local equivalents before pushing for hosted CI. A missing tool, skipped test, unexecuted command, or static workflow check is not a passing result.

## VP-002 verification

1. Start the local stack from documented setup, verify availability, stop, and restart it.
2. Run clean migration reset/validation twice; verify local target restrictions.
3. In disposable copies, inject invalid SQL and lint/format failures. Verify the relevant command exits nonzero; restore or remove copies afterwards.
4. Verify secret-scan failure using a synthetic scanner fixture in a disposable location; redact output and do not commit the fixture.
5. Validate workflow syntax, triggers, minimum permissions, timeouts, concurrency, frozen installs, and absence of deployment/production-secret steps.
6. Push only the task branch; obtain actual GitHub Actions runs covering the final code. If PR-triggered checks require a PR, create a draft awaiting owner review. Record run URLs and conclusions.

## VP-003 verification

1. Repeat common checks with real frontend/backend tests now present.
2. Verify signature validation, expiration, issuer/audience, forged user identifiers, missing authentication, and bounded key-fetch/validation failures.
3. On local Supabase test profile provisioning repeatedly and concurrently, canonical foreign keys/uniqueness, and two-user RLS/authorization isolation under realistic database roles. Do not test isolation only with a privileged database role.
4. Test `/api/v1/me` responses, error shape, request context, and cross-user cache isolation.
5. Test auth callback/sign-out/loading/error flows with controlled fixtures; test redirect restrictions and applicable cookie/CORS/CSRF protections.
6. Separately record a sandbox Google OAuth sign-in/callback/sign-out smoke check and mobile/desktop checks. Fakes do not satisfy actual Google smoke verification. Missing sandbox configuration is an explicit blocker for this criterion.
7. Run `pnpm db:validate` (clean-schema/RLS plus 8-transaction provisioning) and `pnpm auth:integration` (actual local Supabase synthetic sessions, two-user RLS and concurrent compiled-API checks). CI Database runs both with pinned Go; no Google secrets are supplied. Run all hosted checks against the final task commit.

## Evidence and review gates

Implementation agents record working directory, command, tool versions, result/exit status, and a concise redacted summary in the task implementation notes or linked test report. Record the exact commit, branch, hosted run URLs, and any limitations/skips. Do not fabricate outcomes or suppress failures.

A separate Integration/Review subagent independently inspects the exact diff and evidence, repeats meaningful checks where available, and reports APPROVED, CHANGES REQUIRED, or ESCALATION REQUIRED. Corrections require independent re-review and CI for the corrected code. Implementation agents never approve their own work or mark tasks officially COMPLETED.

The Orchestrator records COMPLETED only after all task completion requirements pass. Final branch delivery is for the owner's PR review; no merge or deployment is authorized.

## Owner-approved advisory handling — 2026-10-11
The owner authorized assessment/documentation of a temporary exact exception for GHSA-vfj7-8cjw-p6xm in the existing lint-only braces dependency, subject to independent review. Preserve scanning coverage, document the exposure/rationale/revisit trigger, and keep all other findings blocking. This is not permission to suppress other failures.
