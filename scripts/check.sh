#!/usr/bin/env bash
# Project build check used by .claude/skills/build-check/check.sh. Prints only failures, or "BUILD OK".
# Mirrors CI minus tidy/fmt diffs; note the frontend lint runs with --fix.
set -uo pipefail
cd "$(git rev-parse --show-toplevel)"

log="$(mktemp)"
trap 'rm -f "$log"' EXIT
failed=0

run() {
  if ! "$@" >"$log" 2>&1; then
    echo "FAIL $*"
    tail -n 40 "$log"
    echo
    failed=1
  fi
}

run make web-stub
run go vet ./...
run go build -o /dev/null ./...
run go test ./cmd/... ./internal/... ./web
command -v golangci-lint >/dev/null && run golangci-lint run
run make web-check

[[ $failed -eq 0 ]] && echo "BUILD OK"
exit "$failed"
