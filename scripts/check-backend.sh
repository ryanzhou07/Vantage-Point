#!/usr/bin/env bash
set -euo pipefail
cd "$(dirname "$0")/../services/backend"
files=$(find . -name '*.go' -not -path './vendor/*' -print)
if [[ -n "$files" ]]; then
  drift=$(printf '%s\n' "$files" | while IFS= read -r file; do gofmt -l "$file"; done)
  if [[ -n "$drift" ]]; then printf 'Go formatting drift:\n%s\n' "$drift" >&2; exit 1; fi
fi
go vet ./...
go test -race ./...
go build ./...
