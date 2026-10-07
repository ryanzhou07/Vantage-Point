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

Local setup commands will be added as the initial repository/bootstrap tasks are implemented.

Do not use production financial credentials or production financial data for ordinary local development.

## Contributing / Agents

All implementation work must follow `AGENTS.md` and an approved `VP-###` task.