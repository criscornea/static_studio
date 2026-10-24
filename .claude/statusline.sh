#!/usr/bin/env bash
# Status line: model | branch | context used | session cost | 5h rate limit. Costs no model tokens.
in="$(cat)"
model="$(jq -r '.model.display_name // "?"' <<<"$in")"
ctx="$(jq -r '.context_window.used_percentage // 0 | floor' <<<"$in")"
cost="$(jq -r '.cost.total_cost_usd // 0 | . * 100 | round / 100' <<<"$in")"
rate="$(jq -r '.rate_limits.five_hour.used_percentage // empty | floor' <<<"$in")"
branch="$(git -C "$(jq -r '.workspace.current_dir // "."' <<<"$in")" branch --show-current 2>/dev/null)"

out="$model"
[[ -n "$branch" ]] && out+=" | $branch"
out+=" | ctx ${ctx}% | \$${cost}"
[[ -n "$rate" ]] && out+=" | 5h ${rate}%"
echo "$out"
