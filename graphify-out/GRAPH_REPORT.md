# Graph Report - vp-006-f89  (2026-10-28)

## Corpus Check
- 139 files · ~55,672 words
- Verdict: corpus is large enough that graph structure adds value.
- Unclassified: 26 file(s) not represented in the graph (top: (none) 18, .css 4, .toml 2)

## Summary
- 666 nodes · 1385 edges · 31 communities (23 shown, 8 thin omitted)
- Extraction: 94% EXTRACTED · 6% INFERRED · 0% AMBIGUOUS · INFERRED: 83 edges (avg confidence: 0.84)
- Token cost: 0 input · 0 output

## Community Hubs (Navigation)
- Go Backend Core
- Next.js Pages & Routes
- Dev Tooling Scripts
- Task & Report Docs
- Auth Middleware (Go)
- CI Workflow
- Receipts UI & Shared UI
- Web Dependencies
- Receipt Domain Models
- Dashboard Shell
- Root Workspace Config
- TypeScript Config
- Auth & UI Contracts
- Receipt Money Rules
- Community 14
- Community 15
- Community 16
- Community 17
- Community 18
- Community 19
- Community 20
- Community 21
- Community 22
- Community 23
- Community 24
- Community 25
- Community 26
- Community 27
- Community 28
- Community 29
- Community 30

## God Nodes (most connected - your core abstractions)
1. `ReceiptWorkspace()` - 17 edges
2. `react` - 17 edges
3. `VP-008 Receipt image to split preview (COMPLETED)` - 17 edges
4. `compilerOptions` - 16 edges
5. `AGENTS.md Agent Rules` - 16 edges
6. `DESIGN.md Technical Design` - 16 edges
7. `scripts` - 14 edges
8. `Calculate()` - 13 edges
9. `PROJECT.md MVP 1 Specification` - 13 edges
10. `Implementation coordination` - 13 edges

## Surprising Connections (you probably didn't know these)
- `Integer minor-unit money` --semantically_similar_to--> `Largest-remainder tax/tip distribution`  [INFERRED] [semantically similar]
  DESIGN.md → agents/contracts/RECEIPT_DRAFTS.md
- `VP-006 M1 Review (APPROVED)` --implements--> `Integration / Review agent`  [INFERRED]
  agents/reports/VP-006-M1-review.md → AGENTS.md
- `Extractor interface (provider-neutral)` --implements--> `Provider adapter boundary rule`  [INFERRED]
  agents/contracts/RECEIPT_DRAFTS.md → AGENTS.md
- `Migration ownership README` --references--> `VP-002 Local Supabase, migrations, CI (COMPLETED)`  [INFERRED]
  supabase/migrations/README.md → agents/tasks/completed/VP-002.md
- `Foundation Auth Contract` --references--> `DESIGN.md Technical Design`  [EXTRACTED]
  agents/contracts/FOUNDATION_AUTH.md → DESIGN.md

## Import Cycles
- None detected.

## Hyperedges (group relationships)
- **Governance doc hierarchy (PROJECT, DESIGN, AGENTS, authority order)** — project_md, design_md, agents_md, agents_md_authority_order [EXTRACTED 1.00]
- **Teller Banking task chain VP-009 to VP-011** — agents_tasks_backlog_vp_009, agents_tasks_backlog_vp_010, agents_tasks_backlog_vp_011 [EXTRACTED 1.00]
- **VP-006 sequential milestones M1-M4** — concept_vp_006_m1, concept_vp_006_m2, concept_vp_006_m3, concept_vp_006_m4 [EXTRACTED 1.00]
- **VP-007 Settings and callback package** — agents_tasks_ready_vp_007, agents_reports_vp_007_dashboard, agents_reports_vp_007_foundation [EXTRACTED 1.00]
- **Foundation auth flow (PKCE, useAuthSession, Protect, /me)** — agents_contracts_foundation_auth_pkce, agents_contracts_frontend_components_use_auth_session, agents_contracts_foundation_auth_protect, agents_contracts_foundation_auth_me_endpoint, provider_supabase [INFERRED 0.95]
- **Receipt vision pipeline (API, durable job queue, worker, Extractor, Gemini)** — agents_contracts_receipt_drafts_api, agents_contracts_receipt_drafts_vision_jobs, agents_guides_receipt_setup_receipt_worker, agents_contracts_receipt_drafts_extractor, provider_gemini [INFERRED 0.95]

## Communities (31 total, 8 thin omitted)

### Community 0 - "Go Backend Core"
Cohesion: 0.06
Nodes (39): identityKey, Profile, ProviderError, rateEntry, testExtractor, main(), main(), main() (+31 more)

### Community 1 - "Next.js Pages & Routes"
Cohesion: 0.06
Nodes (51): Callback(), DashboardLayout(), SettingsPage(), Home(), DashboardShell(), handleSignOut(), mocks, renderShell() (+43 more)

### Community 2 - "Dev Tooling Scripts"
Cohesion: 0.08
Nodes (14): import_private(), primary_root(), read_private(), candidates(), pins(), validate(), main(), read_private() (+6 more)

### Community 3 - "Task & Report Docs"
Cohesion: 0.07
Nodes (55): VP-007 Dashboard handoff report, VP-007 Foundation handoff report, VP-008 final review record, VP-008 receipt extraction options research, VP-004 Readable reusable auth UI (ACTIVE), VP-005 Authenticated Dashboard and MVP navigation (ACTIVE), VP-006 Safe local dev configuration and startup (ACTIVE), VP-006 M2 Pinned tool discovery (ACTIVE) (+47 more)

### Community 4 - "Auth Middleware (Go)"
Cohesion: 0.10
Nodes (18): API, API, Extractor, FakeVision, Gemini, Job, Record, Store (+10 more)

### Community 5 - "CI Workflow"
Cohesion: 0.05
Nodes (43): name, runs-on, steps, timeout-minutes, concurrency, cancel-in-progress, group, name (+35 more)

### Community 6 - "Receipts UI & Shared UI"
Cohesion: 0.13
Nodes (26): ReceiptsPage(), Button(), ButtonProps, authorizedFetch(), postJSON(), receiptImage(), receiptRequest(), Cents() (+18 more)

### Community 7 - "Web Dependencies"
Cohesion: 0.07
Nodes (35): dependencies, next@16.4.0, react@19.3.0, react-dom@19.3.0, @supabase/supabase-js@^2.117.3, devDependencies, eslint@9.39.5, eslint-config-next@16.4.0 (+27 more)

### Community 8 - "Receipt Domain Models"
Cohesion: 0.14
Nodes (18): Draft, Item, Participant, Preview, Proposal, Share, bounded(), Calculate() (+10 more)

### Community 9 - "Dashboard Shell"
Cohesion: 0.19
Nodes (12): DashboardPage(), DashboardSectionPage(), metadata, DashboardNavigation(), NavigationGroup(), FeatureUnavailable(), DashboardDestination, destinationFor() (+4 more)

### Community 10 - "Root Workspace Config"
Cohesion: 0.09
Nodes (22): devDependencies, supabase@2.120.0, engines, node, name, packageManager, private, scripts (+14 more)

### Community 11 - "TypeScript Config"
Cohesion: 0.11
Nodes (18): compilerOptions, allowJs, esModuleInterop, incremental, isolatedModules, jsx, lib, module (+10 more)

### Community 12 - "Auth & UI Contracts"
Cohesion: 0.19
Nodes (14): Foundation Auth Contract, GET /api/v1/me endpoint, Supabase PKCE bearer-only auth flow, Frontend Components Contract, UI primitives (Button, Surface, Feedback, Brand), useAuthSession hook, VP-003 Evidence, VP-004 Evidence (+6 more)

### Community 13 - "Receipt Money Rules"
Cohesion: 0.19
Nodes (13): Largest-remainder tax/tip distribution, Allocation concurrency invariant, Atomic receipt finalization, Receipt data model (receipts, items, participants, allocations, receivables), Receipt revision/reset strategy, PROJECT.md MVP 1 Specification, Receipt audit history, MVP 1 boundaries (non-goals) (+5 more)

### Community 14 - "Community 14"
Cohesion: 0.23
Nodes (11): Temporary braces advisory exception (expires 2026-11-11), Go 1.26.9 runtime pin, ADR-002 Runtime Security Remediation, VP-002 Evidence, Integration / Review agent, Test Guidelines VP-002/VP-003, CI checks (GitHub Actions), README (dev setup, checks, CI) (+3 more)

### Community 15 - "Community 15"
Cohesion: 0.35
Nodes (11): Receipt Drafts Contract, Extractor interface (provider-neutral), ADR-008 Receipt Vision Provider Boundary, VP-008 Evidence, Gemini Receipt Setup Guide, Live synthetic Gemini smoke (receipt-command.py), VP-008 Implementation Plan, Receipts domain (+3 more)

### Community 16 - "Community 16"
Cohesion: 0.22
Nodes (10): receipt_vision_jobs durable leased queue, Receipt vision worker process, DESIGN.md Technical Design, API error shape and HTTP conventions, Banking data model (connections, accounts, transactions, provider_events), User-scoped caching policy, Future ledger engine (deferred), Modular monolith Go backend (+2 more)

### Community 17 - "Community 17"
Cohesion: 0.29
Nodes (7): AGENTS.md Agent Rules, Orchestrator agent, Task lifecycle (BACKLOG..COMPLETED, BLOCKED), Banking domain, Tech stack (Next.js, Go, Supabase, Teller/Plaid), Plaid (secondary banking provider), Teller (primary banking provider)

### Community 19 - "Community 19"
Cohesion: 0.40
Nodes (5): VP-006 M1 Evidence, scripts/dev_config.py parser, VP-006 M1 Review (APPROVED), VP-006 Local dev startup, VP-006 M1 private configuration contract

### Community 20 - "Community 20"
Cohesion: 0.40
Nodes (5): foundation.API.Protect middleware, Receipt draft HTTP API (/api/v1/receipts), Private receipt-images bucket, Supabase Realtime (non-authoritative), Supabase (Auth, Postgres, Storage, Realtime)

### Community 21 - "Community 21"
Cohesion: 0.50
Nodes (4): GOOGLE_CLIENT_ID, GOOGLE_CLIENT_SECRET, run_cli(), local-db.sh script

### Community 22 - "Community 22"
Cohesion: 0.60
Nodes (3): apply_files(), new_database(), test-receipt-migration.sh script

### Community 23 - "Community 23"
Cohesion: 0.83
Nodes (3): cleanup(), test-profile-concurrency.sh script, sql()

## Knowledge Gaps
- **27 isolated node(s):** `GOOGLE_CLIENT_ID`, `GOOGLE_CLIENT_SECRET`, `github.com/ryanzhou07/Vantage-Point/services/backend`, `VP-003 Evidence`, `VP-004 Evidence` (+22 more)
  These have ≤1 connection - possible missing edges or undocumented components. (Counts symbols only; 174 node(s) total have ≤1 connection when file, concept and rationale nodes are included.)
- **8 thin communities (<3 nodes) omitted from report** — run `graphify query` to explore isolated nodes.

## Suggested Questions
_Questions this graph is uniquely positioned to answer:_

- **Why does `react` connect `Next.js Pages & Routes` to `Dashboard Shell`, `Receipts UI & Shared UI`, `Web Dependencies`?**
  _High betweenness centrality (0.023) - this node is a cross-community bridge._
- **What connects `GOOGLE_CLIENT_ID`, `GOOGLE_CLIENT_SECRET`, `github.com/ryanzhou07/Vantage-Point/services/backend` to the rest of the system?**
  _27 weakly-connected nodes found - possible documentation gaps or missing edges._
- **Should `Go Backend Core` be split into smaller, more focused modules?**
  _Cohesion score 0.05967540574282147 - nodes in this community are weakly interconnected._
- **Why does `vitest` connect `Next.js Pages & Routes` to `Dashboard Shell`, `Receipts UI & Shared UI`, `Web Dependencies`?**
  _High betweenness centrality (0.011) - this node is a cross-community bridge._
- **Should `Next.js Pages & Routes` be split into smaller, more focused modules?**
  _Cohesion score 0.0560875512995896 - nodes in this community are weakly interconnected._
- **Why does `next` connect `Dashboard Shell` to `Next.js Pages & Routes`, `Web Dependencies`?**
  _High betweenness centrality (0.009) - this node is a cross-community bridge._
- **Should `Dev Tooling Scripts` be split into smaller, more focused modules?**
  _Cohesion score 0.07656341320864991 - nodes in this community are weakly interconnected._