# Vantage-Point implementation coordination

The owner approved VP-002/VP-003 on 2026-10-11 and authorized dedicated task-branch pushes for subsequent owner PR review. Implementation and independent review use separate Codex subagents. No agent merge or production deployment is authorized.

## Current tasks and gates

| Task | Status | Dependency gate | Implementer | Reviewer |
| --- | --- | --- | --- | --- |
| VP-002 — Local Supabase/migrations/initial CI | COMPLETED, independently APPROVED | VP-001 owner-confirmed complete | Foundation subagent | Separate Integration/Review subagent |
| VP-003 — Auth/profiles | COMPLETED, independently APPROVED | VP-002 COMPLETED | Foundation subagent | Separate Integration/Review subagent |
| VP-004 — Readable/reusable auth UI and modern sign-in | READY; detailed proposal approval pending | VP-003 COMPLETED | Proposed Foundation subagent | Separate Integration/Review subagent |
| VP-005 — Dashboard and MVP navigation | BACKLOG; shell-first selected, detailed approval pending | VP-004 COMPLETED | Proposed Dashboard subagent | Separate Integration/Review subagent |

VP-004/VP-005 are owner-requested follow-up proposals, not implementation assignments. No follow-up implementation subagents have been spawned. VP-004 is dependency-satisfied but awaits approval of its detailed work order. Owner selected VP-005 shell/navigation now, domain data integration later. It remains BACKLOG until VP-004 completes and its detailed work order is approved. Live Banking/Receipt APIs are not shell-task dependencies; subsequent domain-data integration needs separately approved domain and Dashboard work orders. No live domain data, financial statistics, fake money, domain logic or backend aggregation belongs in VP-005.

## Accepted implementation and delivery

- VP-001 scaffold `d5acb7a18aae427c516b919c4cad4bc758bf21da` is complete by owner confirmation, not retrospective independent approval. Its historical task-file reconciliation must not invent test/review evidence.
- VP-002 independent Integration/Review returned APPROVED at `9362d9906282ada561acbcecc414a4ab9d4d7b45`, with local checks and actual push/PR CI; Orchestrator recorded COMPLETED. Owner ryanzhou07 subsequently merged PR #1 at `6c3dbbc745813de7997b065dc66b70f5b2ce0d7b`. VP-002 lifecycle documentation publication head was `7511b1907dbeecedc3f02622dc65f614a41bc6ca`.
- VP-003 source `567894cfec0941cf5def1f68449cc3bae27ff84c`, reviewed pushed head `e84a2fef260ae93b665b6b7ad04ca60ea8cfb2ce`, branch `vp-003-auth-profiles`, draft PR https://github.com/ryanzhou07/Vantage-Point/pull/2. Exact-head push CI https://github.com/ryanzhou07/Vantage-Point/actions/runs/38195757329 and PR CI https://github.com/ryanzhou07/Vantage-Point/actions/runs/38195761017 passed Frontend, Backend, Database and Security.
- Owner performed and attested actual Google sign-in/profile/reload/sign-out/narrow-layout smoke. Browser automation was unavailable; canceled-consent manual observation is not claimed. Automated auth/error coverage, implementation evidence and independent-review verification are distinguished in the completed task and evidence.
- Separate Integration/Review subagent `/root/vp003_review` returned APPROVED at exact `e84a2fef260ae93b665b6b7ad04ca60ea8cfb2ce`, with no blocking findings. Orchestrator recorded VP-003 COMPLETED. Lifecycle/follow-up publication changes documentation only; no auth source changes, self-approval, merge or deployment.

## Baseline and preservation

At initial planning, local main/origin-main pointed to `106328e`; that historical snapshot predates the subsequent owner merge recorded above. The simplified current specifications replaced older detailed designs/agent files in `106328e`; deleted historical requirements are not revived.

Owner DESIGN.md edits remain uncommitted and must be preserved. Initial planning identified AWS hosting edits and older Cloud Run wording in README; these local implementation tasks do not settle deployment decisions or change approved architecture. Primary checkout, local OAuth/frontend configuration and active preview must remain intact.

## Subagent execution protocol

1. Orchestrator verifies dependencies and approved scope, then prepares a dedicated branch/worktree from accepted commits while preserving owner edits/configuration.
2. Assign the designated Foundation, Banking, Receipt or Dashboard subagent the exact VP work order and allowed paths. Implement only that scope, run required checks, record evidence and report READY FOR REVIEW. No self-approval, merge, official completion or production deployment.
3. After implementation finishes, Orchestrator records REVIEW and assigns a separate Integration/Review subagent the exact head/diff, contracts, acceptance criteria, local evidence and hosted CI. Reviewer normally does not fix implementation.
4. CHANGES REQUIRED returns findings to the implementer; corrections require independent re-review and exact-head CI. ESCALATION REQUIRED routes product/architecture decisions to the owner without silently changing requirements.
5. Orchestrator records COMPLETED only after implementation, acceptance, required tests, actual hosted CI, independent APPROVED review and documentation/contracts satisfy AGENTS.md.
6. Avoid concurrent edits; parallelize only independent tasks. VP-004 precedes VP-005 because they share frontend/UI paths and the Dashboard consumes the proposed reusable baseline.

## Delivery and testing

Follow agents/TESTING.md with pinned runtimes and meaningful tests. Push ordinary dedicated task branches; create draft PRs for actual PR CI and owner review. Record exact SHA, run URLs, results and limitations; green CI alone is not independent approval. No force pushes, main pushes, agent merges or deployments. Publication-only final run URLs are reported to the coordinator rather than creating an endless evidence-commit loop.

## Owner-approved security remediation

On 2026-10-11 owner explicitly approved Go1.26.9 and full retesting, plus assessment of the exact unpatched lint-only braces advisory GHSA-vfj7-8cjw-p6xm subject to independent review. ADR-002 records the bounded temporary exception; all other findings remain blocking.

## Resolved worktree cleanup and active preview

Owner requested automatic removal of resolved worktrees. Orchestrator safely removed the completed/pushed VP-002 worktree after confirming its only dirty file was a byte-identical copy of the owner's root DESIGN.md and no local credential files were present. Branches and original files remain.

**Keep VP-003 frontend localhost3000, API localhost8080, local Supabase563xx and its worktree running while the owner inspects the preview.** Resolved cleanup is deferred during inspection. Do not reset the live database, stop services, or run a build over live frontend artifacts. Future cleanup requires committed/preserved work and private preservation of local configuration outside the removed checkout; no force removal or branch deletion. Keep the primary checkout and active worktrees.
