#!/usr/bin/env bash
set -euo pipefail
cd "$(dirname "$0")/.."
[[ $(gitleaks version) == '8.30.1' ]] || { echo 'Requires gitleaks 8.30.1' >&2; exit 1; }
# Scan tracked files including staged/current contents, not generated dependencies or local .env files.
scan_dir=$(mktemp -d)
trap 'rm -rf "$scan_dir"' EXIT
git ls-files -z | while IFS= read -r -d '' file; do
  [[ -f "$file" ]] || continue
  mkdir -p "$scan_dir/$(dirname "$file")"
  cp "$file" "$scan_dir/$file"
done
gitleaks dir "$scan_dir" --redact=100 --no-banner
