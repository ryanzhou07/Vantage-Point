#!/usr/bin/env bash
set -euo pipefail
cd "$(dirname "$0")/.."
container=${RECEIPT_TEST_DB_CONTAINER:-supabase_db_vantage-point-local}
case "$container" in
  supabase_db_vantage-point-receipt-tests) ;;
  supabase_db_vantage-point-local) [[ ${CI:-} == true && ${GITHUB_ACTIONS:-} == true ]] || { echo 'Use only the dedicated Receipt test container outside GitHub CI.' >&2; exit 2; } ;;
  *) echo 'Unknown test container.' >&2; exit 2 ;;
esac
databases=()
cleanup() { for database_name in "${databases[@]}"; do docker exec "$container" dropdb -U postgres --if-exists "$database_name" >/dev/null; done; }
trap cleanup EXIT
new_database() {
 database="vp008_migration_${RANDOM}_$$_$1"
 databases+=("$database")
 docker exec "$container" createdb -U postgres --template=template0 "$database"
}
apply_files() {
 for file in "$@"; do docker exec -i "$container" psql -U postgres -d "$database" -v ON_ERROR_STOP=1 --single-transaction < "$file" >/dev/null; done
}
new_database legacy
apply_files supabase/tests/fixtures/receipt-legacy-bootstrap.sql supabase/migrations/20261011000100_foundation_profiles.sql supabase/migrations/20261015000100_receipt_drafts.sql supabase/tests/fixtures/receipt-legacy-seed.sql supabase/migrations/20261015000150_receipt_legacy_review_compat.sql supabase/migrations/20261015000200_receipt_normalized_drafts.sql supabase/migrations/20261015000300_receipt_item_allocation_guard.sql supabase/tests/fixtures/receipt-legacy-assert.sql
new_database normalized
apply_files supabase/tests/fixtures/receipt-legacy-bootstrap.sql supabase/migrations/20261011000100_foundation_profiles.sql supabase/migrations/20261015000100_receipt_drafts.sql supabase/migrations/20261015000200_receipt_normalized_drafts.sql supabase/migrations/20261015000300_receipt_item_allocation_guard.sql supabase/tests/fixtures/receipt-normalized-seed.sql supabase/migrations/20261015000150_receipt_legacy_review_compat.sql supabase/tests/fixtures/receipt-normalized-noop-assert.sql
echo 'Receipt forward migrations passed: raw dates/incomplete drafts audited, unknown fields require review, exact corrections retained, normalized databases unchanged.'
