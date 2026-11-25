# VP-018 — Plaid Banking provider (replaces Teller)

## Status
`BACKLOG` (owner decision 2026-10-28, ADR-011; milestones below are serial; each milestone is its own branch/PR and its own REVIEW; split into VP-018a..d task files when the Orchestrator makes them READY)

## Owner
Banking Agent (backend, schema, adapter); Dashboard Agent (Link launcher UI, milestone d only). Independent Integration/Review Agent. No self-approval.

## Purpose
Teller is discontinued (per owner). Deliver the Banking connection lifecycle on Plaid: a signed-in user links an institution through Plaid Link, the backend stores the encrypted access token, lists connections, and disconnects. Accounts/balances/transactions sync stays in VP-011 (re-scoped to Plaid after this task).

## Required reading
AGENTS.md (esp. §14 DB, §16 security, §19 provider boundaries, §24, §31), ADR-009 (secrets and AES-256-GCM encryption, approved), ADR-011, ADR-010, DESIGN.md Banking sections, PROJECT.md Banking sections, agents/contracts/FOUNDATION_AUTH.md, the code from the merged VP-009a (`services/backend/internal/banking`: `SecretSource`, `Seal`/`Open`), and the open PRs listed in the handoff section.

## VERIFY AGAINST OFFICIAL DOCS FIRST
The owner already caught the Teller work being written without reading provider docs. Before writing code, fetch and read these from plaid.com/docs (do not rely on memory; the endpoint names below are the author's recollection and unverified):
- Link overview and web integration (`/link/token/create`, the Link script/`react-plaid-link`, `onSuccess`/`onExit`).
- Token exchange (`/item/public_token/exchange`), `/item/remove`, `/accounts/get`.
- Sandbox: `/sandbox/public_token/create`, test credentials, test institutions.
- Webhooks and verification (`/webhook_verification_key/get`, `Plaid-Verification` JWT header).
- Trial plan page, Link customization "use case" requirement, error handling and `ITEM_LOGIN_REQUIRED`/update mode.
- Official Go client (`github.com/plaid/plaid-go`) current version; pin an exact version.
Record anything that contradicts this file in the task report; docs win.

## Owner decisions already made
- Provider is Plaid; start on the Trial plan (US/Canada, real production data free, **max 10 Production Items; /item/remove does not restore the cap**), tests and CI use Sandbox only.
- The shared encryption/secret layer from VP-009a stays (rename Teller-flavoured names to provider-neutral ones, e.g. `BANKING_SECRET_SOURCE`, secret names `vantage-point/<env>/plaid-*`; keep the existing local-file layout and permission checks).
- ADR-009 approved (AWS Secrets Manager dependency allowed).

## Milestones

### a — Neutral secrets and Plaid config (Banking)
- Rename Teller-specific config and secret names to provider-neutral/Plaid names, keeping behavior and tests of VP-009a.
- Config: `PLAID_ENV=sandbox|production` (Plaid names; Sandbox default; fail closed on unknown), `PLAID_CLIENT_ID`, `PLAID_SECRET` (from `SecretSource`, never env in production), `TOKEN_ENCRYPTION_KEY` unchanged.
- Fail closed at startup when Production is selected without credentials. Never log secrets.
- Acceptance: existing 009a tests pass renamed; missing credentials fail closed; no secret in errors.

### b — Plaid adapter (Banking)
- A Banking-owned provider interface returning normalized Vantage-Point types; Plaid structs never leave the adapter (AGENTS §19).
- Operations: create link token (user id from Foundation context as `client_user_id`, products `transactions`, country US, redirect handling only if OAuth requires it), exchange public token, get item/institution name, remove item.
- HTTP client with timeouts, bounded retries with jitter, concurrency cap, response size limit. Map Plaid errors to a small typed set with no tokens in messages.
- Tests: fake Plaid server (httptest), error mapping, retry/timeout bounds; no real Plaid call in CI. A manual Sandbox check is evidence only.

### c — Connection lifecycle API and schema (Banking)
- Rework VP-010 from PR #19 on top of `main` (do not merge the Teller PR stack). Keep `institution_connections` (`provider='plaid'`, `provider_connection_id` = Plaid `item_id`, encrypted credential + key version, RLS `user_id = auth.uid()`, unique on user+provider+item). Add a new migration; never edit an applied migration.
- Endpoints (user id from verified context only, never the request body): `POST /api/v1/banking/link-token` (rate limited, bounded), `POST /api/v1/banking/connections` (body: public_token + metadata; backend exchanges, encrypts with AAD user+connection, idempotent on item id; 201 new / 200 existing), `GET` list, `DELETE` disconnect (call `/item/remove`; on provider failure return 502 and keep the credential so retry works, as in the Teller version).
- Credential column excluded from browser-role grants; writes via service-role functions (as in PR #19); access token never returned or logged.
- Tests: two-user isolation at API and database level, idempotency, redaction, rate limit, disconnect failure/retry, migration on clean database.

### d — Dashboard Link launcher (Dashboard)
- Rework VP-017 from PR #22 on top of `main`: Settings > Plugins/Connections "Connect a bank" requests a link token from the API, opens Plaid Link, and posts the returned `public_token` to the connections endpoint. Keep loading/empty/error/cancelled/duplicate/disconnect states and the honest copy.
- Load the Plaid Link script only on click, no localStorage, never keep the public token longer than one request. Use the ui-ux-pro-max skill before writing components; reuse components/ui; mobile-first and accessible.
- Tests: mock the Link global and the API; no real bank.

## Out of scope
Accounts, balances, transactions and sync (VP-011 re-scoped for `/transactions/sync` afterwards), webhooks (a later task: signature verification, idempotent events), update-mode re-link, Plaid payments or Auth/Identity products, recurring detection, multi-region.

## Dependencies
VP-009a merged (PR #15). ADR-011 merged. Owner has a Plaid Dashboard account, Trial plan accepted, Link use case selected, and `client_id`/Sandbox `secret` in the private config (never pasted into chat).

## Requirements
1. Plaid `access_token`s are encrypted at rest with AAD (user id + connection id); only the backend ever holds them.
2. `user_id` comes from the Foundation context; receipt/other-domain code is untouched.
3. Money is not handled here; later amounts use integer minor units and follow DESIGN.md.
4. Expensive provider calls are rate limited and bounded; ordinary page loads never call Plaid.
5. Trial cap awareness: Sandbox for all automated tests; document the 10-Item cap in the Banking README and avoid creating Production Items in tests.

## Acceptance Criteria
- [ ] Each milestone's tests pass locally, then CI.
- [ ] Sandbox end to end by the owner: sign in, Connect, log in with Plaid Sandbox credentials, see the connection, disconnect.
- [ ] No Teller code remains in the new path; obsolete Teller-only files (mutual-TLS client, probe) are not merged.
- [ ] Two-user isolation, idempotency, redaction, rate limit tests pass.
- [ ] Banking README documents setup, env names, the Trial cap and how to remove the probe-free path.

## Testing Requirements
Targeted Go tests for `internal/banking` with `-race`, `go vet`, secret scan of each diff, frontend `tsc`/`eslint`/`vitest` for the launcher. Use Node 22 (see handoff). Migration check against the local Supabase stack once Docker is running.

## Shared Resource Impact
Canonical identity: reads only. Cross-domain foreign key to `profiles(id)` (already in PR #19) requires Integration Review. Infrastructure: AWS Secrets Manager already approved (ADR-009). New dependency `plaid-go` needs an exact pin and a note in the PR.

## Database Impact
NEW MIGRATION (milestone c).

## Contract Impact
NEW CONTRACT (Banking connection API; document in `agents/contracts/BANKING_CONNECTIONS.md`). Supersedes the Teller enroll contract in PR #19.

## Security Considerations
Highest sensitivity. Least-privilege IAM (GetSecretValue on the prefix), redact `client_id`/`secret`/tokens everywhere, public tokens are single-use and short-lived, link tokens are per user and short-lived, webhooks (later) must verify signatures.

## Financial Correctness Considerations
Not applicable in this task.

## Blockers / open questions for the owner
1. Plaid Dashboard account created, Trial plan approved, Link use case chosen, Sandbox and Production keys obtained.
2. AWS region and production secret names (placeholder `vantage-point/<env>/plaid-*`).
3. OAuth institutions (Chase, Bank of America, Wells Fargo) may require redirect URI registration and extra approval; decide whether MVP needs them.
4. Approve ADR-010 (import-boundary strategy) if still PROPOSED.
5. Which Plaid products to request at Link time (`transactions` is the minimum for VP-011; each product may bill after the Trial).

## Session handoff (written 2026-10-28; read before starting)

### What exists (all PRs at github.com/ryanzhou07/Vantage-Point)
Banking stack (stacked, base to top): #15 VP-009a secrets/encryption (KEEP, merge first, includes ADR-009 APPROVED commit) → #16 VP-009b Teller mTLS client (OBSOLETE, close) → #17 VP-009c Teller probe (OBSOLETE, close) → #19 VP-010 connections API/migration `20261028000100_banking_connections.sql` + test `030_banking_connections.sql` (REWORK for Plaid) → #22 VP-017 Dashboard Connect UI in Settings (REWORK for Plaid Link).
Receipt stack (independent of Banking, unaffected by the Plaid switch): #18 VP-015a split sessions + PROJECT.md §17 5-minute edit + the three Receipt task files → #20 VP-015b QR/join/Realtime → #21 VP-016 receipts list. Review and merge from the bottom.
All of these are REVIEW, none COMPLETED. The owner reviews and merges; agents never self-approve or merge.

### Uncommitted state in the main worktree
`.worktrees/main` (the folder where this session ran) holds uncommitted copies of: `agents/decisions/ADR-011-replace-teller-with-plaid.md` (needs its own small PR), this file, and duplicates of the Receipt task files and the `PROJECT.md` edit that are already in PR #18. Discard the duplicates (`git checkout` / remove) instead of committing them, or they will conflict when #18 merges. The primary checkout (repo root) is on `vp-001-foundation-scaffold` with many untracked files; do not disturb it. `.worktrees/main` was detached on request so the root could check out `main`; reattach before committing there.

### Owner decisions recorded outside the code
- 5-minute split-session TTL (done in #18) and Supabase Realtime for live state.
- Realtime channel `split:<session id>` stays public (payload-free). A split session may start only for a receipt that is in the confirmed/submitted state after the owner confirms it. NOT implemented yet; PR #18/#20 allow any status and show the panel only for NEEDS_REVIEW. Needs a small Receipt follow-up task.
- Unanswered: trusted proxy header for rate limiting behind the AWS load balancer (rate-limit state is per instance); whether Connect `products` stays minimal.
- Receipt `outstanding_minor` is always 0 until a finalization/receivable task exists; no such task is defined yet.

### Environment and tooling gotchas
- Use Node 22: `/opt/homebrew/opt/node@22/bin/node`; install with `COREPACK_INTEGRITY_KEYS=0 pnpm install` (default Node 20 and the corepack signature check fail here).
- Docker was not running, so `pnpm db:validate` never ran; migrations were checked only on a throwaway Postgres 14 with stubbed Supabase roles. CI is the real gate; start Docker and run it before merging migrations.
- Private config lives in ignored `.env.*.local` files at the stable repo root (`.env.receipts.local` has `GEMINI_API_KEY`; `.env.oauth.local` has Google OAuth; no Teller or Plaid values exist). Read key names only, never values. Repo helpers: `scripts/` private-config readers and `scripts/local-db.sh`.
- `gitleaks` is not installed; agents used a grep-based secret scan.
- `gh` works for PRs. Attribution lines for commits/PRs are specified by the session system reminder.

### Working agreements
- AGENTS.md §31 applies: small atomic tasks, targeted tests, silent execution, batched reads, targeted edits.
- Task folders are inconsistent on `main` (for example VP-004 is in both `active/` and `review/`; VP-005 sits in `active/` though COORDINATION.md says BACKLOG). The Orchestrator owns status; do not fix these opportunistically.
- Owner prefers: read provider docs before claiming a provider works, short reports, no unverified claims, and no merging without their review.
- Teller facts verified from docs before the shutdown notice: Sandbox needed no certificate; Connect was the only way to get a token. Treat as historical.
