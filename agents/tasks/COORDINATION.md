# Proposed sequence after VP-001

Approved by the project owner on 2026-10-11. Implementation and review are delegated to separate Codex subagents. The owner authorized task-branch pushes for their subsequent PR review; no merge is authorized.

## Baseline
- Project owner states VP-001 is complete. Treat that as the satisfied scaffold dependency; this is owner confirmation, not retrospective Integration/Review approval.
- Scaffold is committed at `d5acb7a18aae427c516b919c4cad4bc758bf21da` on `vp-001-foundation-scaffold`.
- Local `main` and `origin/main` references point at `106328e`; no fresh remote fetch or hosted PR/CI evidence was obtained in this planning pass.
- Existing VP-001 file remains under `ready/` with unchecked acceptance criteria. Preserve the owner's local edits; its historical status/evidence needs reconciliation without fabricating tests/reviews.
- DESIGN.md has uncommitted AWS hosting edits; preserve them. README still mentions Google Cloud Run. Local tooling tasks may refer readers to DESIGN.md but must not settle deployment choices or alter approved architecture. No deployment work is proposed.
- The simplified current documents replaced older detailed designs/agent files in commit `106328e`. Deleted historical specifications are not revived as approved requirements.

## Proposed tasks and gates
| Task | Status | Dependency gate | Implementer | Reviewer |
| --- | --- | --- | --- | --- |
| VP-002 — Local Supabase/migrations/initial CI | ACTIVE, approved | VP-001 owner-confirmed complete | Foundation subagent | Separate Integration/Review subagent |
| VP-003 — Auth/profiles | BACKLOG | VP-002 COMPLETED | Foundation subagent | Separate Integration/Review subagent |

Local tooling and initial CI form one bounded setup task so there is no circular completion dependency. VP-002 must pass hosted CI and independent review before completion. VP-003 remains BACKLOG until that full gate is satisfied.

## Subagent execution protocol
1. Orchestrator verifies dependencies and approved scope, then prepares a dedicated branch/worktree based on the required accepted commits, preserving local owner edits.
2. Spawn a Foundation implementation subagent with AGENTS.md, relevant PROJECT.md/DESIGN.md sections, the specific VP task, allowed paths, task base, and branch. Instruct it to implement only that scope, run required checks, record evidence, and report READY FOR REVIEW. No self-approval, merge, task completion, or production deployment.
3. Wait for implementation to finish. Move official task state to REVIEW and spawn a separate Integration/Review subagent against the exact task diff, acceptance criteria, contracts, tests, and hosted CI. Review agent normally makes no implementation fixes.
4. CHANGES REQUIRED: send actionable findings to the implementation subagent; after corrections, obtain independent review again. ESCALATION REQUIRED: present the requirement/architecture decision to the project owner without silently changing it.
5. Record COMPLETED only after implementation, acceptance, required tests, hosted CI, independent APPROVED review, and documentation/contracts satisfy AGENTS.md. Record PR/commit/review/CI evidence without claiming unavailable results.
6. Execute these tasks sequentially. They share tooling/configuration/docs/migration paths; implementation and review do not edit them concurrently.

After authentication, propose Dashboard shell and one domain vertical slice using their designated agents and separate review. Do not invent or approve detailed Banking/Receipt work orders prematurely. Parallel feature work follows the initial slice and architectural review described in PROJECT.md §§37–38.

## Delivery and testing
Implementation agents follow `agents/TESTING.md`, push only their dedicated task branches, and provide branch/compare or draft PR links. Push locally checked work to trigger hosted CI, then corrections as necessary; final delivery must identify the remote commit reviewed. No force pushes, main pushes, merges, or deployments. Hosted CI must pass before task completion.
