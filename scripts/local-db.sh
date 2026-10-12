#!/usr/bin/env bash
set -euo pipefail
cd "$(dirname "$0")/.."
# Ignored owner-created local OAuth configuration. Never print these values.
if [[ -f .env.oauth.local ]]; then
  set -a
  source .env.oauth.local
  set +a
fi
# Synthetic values let offline CI start Auth; they cannot pass real Google login.
export GOOGLE_CLIENT_ID=${GOOGLE_CLIENT_ID:-offline-fixture.apps.googleusercontent.com}
export GOOGLE_CLIENT_SECRET=${GOOGLE_CLIENT_SECRET:-offline-fixture-only}
if [[ $# != 1 ]]; then echo 'Usage: local-db.sh start|stop|validate (no target arguments)' >&2; exit 2; fi
case "$1" in start|stop|validate) ;; *) echo 'Unsupported local operation' >&2; exit 2;; esac
# Reject remote Docker daemons, even though --local targets Docker rather than Supabase cloud.
endpoint=${DOCKER_HOST:-$(docker context inspect --format '{{.Endpoints.docker.Host}}')}
case "$endpoint" in unix://*|npipe://*) ;; *) echo 'Only a local Docker socket is allowed' >&2; exit 2;; esac
# Do not inherit database or hosted Supabase target configuration.
unset DATABASE_URL PGHOST PGPORT PGDATABASE PGUSER PGPASSWORD SUPABASE_DB_URL SUPABASE_PROJECT_ID SUPABASE_ACCESS_TOKEN
if [[ $(pnpm exec supabase --version) != '2.120.0' ]]; then echo 'Install pinned Supabase CLI with pnpm install --frozen-lockfile' >&2; exit 1; fi
umask 077
mkdir -p .local-logs
run_cli() {
  if pnpm exec supabase --workdir "$PWD" --yes "$@" >.local-logs/supabase.log 2>&1; then
    echo "Local Supabase command succeeded: $*"
  else
    echo 'Local Supabase command failed. Inspect ignored .local-logs/supabase.log privately (may contain local keys).' >&2
    return 1
  fi
}
case "$1" in
  start) run_cli start ;;
  stop) run_cli stop ;;
  validate)
    run_cli db reset --local --no-seed
    # No URL/password input: connect inside the named local container only.
    for assertion in supabase/tests/*.sql; do
      docker exec -i supabase_db_vantage-point-local psql -U postgres -d postgres -v ON_ERROR_STOP=1 < "$assertion"
    done
    if compgen -G 'supabase/migrations/*.sql' >/dev/null; then
      docker exec supabase_db_vantage-point-local psql -U postgres -d postgres -v ON_ERROR_STOP=1 -c "DO \$\$ BEGIN IF to_regclass('supabase_migrations.schema_migrations') IS NULL THEN RAISE EXCEPTION 'Migration registry absent'; END IF; END \$\$;"
    else
      echo 'No application migrations yet; CLI does not create an empty migration registry.'
    fi
    if [[ -f scripts/test-profile-concurrency.sh ]]; then bash scripts/test-profile-concurrency.sh; fi
    echo 'Repository schema assertions passed.'
    ;;
esac
