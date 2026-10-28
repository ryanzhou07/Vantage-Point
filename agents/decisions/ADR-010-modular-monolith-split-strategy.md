# ADR-010 — Modular monolith now, service split later

Status: PROPOSED 2026-10-28; owner approval required.

Decision: Vantage-Point stays one Go API and one database for the MVP, with domains built so they can be split into services later without a rewrite. No new microservice is created now (AGENTS.md §24).

Rules: (1) no cross-domain table access; `profiles(id)` is the only shared reference; (2) domains call each other only through small Go interfaces, never internals; (3) identity is the verified token, not shared state; (4) no shared in-memory domain state; (5) async work crosses a job/event boundary; (6) contracts versioned in `agents/contracts/`; (7) import boundaries enforced by CI (VP-014).

Split path: (a) separate binaries from the same repo and database, Banking first to isolate the Teller certificate and encrypted tokens; (b) separate database per domain once rule 1 has held; (c) fully independent services only for demonstrated scale, team or isolation needs.

Consequences: money/allocation correctness stays in single-database transactions for MVP. Splitting later requires deliberately designing eventual consistency for any cross-domain flow.
