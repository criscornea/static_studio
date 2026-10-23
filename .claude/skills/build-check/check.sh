#!/usr/bin/env bash
# Stack-agnostic build check. Prints only failures, or "BUILD OK".
# A project-specific scripts/check.sh overrides auto-detection.
set -uo pipefail

root="$(git rev-parse --show-toplevel 2>/dev/null || pwd)"
cd "$root"

if [[ -x scripts/check.sh ]]; then
  exec scripts/check.sh
fi

log="$(mktemp)"
trap 'rm -f "$log"' EXIT
ran=0
failed=0

run() { # run <dir> <command...>
  local dir="$1"; shift
  ran=1
  if ! (cd "$dir" && "$@") >"$log" 2>&1; then
    echo "FAIL [$dir] $*"
    tail -n 40 "$log"
    echo
    failed=1
  fi
}

# Project roots: repo root plus subdirectories up to depth 2 (monorepos, backend/ + frontend/).
mapfile -t manifests < <(find . -maxdepth 3 \
  \( -name node_modules -o -name vendor -o -name .git -o -name dist -o -name build \) -prune -o \
  \( -name go.mod -o -name package.json -o -name composer.json \) -print | sort)

for m in "${manifests[@]}"; do
  dir="$(dirname "$m")"
  case "$(basename "$m")" in
    go.mod)
      run "$dir" go vet ./...
      run "$dir" go build -o /dev/null ./...
      run "$dir" go test ./...
      ;;
    package.json)
      if   [[ -f "$dir/pnpm-lock.yaml" ]]; then pm=pnpm
      elif [[ -f "$dir/yarn.lock" ]];      then pm=yarn
      elif [[ -f "$dir/bun.lockb" || -f "$dir/bun.lock" ]]; then pm=bun
      else pm=npm; fi
      has() { (cd "$dir" && node -e "process.exit(require('./package.json').scripts?.['$1'] ? 0 : 1)") 2>/dev/null; }
      if has typecheck; then run "$dir" "$pm" run typecheck
      elif has build;   then run "$dir" "$pm" run build; fi
      has lint && run "$dir" "$pm" run lint
      has test && CI=1 run "$dir" "$pm" run test
      ;;
    composer.json)
      [[ -x "$dir/vendor/bin/phpstan" ]] && run "$dir" vendor/bin/phpstan analyse --no-progress --error-format=raw
      if   [[ -x "$dir/vendor/bin/phpunit" ]]; then run "$dir" vendor/bin/phpunit
      elif [[ -x "$dir/bin/phpunit" ]];        then run "$dir" bin/phpunit; fi
      [[ -x "$dir/bin/console" ]] && run "$dir" php bin/console lint:container
      ;;
  esac
done

if [[ $ran -eq 0 ]]; then
  echo "NO CHECKS: no go.mod, package.json or composer.json found. Add scripts/check.sh."
  exit 0
fi
[[ $failed -eq 0 ]] && echo "BUILD OK"
exit "$failed"
