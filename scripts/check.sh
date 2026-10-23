#!/usr/bin/env bash
# Build-check entry point: same gates as CI (make check). Prints only errors or BUILD OK.
log="$(mktemp)"
trap 'rm -f "$log"' EXIT
if make check >"$log" 2>&1; then
  echo "BUILD OK"
else
  echo "FAILED: make check"
  tail -n 40 "$log"
  exit 1
fi
