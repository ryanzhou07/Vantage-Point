# Vantage-Point

Vantage-Point is a mobile-first personal-finance application focused on connected financial visibility and collaborative receipt splitting.

## Documentation

The project intentionally keeps permanent documentation small.

```text
PROJECT.md
```

Defines **what Vantage-Point must do**.

```text
DESIGN.md
```

Defines **how Vantage-Point is designed and built**.

```text
AGENTS.md
```

Defines **how coding agents operate inside the repository**.

```text
agents/tasks/
```

Contains executable `VP-###` work items.

## MVP

MVP 1 focuses on:

- Google authentication
- Connected financial accounts
- Balances and transactions
- Recurring financial activity
- Receipt OCR/review
- Realtime collaborative receipt splitting
- Reimbursement tracking
- Mobile-friendly Dashboard

See `PROJECT.md` for authoritative product requirements.

## Technology Direction

```text
Frontend     Next.js / TypeScript
Backend      Go
Database     PostgreSQL / Supabase
Auth         Supabase Auth / Google OAuth
Banking      Teller + Plaid fallback
Realtime     Supabase Realtime direction
Deployment   Vercel + Google Cloud Run direction
```

See `DESIGN.md` for the authoritative technical design.

## Development

Use Node.js `22.21.0`, pnpm `10.34.6`, and Go `1.26.9`. Enable pnpm through Corepack (or install the pinned pnpm release), then:

```sh
pnpm install --frozen-lockfile
pnpm dev
```

The web app runs at `http://localhost:3000`. To start the API in another terminal:

```sh
cd services/backend
go run ./cmd/api
```

The API listens on `http://localhost:8080` by default and exposes `GET /health`. Copy `.env.example` to `apps/web/.env.local` for web configuration. Set `API_PORT` in the API process environment if a different port is needed.

### Local Supabase and migrations

Install Docker Desktop (or an equivalent local Docker socket) and start its daemon.
The pinned Supabase CLI `2.120.0` is installed by pnpm; its packaged native executable
requires no separate Bun installation. PostgreSQL major version is 17.
Run these commands from the repository root:

```sh
pnpm db:start
pnpm db:validate
pnpm db:stop
```

`db:validate` destroys only the disposable `vantage-point-local` Docker database,
reapplies committed migrations from zero, and runs SQL assertions with errors fatal.
It accepts no URL or extra arguments and rejects remote Docker endpoints. Stop backs
up local data; reset intentionally discards it. Run only one worktree stack at a time.
Initial image downloads may take several minutes. CLI output is saved privately under
ignored `.local-logs/` because it can contain generated local keys; do not publish it.

Ports: web 3000, API 8080, Supabase gateway/Auth/REST 56321, PostgreSQL 56322,
Studio 56323, local mail viewer 56324, shadow database 56320. No production providers
are required. The scaffold API/web do not yet consume Supabase configuration.
Copy only frontend variables to `apps/web/.env.local`; the Go process does not read
`.env` automatically, so export backend variables in its terminal. Obtain local
keys privately using `pnpm exec supabase status`; never put service keys/database
credentials in `NEXT_PUBLIC_*` variables. OAuth application integration is VP-003.

See `supabase/migrations/README.md` for append-only UTC migration names and domain
ownership. Add domain SQL assertions to `supabase/tests/` with each schema task;
there are no application migrations, seeds, or fake provider implementations yet.

### Checks and CI

Local review commands:

```sh
pnpm lint
pnpm typecheck
pnpm test
pnpm build
pnpm backend:check
pnpm db:validate
pnpm security:secrets
pnpm security:dependencies
```

`pnpm test` uses Vitest 5.0.3; the scaffold has no tests and reports that absence.
`--passWithNoTests` permits only missing tests; an existing failing test still fails.
Go likewise currently reports no test files. These results provide no behavioral
coverage. Feature tasks must add meaningful tests per `agents/TESTING.md`.

Install Gitleaks 8.30.1 from its official release (verify release checksums), and
`go install golang.org/x/vuln/cmd/govulncheck@v1.1.4` for local security checks.
Gitleaks scans tracked current file contents with findings fully redacted; it does
not scan history or ignored local secrets. The full pnpm audit fails on high/critical manifest
advisories except the exact, owner-approved temporary lint-only braces finding
documented in `agents/decisions/ADR-002-runtime-security.md` (expires 2026-11-11); govulncheck scans reachable Go vulnerabilities including the runtime.
These network-backed databases can change; unavailable scans fail, and scans do
not guarantee absence of vulnerabilities. No allowlists or production keys are used.

GitHub Actions checks are **Frontend**, **Backend**, **Database**, and **Security**,
on PRs and `main`/`vp-*` pushes. Jobs have timeouts, read-only permissions and run
cancellation. CI applies migrations to a fresh local Docker stack. Recommend requiring
all four checks and an independent review before merging to protected `main`; this
task does not change repository branch-protection settings or deploy anything.

Do not use production financial credentials or production financial data for ordinary local development.

## Contributing / Agents

All implementation work must follow `AGENTS.md` and an approved `VP-###` task.
