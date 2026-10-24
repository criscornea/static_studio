#!/usr/bin/env bash
# SessionStart: silent when all is fine; stdout lands in Claude's context, so print only problems.
missing=()
for t in git jq aider; do command -v "$t" >/dev/null 2>&1 || missing+=("$t"); done
cd "${CLAUDE_PROJECT_DIR:-.}" 2>/dev/null
git rev-parse --is-inside-work-tree >/dev/null 2>&1 || missing+=("git repo")
[[ ${#missing[@]} -eq 0 ]] || echo "vibe_starter: missing ${missing[*]}. Tell the user to run scripts/bootstrap.sh."
exit 0
