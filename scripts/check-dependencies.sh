#!/usr/bin/env bash
set -uo pipefail
cd "$(dirname "$0")/.."
status=0
node scripts/check-pnpm-audit.mjs || status=1
cd services/backend
govulncheck ./... || status=1
exit "$status"
