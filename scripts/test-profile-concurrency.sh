#!/usr/bin/env bash
set -euo pipefail
# Only synthetic identity on the established local stack. Never a remote URL.
endpoint=${DOCKER_HOST:-$(docker context inspect --format '{{.Endpoints.docker.Host}}')}
case "$endpoint" in unix://*|npipe://*) ;; *) echo 'Only a local Docker socket is allowed' >&2; exit 2;; esac
container=supabase_db_vantage-point-local
identity=00000000-0000-4000-8000-000000000010
sql() { docker exec "$container" psql -U postgres -d postgres -v ON_ERROR_STOP=1 -q -c "$1"; }
cleanup() { sql "delete from auth.users where id='$identity';" >/dev/null; }
trap cleanup EXIT
sql "insert into auth.users(id,email) values('$identity','concurrent@example.invalid');" >/dev/null
pids=()
for i in {1..8}; do
  sql "begin; update auth.users set raw_user_meta_data=coalesce(raw_user_meta_data,'{}'::jsonb) where id='$identity'; select pg_sleep(0.025); commit;" >/dev/null &
  pids+=("$!")
done
for pid in "${pids[@]}"; do wait "$pid"; done
sql "DO \$\$ BEGIN IF (select count(*) from public.profiles where id='$identity') <> 1 THEN RAISE EXCEPTION 'Concurrent provisioning failed'; END IF; END \$\$;"
echo 'Concurrent canonical profile provisioning passed (8 independent transactions).'
