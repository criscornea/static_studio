#!/usr/bin/env bash
# Install the tools vibe_starter needs and, optionally, copy the starter into a project.
#   scripts/bootstrap.sh              check/install tools only
#   scripts/bootstrap.sh <target-dir> also copy the starter into <target-dir> and git init it
# Existing files in the target are never overwritten.
set -euo pipefail

starter="$(cd "$(dirname "$0")/.." && pwd)"
target="${1:-}"

ok()   { echo "  ok    $*"; }
warn() { echo "  MISS  $*"; }

echo "Tools:"
for t in git jq; do
  command -v "$t" >/dev/null && ok "$t" || warn "$t (install with your package manager, e.g. sudo apt install $t)"
done

if command -v uv >/dev/null; then ok uv
else warn "uv: install with: curl -LsSf https://astral.sh/uv/install.sh | sh  (then re-run this script)"; fi

if command -v aider >/dev/null; then ok "aider $(aider --version 2>/dev/null | awk '{print $2}')"
elif command -v uv >/dev/null; then
  echo "  ...   installing aider (repo-man) via uv"
  uv tool install --python 3.12 aider-chat && ok aider
else warn "aider: needs uv first"; fi

if command -v claude >/dev/null; then
  ok "claude $(claude --version 2>/dev/null | awk '{print $1}')"
  if claude plugin marketplace list 2>/dev/null | grep -qi caveman; then ok "caveman marketplace"
  else claude plugin marketplace add JuliusBrussee/caveman && ok "caveman marketplace added"; fi
  if claude plugin list 2>/dev/null | grep -qi 'caveman@caveman'; then ok "caveman plugin"
  else claude plugin install caveman@caveman && ok "caveman plugin installed"; fi
else warn "claude (Claude Code CLI)"; fi

echo "Optional (per stack, /kickoff enables the matching plugin):"
opt() { command -v "$1" >/dev/null && ok "$1" || echo "  opt   $1: $2"; }
opt gopls "go install golang.org/x/tools/gopls@latest"
opt typescript-language-server "npm install -g typescript-language-server typescript"
opt intelephense "npm install -g intelephense"
opt npx "install Node.js (needed by the Playwright plugin)"

[[ -z "$target" ]] && exit 0

echo "Project: $target"
mkdir -p "$target"
target="$(cd "$target" && pwd)"
[[ "$target" == "$starter" ]] && { echo "  target is the starter itself, skipping copy"; exit 0; }

copy() { # copy <relative path>, never overwrite
  local src="$starter/$1" dst="$target/$1"
  if [[ -e "$dst" ]]; then echo "  keep  $1 (exists)"; return; fi
  mkdir -p "$(dirname "$dst")"
  cp -r "$src" "$dst" && echo "  add   $1"
}
copy CLAUDE.md
copy .gitignore
copy scripts/bootstrap.sh
for f in $(cd "$starter" && find .claude -type f ! -name settings.local.json ! -name .stop-state); do copy "$f"; done
chmod +x "$target"/.claude/hooks/*.sh "$target"/.claude/skills/*/*.sh 2>/dev/null || true

cd "$target"
if [[ ! -d .git ]]; then git init -q -b main && ok "git init (main)"; fi
if ! git rev-parse HEAD >/dev/null 2>&1; then
  git add -A && git commit -qm "chore: bootstrap from vibe_starter" && ok "initial commit"
fi
git remote | grep -q . && ok "remote $(git remote get-url origin 2>/dev/null)" \
  || warn "no git remote: pushes are skipped. Add one with: git remote add origin <url>"

echo "Next: cd $target && claude, then /kickoff"
