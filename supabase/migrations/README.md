# Migration ownership

No application migrations exist at VP-002. Supabase supplies its platform schema.
Create append-only files named `YYYYMMDDHHMMSS_domain_description.sql` (UTC timestamps,
unique across branches). Foundation owns profiles/shared identity; Banking owns banking;
Receipt owns receipts. Applied migrations must never be rewritten. Shared identities,
cross-domain foreign keys, grants, and database-wide changes require Integration Review.

Run `pnpm db:start` then `pnpm db:validate` to rebuild the disposable local database
from all migrations, in timestamp order, and run `supabase/tests/*.sql` assertions.
Add assertions for each new domain object and real-role authorization/RLS tests.
Use exact integer minor units, foreign keys, and uniqueness/idempotency constraints.
Do not create placeholder tables or add production data. No seed runs by default.
