#!/usr/bin/env bash
# Print Aider's repo map without the banner. Needs a git repo with at least one tracked file.
set -euo pipefail

tokens="${1:-2048}"

if ! command -v aider >/dev/null 2>&1; then
  echo "repo-man: aider not installed. Run scripts/bootstrap.sh" >&2
  exit 1
fi
if ! git rev-parse --is-inside-work-tree >/dev/null 2>&1; then
  echo "repo-man: not a git repo" >&2
  exit 1
fi

aider --show-repo-map \
  --map-tokens "$tokens" \
  --no-gitignore --no-check-update --no-analytics --no-show-model-warnings \
  --chat-history-file /dev/null --input-history-file /dev/null \
  --aiderignore "$(dirname "$0")/aiderignore" \
  2>/dev/null \
  | sed -n '/^Here are summaries/,$p' | tail -n +4 | grep -v '^$'
