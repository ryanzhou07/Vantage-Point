#!/usr/bin/env bash
set -euo pipefail
cd "$(dirname "$0")/.."
container=${RECEIPT_TEST_DB_CONTAINER:-supabase_db_vantage-point-local}
case "$container" in
  supabase_db_vantage-point-receipt-tests) ;;
  supabase_db_vantage-point-local) [[ ${CI:-} == true && ${GITHUB_ACTIONS:-} == true ]] || { echo 'Use only the dedicated Receipt test container outside GitHub CI.' >&2; exit 2; } ;;
  *) echo 'Unknown test container.' >&2; exit 2 ;;
esac
database="vp008_migration_${RANDOM}_$$"
cleanup() { docker exec "$container" dropdb -U postgres --if-exists "$database" >/dev/null; }
trap cleanup EXIT
docker exec "$container" createdb -U postgres --template=template0 "$database"
for file in supabase/tests/fixtures/receipt-legacy-bootstrap.sql supabase/migrations/20261011000100_foundation_profiles.sql supabase/migrations/20261015000100_receipt_drafts.sql supabase/tests/fixtures/receipt-legacy-seed.sql supabase/migrations/20261015000200_receipt_normalized_drafts.sql supabase/migrations/20261015000300_receipt_item_allocation_guard.sql supabase/tests/fixtures/receipt-legacy-assert.sql; do
  docker exec -i "$container" psql -U postgres -d "$database" -v ON_ERROR_STOP=1 --single-transaction < "$file" >/dev/null
done
echo 'Receipt forward migration passed: legacy owner corrections, UUID mappings, exact cents, normalized rows and audit retained; JSON authority removed.'
