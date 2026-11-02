#!/usr/bin/env bash
# One-time local Banking setup: asks for the Plaid Sandbox keys (hidden input) and stores them privately.
set -euo pipefail
cd "$(dirname "$0")/.."
dir=${BANKING_SECRET_DIR:-$HOME/.config/vantage-point/banking}
umask 077
mkdir -p "$dir" && chmod 700 "$dir"
read -rsp 'Plaid client_id: ' id; echo
read -rsp 'Plaid SANDBOX secret: ' sec; echo
[[ -n $id && -n $sec ]] || { echo 'Both values are required.' >&2; exit 1; }
printf '%s' "$id" >"$dir/plaid-client-id"
printf '%s' "$sec" >"$dir/plaid-secret"
chmod 600 "$dir/plaid-client-id" "$dir/plaid-secret"
# Never regenerate: a new key would make already-stored bank tokens unreadable.
if [[ ! -f .env.banking.local ]]; then
  printf 'BANKING_SECRET_SOURCE=local\nPLAID_ENV=sandbox\nTOKEN_ENCRYPTION_KEY=%s\n' "$(openssl rand -base64 32)" >.env.banking.local
  chmod 600 .env.banking.local
fi
echo "Saved to $dir and .env.banking.local (both private, git-ignored). Tell Claude: keys are in."
