#!/usr/bin/env bash
# PostToolUse(Edit|Write): format the edited file with the project's own formatter. Silent; never blocks.
f="$(jq -r '.tool_input.file_path // empty')"
[[ -n "$f" && -f "$f" ]] || exit 0
cd "${CLAUDE_PROJECT_DIR:-.}" || exit 0

# Nearest ancestor dir of the file that contains $1 (handles monorepos like web/ + api/).
up() { local d; d="$(dirname "$f")"; while [[ "$d" == "$PWD"* ]]; do [[ -e "$d/$1" ]] && { echo "$d"; return; }; d="$(dirname "$d")"; done; }

case "$f" in
  *.go)
    d="$(up go.mod)"   # group the module's own imports separately, like golangci-lint local-prefixes
    mod="$([[ -n "$d" ]] && awk '/^module /{print $2; exit}' "$d/go.mod")"
    if command -v goimports >/dev/null; then goimports ${mod:+-local "$mod"} -w "$f"; elif command -v gofmt >/dev/null; then gofmt -w "$f"; fi
    ;;
  *.ts|*.tsx|*.js|*.jsx|*.mjs|*.cjs|*.vue|*.css|*.scss|*.html|*.json|*.yaml|*.yml)
    d="$(up node_modules/.bin/prettier)"   # only a locally installed prettier, never an npx download
    [[ -n "$d" ]] && "$d/node_modules/.bin/prettier" --write --log-level silent "$f"
    ;;
  *.php)
    d="$(up vendor/bin/php-cs-fixer)"
    [[ -n "$d" ]] && (cd "$d" && vendor/bin/php-cs-fixer fix --quiet "$f")
    ;;
esac >/dev/null 2>&1
exit 0
