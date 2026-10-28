# VP-006 milestone plan

Milestones are sequential within the VP-006 approved launcher scope. Each has its own implementation handoff and a different Integration/Review agent. Keep core new logic for each milestone below 50 lines; split again and return to the Orchestrator if that bound cannot be met without obscuring correctness. Only M1 is READY/ACTIVE now; dependent milestones stay BACKLOG until M1 is independently approved.

## M1 — Private configuration contract (READY → ACTIVE)

Owner: Foundation Agent. Reviewer: separate Integration/Review Agent.

Implement a small configuration module and focused tests for locating one persistent owner-controlled config across worktrees, parsing an allowlisted `KEY=value` file as data (never shell-source it), preserving existing files/permissions, safe import without overwriting, and deriving a strictly allowlisted public web environment. Do not implement service startup, tool installation, or process lifecycle in M1. Do not read or print real credential values in evidence. Synthetic fixtures only.

Local gate: focused config tests, static/lint checks for changed files, and secret scan over changed files. Record branch/head and exact commands. Push the milestone branch for owner PR review, then request independent review. Do not wait for GitHub CI.

## M2 — Pinned tool discovery and first-time setup (BACKLOG)

Dependency: M1 APPROVED. Own only pinned Node/pnpm/Go/Supabase discovery and bounded, checksum-verified local setup guidance/helper. Never silently replace existing tools; never install into system directories. Local tests must cover supported/missing/wrong versions and failure cleanup. Keep core logic under 50 lines; split further if needed.

## M3 — Owned process lifecycle primitives (BACKLOG)

Dependency: M1 APPROVED. Own only private process state, lock/identity verification, safe status/stop, bounded readiness, signal cleanup, stale state and unowned/busy port handling. Never kill an unowned process or stop unrelated Docker. Use disposable fixtures/fake processes. Keep core logic under 50 lines; split further if needed.

## M4 — Thin local service orchestration and docs (BACKLOG)

Dependencies: M2 and M3 APPROVED. Compose config, tools and lifecycle primitives to start local Supabase, API, web, and existing configured workers; offer API-only, status, and data-preserving stop. Verify startup twice against the owner's existing local stack only after a safe plan; never reset it. Preserve/coordinate any already-running environment. Local smoke must verify Auth/API/web health, config persistence and existing synthetic marker/data across restart. Keep orchestration glue under 50 lines; split further if needed.

## Completion

VP-006 completes only when M1–M4 are implemented and independently approved, all task-required local checks pass, setup/run/stop/recovery docs are usable, and the feature branch is pushed for owner review. Owner checks hosted GitHub CI. No live DB reset, agent merge, or production deployment.
