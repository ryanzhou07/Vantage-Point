# Graph Report - main-run  (2026-10-29)

## Corpus Check
- 174 files · ~69,699 words
- Verdict: corpus is large enough that graph structure adds value.
- Unclassified: 29 file(s) not represented in the graph (top: (none) 17, .css 6, .log 2)

## Summary
- 860 nodes · 1779 edges · 50 communities (31 shown, 19 thin omitted)
- Extraction: 92% EXTRACTED · 8% INFERRED · 0% AMBIGUOUS · INFERRED: 139 edges (avg confidence: 0.85)
- Token cost: 0 input · 0 output

## Community Hubs (Navigation)
- Web Auth & Dashboard Pages
- AWS Secrets Source
- Receipts & Split Pages
- Python Dev Scripts
- Receipt Contracts
- Task Handoff Reports
- CI Workflow
- Banking Secrets ADR & Agent Rules
- Web Dependencies
- Backend Identity & Receipts Store
- Foundation & Frontend Contracts
- Go Stdlib Usage
- Root Package Config
- Dashboard Section Routing
- ADR & Teller Task Docs
- JWT & HTTP Test Doubles
- TypeScript Config
- Receipt Join Tests
- Token Keyring Encryption
- Local Secret Source
- Gemini Provider Errors
- Image Validation
- Receipt List Tests
- API Entrypoint
- Project Specs
- Community 25
- Community 26
- Community 27
- Community 28
- Community 29
- Community 30
- Community 31
- Community 32
- Community 33
- Community 34
- Community 35
- Community 36
- Community 37
- Community 38
- Community 39
- Community 40
- Community 41
- Community 42
- Community 43
- Community 44
- Community 45
- Community 46
- Community 47

## God Nodes (most connected - your core abstractions)
1. `react` - 21 edges
2. `ReceiptWorkspace()` - 18 edges
3. `API` - 18 edges
4. `compilerOptions` - 16 edges
5. `VP-008 Receipt image to split preview (COMPLETED)` - 16 edges
6. `Button()` - 15 edges
7. `Store` - 15 edges
8. `scripts` - 14 edges
9. `VP-018 Plaid Banking provider (replaces Teller)` - 14 edges
10. `Feedback()` - 13 edges

## Surprising Connections (you probably didn't know these)
- `Domain ownership rule` --semantically_similar_to--> `Seven domain boundary rules`  [INFERRED] [semantically similar]
  AGENTS.md → agents/decisions/ADR-010-modular-monolith-split-strategy.md
- `Architecture change escalation` --references--> `ADR-011 Replace Teller with Plaid`  [INFERRED]
  AGENTS.md → agents/decisions/ADR-011-replace-teller-with-plaid.md
- `Technology stack (Next.js, Go, Supabase)` --conceptually_related_to--> `ADR-011 Replace Teller with Plaid`  [AMBIGUOUS]
  DESIGN.md → agents/decisions/ADR-011-replace-teller-with-plaid.md
- `ADR-010 Modular monolith, service split later` --references--> `Modular monolith backend`  [INFERRED]
  agents/decisions/ADR-010-modular-monolith-split-strategy.md → DESIGN.md
- `Banking credentials and secrets handling` --references--> `ADR-009 Banking secrets and token encryption`  [INFERRED]
  PROJECT.md → agents/decisions/ADR-009-banking-secrets-and-token-encryption.md

## Import Cycles
- None detected.

## Hyperedges (group relationships)
- **Banking provider secrets and token handling** — agents_decisions_adr_009_banking_secrets_and_token_encryption_secret_source, agents_decisions_adr_009_banking_secrets_and_token_encryption_aes_gcm, design_institution_connections, project_banking_secrets [INFERRED 0.85]
- **Receipt split session lifecycle and access** — agents_contracts_receipt_sessions_session_endpoints, agents_contracts_receipt_sessions_public_join, agents_contracts_receipt_sessions_split_sessions_schema, agents_contracts_receipt_sessions_realtime_broadcast, design_split_sessions_table [INFERRED 0.95]
- **Agent governance documents** — agents, agents_authority_order, agents_task_lifecycle, agents_tasks_coordination_subagent_protocol, agents_git_workflow [INFERRED 0.85]
- **Banking tasks reworked by Plaid switch** — agents_tasks_backlog_vp_009c, agents_tasks_backlog_vp_010, agents_tasks_backlog_vp_011, agents_tasks_backlog_vp_018_plaid_banking [EXTRACTED 1.00]
- **Receipt stack VP-015a, VP-015b, VP-016** — agents_tasks_review_vp_015a, agents_tasks_review_vp_015b, agents_tasks_review_vp_016 [EXTRACTED 1.00]

## Communities (50 total, 19 thin omitted)

### Community 0 - "Web Auth & Dashboard Pages"
Cohesion: 0.06
Nodes (51): Callback(), DashboardLayout(), SettingsPage(), Home(), DashboardShell(), handleSignOut(), mocks, renderShell() (+43 more)

### Community 1 - "AWS Secrets Source"
Cohesion: 0.06
Nodes (32): AWSSource, cached, fakeSM, SecretsManagerAPI, API, Profile, API, Extractor (+24 more)

### Community 2 - "Receipts & Split Pages"
Cohesion: 0.07
Nodes (44): ReceiptsPage(), metadata, SplitPage(), Button(), ButtonProps, authorizedFetch(), postJSON(), receiptImage() (+36 more)

### Community 3 - "Python Dev Scripts"
Cohesion: 0.08
Nodes (11): import_private(), primary_root(), read_private(), main(), read_private(), request(), verify(), DevConfigurationTests (+3 more)

### Community 4 - "Receipt Contracts"
Cohesion: 0.04
Nodes (47): Receipt list contract (VP-016), amount_kind (DRAFT_PREVIEW/NONE), outstanding_minor (finalized receivables only), GET /api/v1/receipts/summary, Receipt split-session contract (VP-015a/b), Store.CheckSession participant gate, POST /api/v1/split/session public join endpoint, Payload-free realtime broadcast split:<session id> (+39 more)

### Community 5 - "Task Handoff Reports"
Cohesion: 0.07
Nodes (47): VP-007 Dashboard handoff report, VP-007 Foundation handoff report, VP-008 final review record, VP-008 receipt extraction options research, VP-004 Readable reusable auth UI (ACTIVE), VP-005 Authenticated Dashboard and MVP navigation (ACTIVE), VP-006 Safe local dev configuration and startup (ACTIVE), VP-006 M2 Pinned tool discovery (ACTIVE) (+39 more)

### Community 6 - "CI Workflow"
Cohesion: 0.05
Nodes (43): name, runs-on, steps, timeout-minutes, concurrency, cancel-in-progress, group, name (+35 more)

### Community 7 - "Banking Secrets ADR & Agent Rules"
Cohesion: 0.06
Nodes (39): Architecture change escalation, Banking agent, ADR-009 Banking secrets and token encryption, AES-256-GCM per-user token encryption, SecretSource interface (local files / AWS Secrets Manager), SyncConnection manual refresh with cooldown, ADR-010 Modular monolith, service split later, Seven domain boundary rules (+31 more)

### Community 8 - "Web Dependencies"
Cohesion: 0.07
Nodes (38): dependencies, next@16.4.1, qrcode-generator@2.0.4, react@19.3.0, react-dom@19.3.0, @supabase/supabase-js@^2.117.3, devDependencies, eslint@9.39.5 (+30 more)

### Community 9 - "Backend Identity & Receipts Store"
Cohesion: 0.14
Nodes (5): identityKey, shareRow, SummaryRow, main(), summarize()

### Community 10 - "Foundation & Frontend Contracts"
Cohesion: 0.09
Nodes (32): Foundation Auth Contract, GET /api/v1/me endpoint, Supabase PKCE bearer-only auth flow, Frontend Components Contract, Receipt Drafts Contract, Receipt draft HTTP API (/api/v1/receipts), Largest-remainder tax/tip distribution, Private receipt-images bucket (+24 more)

### Community 11 - "Go Stdlib Usage"
Cohesion: 0.12
Nodes (19): Draft, Item, Participant, Preview, Proposal, Share, testExtractor, bounded() (+11 more)

### Community 12 - "Root Package Config"
Cohesion: 0.08
Nodes (26): devDependencies, supabase@2.120.0, engines, node, name, undici@8.11.2, zod@4.6.5, packageManager (+18 more)

### Community 13 - "Dashboard Section Routing"
Cohesion: 0.19
Nodes (12): DashboardPage(), DashboardSectionPage(), metadata, DashboardNavigation(), NavigationGroup(), FeatureUnavailable(), DashboardDestination, destinationFor() (+4 more)

### Community 14 - "ADR & Teller Task Docs"
Cohesion: 0.17
Nodes (21): ADR-009 secrets and AES-256-GCM encryption, ADR-010 modular monolith import boundaries, ADR-011 Plaid replaces Teller, VP-009c Removable Teller connection probe, VP-010 Authenticated Teller connection lifecycle, VP-011 Normalized accounts and manual refresh, VP-014 Domain import-boundary CI check, VP-018 Plaid Banking provider (replaces Teller) (+13 more)

### Community 15 - "JWT & HTTP Test Doubles"
Cohesion: 0.26
Nodes (13): New(), fixture(), request(), TestBoundedRemoteValidation(), TestConcurrentUsers(), TestConfigurationRejectsUnsafeTargets(), TestMismatchAnonymousAndOversizedProvider(), TestOriginAndContext() (+5 more)

### Community 16 - "TypeScript Config"
Cohesion: 0.11
Nodes (18): compilerOptions, allowJs, esModuleInterop, incremental, isolatedModules, jsx, lib, module (+10 more)

### Community 17 - "Receipt Join Tests"
Cohesion: 0.23
Nodes (8): join(), TestJoinExpiryParticipantsAndNoOwnerLeak(), TestJoinRateLimitedAndCORS(), call(), sessionAPI(), TestSessionCreationRateLimitedAndTTLConfig(), TestSessionExpiryTokenReplacementAndNoRawToken(), TestSessionOwnerOnlyAndClose()

### Community 18 - "Token Keyring Encryption"
Cohesion: 0.20
Nodes (9): Keyring, aad(), NewKeyring(), LoadKeyring(), key(), ring(), TestKeyringFailsClosed(), TestNonceUnique() (+1 more)

### Community 19 - "Local Secret Source"
Cohesion: 0.20
Nodes (7): LocalSource, SecretSource, DefaultLocalDir(), NewAWSSource(), NewSecretSource(), TestLocalPermissions(), TestNewSecretSourceFailsClosed()

### Community 20 - "Gemini Provider Errors"
Cohesion: 0.20
Nodes (11): ProviderError, geminiFailure(), NewGemini(), proposalSchema(), TestGeminiBadRequestFieldPathsAreWhitelisted(), TestGeminiFailureBounds(), TestGeminiModelSchemaUsesVerifiedWireContract(), TestGeminiNoKeyAndCancelled() (+3 more)

### Community 21 - "Image Validation"
Cohesion: 0.22
Nodes (7): NewAPI(), pngFixture(), TestImagesContentAndLimits(), TestJSONBoundsAndUnknownFields(), TestReceiptHandlerCannotBypassFoundation(), TestReceiptRateLimiterBoundAndExpiry(), ValidateImage()

### Community 22 - "Receipt List Tests"
Cohesion: 0.32
Nodes (4): fakeReceipt, listDB, summaryCall(), TestSummaryIsolationPaginationAndExactSums()

### Community 23 - "API Entrypoint"
Cohesion: 0.33
Nodes (5): main(), main(), ParseSessionTTL(), NewStore(), TestWorkerOneDeliveryFinishesOrSchedulesBoundedRetry()

### Community 24 - "Project Specs"
Cohesion: 0.50
Nodes (4): AGENTS.md Agent Rules, Credit optimization and local validation, DESIGN.md Technical Design, PROJECT.md Vision and MVP 1 Spec

### Community 25 - "Community 25"
Cohesion: 0.50
Nodes (4): GOOGLE_CLIENT_ID, GOOGLE_CLIENT_SECRET, run_cli(), local-db.sh script

### Community 26 - "Community 26"
Cohesion: 0.60
Nodes (3): apply_files(), new_database(), test-receipt-migration.sh script

### Community 27 - "Community 27"
Cohesion: 0.67
Nodes (4): VP-006 M1 Evidence, VP-006 M1 Review (APPROVED), VP-006 Local dev startup, VP-006 M1 private configuration contract

### Community 28 - "Community 28"
Cohesion: 0.83
Nodes (3): cleanup(), test-profile-concurrency.sh script, sql()

### Community 29 - "Community 29"
Cohesion: 0.67
Nodes (3): Traffic protection rules, Caching strategy (user-scoped), Traffic protection architecture

### Community 30 - "Community 30"
Cohesion: 0.67
Nodes (3): Authentication flow (Google, Supabase Auth, JWT, ctx.user_id), profiles table, Single Vantage-Point user identity

## Ambiguous Edges - Review These
- `Technology stack (Next.js, Go, Supabase)` → `ADR-011 Replace Teller with Plaid`  [AMBIGUOUS]
  DESIGN.md · relation: conceptually_related_to
- `ADR-011 Replace Teller with Plaid` → `Banking/Teller plan 2026-10-28`  [AMBIGUOUS]
  agents/tasks/COORDINATION.md · relation: conceptually_related_to

## Knowledge Gaps
- **77 isolated node(s):** `GOOGLE_CLIENT_ID`, `GOOGLE_CLIENT_SECRET`, `github.com/ryanzhou07/Vantage-Point/services/backend`, `Orchestrator agent`, `Foundation agent` (+72 more)
  These have ≤1 connection - possible missing edges or undocumented components. (Counts symbols only; 246 node(s) total have ≤1 connection when file, concept and rationale nodes are included.)
- **19 thin communities (<3 nodes) omitted from report** — run `graphify query` to explore isolated nodes.

## Suggested Questions
_Questions this graph is uniquely positioned to answer:_

- **Why does `react` connect `Web Auth & Dashboard Pages` to `Web Dependencies`, `Receipts & Split Pages`, `Dashboard Section Routing`?**
  _High betweenness centrality (0.020) - this node is a cross-community bridge._
- **What connects `GOOGLE_CLIENT_ID`, `GOOGLE_CLIENT_SECRET`, `github.com/ryanzhou07/Vantage-Point/services/backend` to the rest of the system?**
  _77 weakly-connected nodes found - possible documentation gaps or missing edges._
- **Should `Web Auth & Dashboard Pages` be split into smaller, more focused modules?**
  _Cohesion score 0.0559244126659857 - nodes in this community are weakly interconnected._
- **Why does `vitest` connect `Web Auth & Dashboard Pages` to `Web Dependencies`, `Receipts & Split Pages`, `Dashboard Section Routing`?**
  _High betweenness centrality (0.009) - this node is a cross-community bridge._
- **Should `AWS Secrets Source` be split into smaller, more focused modules?**
  _Cohesion score 0.06265822784810127 - nodes in this community are weakly interconnected._

### Low-confidence Hints
_AMBIGUOUS edges — the extractor was unsure. Verify before acting on these._

- **What is the exact relationship between `Technology stack (Next.js, Go, Supabase)` and `ADR-011 Replace Teller with Plaid`?**
  _Edge tagged AMBIGUOUS (relation: conceptually_related_to) - confidence is low._
- **What is the exact relationship between `ADR-011 Replace Teller with Plaid` and `Banking/Teller plan 2026-10-28`?**
  _Edge tagged AMBIGUOUS (relation: conceptually_related_to) - confidence is low._