# ADR-011 — Replace Teller with Plaid as the Banking provider

Status: APPROVED 2026-10-28 by project owner (AGENTS.md §24 "Replacing Teller"). Owner reports Teller has been shut down; this is not independently verified by agents.

Decision: Plaid replaces Teller behind the Banking provider adapter. Domain code keeps consuming normalized Vantage-Point types. Start on a Plaid Trial plan (US/Canada, real production data, free, 10 Production Items cap; removing Items does not restore the cap) and use Sandbox for tests/CI.

Consequences:
- Plaid Link flow differs from Teller Connect: backend creates a `link_token`; the browser returns a short-lived `public_token`; the backend exchanges it for an `access_token` and `item_id`. The access token never reaches the browser. VP-010 enroll and VP-017 UI must be reworked.
- Auth is `client_id` + `secret`; no client certificate. VP-009b (mutual TLS) and VP-009c (probe) are obsolete for Plaid. VP-009a (SecretSource, AES-256-GCM token encryption) is reused for Plaid secrets and access tokens.
- Transactions use cursor-based `/transactions/sync` (feeds VP-011 and pending to posted reconciliation); webhooks need signature verification and idempotency.
- Trial cap of 10 Items bounds testing; do not create Items needlessly.

Not verified: Plaid pricing after the Trial, which products the Trial plan permits, the Link use case selection, and Teller's shutdown notice.
