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

Use Node.js `22.21.0`, pnpm `10.34.6`, and Go `1.24.2`. Enable pnpm through Corepack (or install the pinned pnpm release), then:

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

Local review commands:

```sh
pnpm lint
pnpm typecheck
pnpm build
cd services/backend && go vet ./... && go test ./... && go build ./...
```

Do not use production financial credentials or production financial data for ordinary local development.

## Contributing / Agents

All implementation work must follow `AGENTS.md` and an approved `VP-###` task.
