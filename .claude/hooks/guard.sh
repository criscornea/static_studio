#!/usr/bin/env bash
# PreToolUse(Bash): block destructive commands during unattended runs. Exit 2 = block, stderr goes to Claude.
cmd="$(jq -r '.tool_input.command // empty')"

block() { echo "Blocked by .claude/hooks/guard.sh: $1. Find a non-destructive way or write BLOCKED: in PLAN.md." >&2; exit 2; }

grep -Eq 'git[[:space:]].*push.*(--force|[[:space:]]-f([[:space:]]|$)|[[:space:]]\+[^[:space:]])' <<<"$cmd" && block "force push"
grep -Eq 'git[[:space:]].*reset[[:space:]].*--hard' <<<"$cmd"                       && block "git reset --hard"
grep -Eq 'git[[:space:]].*clean[[:space:]].*-[a-zA-Z]*f' <<<"$cmd"                    && block "git clean -f"
grep -Eq 'git[[:space:]].*branch[[:space:]].*-D[[:space:]]+(main|master)\b' <<<"$cmd" && block "deleting main"
grep -Eq 'git[[:space:]].*push.*(--delete[[:space:]]+|[[:space:]]-d[[:space:]]+|[[:space:]]:)(main|master)\b' <<<"$cmd"     && block "deleting remote main"
grep -Eq 'rm[[:space:]]+-[a-zA-Z]*[rR][a-zA-Z]*[[:space:]]+(-[^[:space:]]+[[:space:]]+)*("?(/|~|\$HOME|\.\.?|\*)"?/?([[:space:]]|$))' <<<"$cmd" && block "rm -r on /, ~, ., .. or *"
exit 0
