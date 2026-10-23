#!/usr/bin/env bash
# Enable per-stack plugins in .claude/settings.json and report missing language servers.
#   enable-stack.sh go ts php web
# go/ts/php: LSP plugin. web: Playwright browser MCP (for projects with a UI).
set -euo pipefail
cd "${CLAUDE_PROJECT_DIR:-$(git rev-parse --show-toplevel)}"
settings=.claude/settings.json

for s in "$@"; do
  case "$s" in
    go)  plugin=gopls-lsp;      bin=gopls;                      hint="go install golang.org/x/tools/gopls@latest" ;;
    ts)  plugin=typescript-lsp; bin=typescript-language-server; hint="npm install -g typescript-language-server typescript" ;;
    php) plugin=php-lsp;        bin=intelephense;               hint="npm install -g intelephense" ;;
    web) plugin=playwright;     bin=npx;                        hint="install Node.js; first browser run may need: npx playwright install chromium" ;;
    *)   echo "unknown stack: $s (use go, ts, php, web)" >&2; exit 1 ;;
  esac
  tmp="$(mktemp)"
  jq --arg p "$plugin@claude-plugins-official" '.enabledPlugins[$p] = true' "$settings" >"$tmp" && mv "$tmp" "$settings"
  echo "enabled $plugin"
  command -v "$bin" >/dev/null || echo "MISSING $bin: $hint"
done
echo "Restart Claude Code to load the plugins."
