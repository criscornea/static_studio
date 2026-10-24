#!/usr/bin/env bash
# Stop hook: keep the coordinator working while PLAN.md has "autopilot: on" and unfinished phases.
# Lets Claude stop when: no plan, autopilot off, BLOCKED: present, all phases merged, or no progress since the last nudge.
cd "${CLAUDE_PROJECT_DIR:-.}" || exit 0
input="$(cat)"
plan=PLAN.md
state=.claude/.stop-state

[[ -f $plan ]] || exit 0
grep -Eq '^autopilot:[[:space:]]*on' "$plan" || exit 0
grep -Eq '^[[:space:]]*BLOCKED:' "$plan" && exit 0
grep -Eq '^Status:[[:space:]]*(todo|in-progress|review)' "$plan" || exit 0

# Progress = any change to PLAN.md or any branch head.
sig="$( { cat "$plan"; git rev-parse --all 2>/dev/null; } | md5sum | cut -d' ' -f1)"
if [[ "$(jq -r '.stop_hook_active // false' <<<"$input")" == "true" && -f $state && "$(cat "$state")" == "$sig" ]]; then
  rm -f "$state"
  exit 0  # nudged once already and nothing changed: let the user take over
fi
echo "$sig" > "$state"

jq -n '{decision: "block", reason: "PLAN.md has autopilot: on and unfinished phases. Continue the run-plan loop with the next phase. If you need the user, write BLOCKED: <reason> in PLAN.md and stop."}'
