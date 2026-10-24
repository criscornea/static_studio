# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## What this is

Static Studio: a local desktop-style editor for static-site-generator projects (Hugo is supported; Astro is detected but rejected). A single Go binary serves a JSON API plus an embedded React 19 SPA on `127.0.0.1:7777`. `ROADMAP.md` lists what is built, what is next, and deliberately out of scope; shipped items move to `CHANGELOG.md`. Check "Known shortcuts to revisit" there before assuming something is a bug (e.g. `ContentDir`/`AssetDir` are read from the top level of the Hugo config only).

## Commands

Use the Makefile (`make help`):

- `scripts/check.sh` — build-check entry point for agents; runs `make check`
- `make check` — everything CI runs (tidy, fmt, lint, race tests, frontend check)
- `make test` / `make race` — Go tests (`go test ./cmd/... ./internal/... ./web`)
- Single Go test: `go test ./internal/content -run TestName`
- `make lint` / `make fmt` — `golangci-lint run` / `golangci-lint fmt`
- `make server` — run the server (`go run ./cmd/studio`, flag `-addr`)
- `make run` — run the server and the Vite dev server together; open http://localhost:5173, Ctrl+C stops both
- `make web-install` (`npm ci`), `make web-dev` (Vite on :5173, proxies `/api` to :7777), `make web-build`, `make web-check` (type-check + lint + vitest)
- Single frontend test: `cd web && npx vitest --run src/stores/__tests__/page.spec.ts`
- `make build` embeds the real frontend; `make build-dev` builds without it.

Go targets depend on `web-stub`, which writes a placeholder `web/dist/index.html` so `//go:embed all:dist` in `web/embed.go` resolves without a Node build. Dev workflow: `make run` (or `make server` and `make web-dev` side by side).

Note: the frontend `lint` script runs with `--fix`, so it modifies files.

## Architecture

Backend (`internal/`), wired in `cmd/studio/main.go` → `server.New(logger, web.Assets()).Routes()`:

- `ssg` — `SSG` interface (Hugo/Astro implementations) and `Detect(dir)`, which yields an `ssg.Project` (root, kind, config file, content/asset dirs). Unsupported-but-recognised generators return `ErrUnsupported`.
- `project` — `Manager` owns the single open project (mutex-guarded). Opening creates an `os.Root`; **all filesystem access to project files must go through `Open.Root()`** so traversal, absolute paths and symlink escapes are refused. Each open gets a random ID; API calls carry it and mismatches return `ErrStaleID`.
- `content` — walks the content dir into a tree (`tree.go`), loads a single page (`page.go`), searches pages (`search.go`) and creates pages from archetypes (`archetype.go`, the only code that writes project files so far), operating on the confined root.
- `frontmatter` — splits and parses YAML/TOML frontmatter from markdown.
- `server` — `net/http` mux with method-prefixed patterns (`api.go` handlers, `server.go` routing). Unknown `/api/*` returns a JSON 404; everything else is an SPA handler over the embedded assets with index.html fallback (or a "no UI" message when assets are nil).

Frontend (`web/`, React 19 + TS + Zustand + Vite + Tailwind v4 + shadcn/ui): `src/api/client.ts` and `types.ts` mirror the Go JSON shapes (keep in sync when changing API structs); state lives in `src/stores/{project,page,search,view}.ts`; pure tree sorting/filtering is in `src/lib/tree.ts`; components are `ProjectPicker`, `Sidebar` (search, sort/filter, `ContentTree`, `NewPageDialog`), `PageView` (`src/components/ui/` is generated shadcn code; add more with `yes n | npx shadcn@latest add <name>`, which declines overwriting existing components; afterwards replace `from "cn"` with `from "@/lib/utils"` in the new files and run `npm uninstall cn`, because the generator pulls in an unrelated third-party package). `web/embed.go` exposes `dist/` to the Go binary.

CI (`.github/workflows/ci.yml`) has a Go job (tidy/gofmt diff, vet, race tests, golangci-lint) and a web job (type-check, lint, build, then `go build ./...` with the real embed).

## Claude workflow

Tooling copied from `vibe_starter` lives in `.claude/` (agents, skills, hooks, status line) and `scripts/`.

- Build check: `.claude/skills/build-check/check.sh` runs `scripts/check.sh` (`make check`, the same gates as CI) and prints only failures or `BUILD OK`.
- `ROADMAP.md` is the long-term list (built, next, out of scope, known shortcuts). `PLAN.md` breaks it into phases and milestones. When a phase is done, tick its items in `ROADMAP.md` and move shipped items to `CHANGELOG.md`.
- `PLAN.md` drives `/run-plan`: the `planner` agent (Opus) writes it, `implementer` (Sonnet) builds one phase per `feat/p<N>-<slug>` branch with one commit per milestone, `code-reviewer` (Opus) reviews only when the user agrees, then a local squash merge into `main`. Derive phases from `ROADMAP.md`; `/kickoff` is for new projects.
- `/run-plan` is manual by default: it stops at `Status: ready` after each phase so the user can test the branch; the next `/run-plan` merges it. `/run-plan auto` sets `autopilot: on` and runs all phases without stopping. Merges stay local: `main` is pushed only after the user confirms (build-check green first), because every push to `main` runs CI.
- Need the user during an autopilot run? Write `BLOCKED: <reason>` in `PLAN.md` and stop.
- Hooks: `guard.sh` blocks force pushes, `reset --hard`, `clean -f`, deleting main and `rm -r` on broad paths; `format.sh` formats each edited file (gofmt/goimports, local prettier); `stop-continue.sh` keeps the loop going while `PLAN.md` says `autopilot: on`.

## Token rules

- Run the `repo-man` skill before exploring unfamiliar code. Open only the files the map points to.
- Read file ranges, not whole large files. Do not re-read files you just edited.
- Build/test output: errors only. Formatting runs automatically after each edit; never fix style by hand.
- context7: only for third-party library APIs you are unsure about (version-specific). Not for the standard library.
- Playwright: verify a UI milestone once at the end. Prefer snapshots over screenshots. Close the browser when done.
- Subagents return the short report format their definition gives. No prose recaps.

## Conventions

- Conventional Commits (`feat:`, `fix:`, `chore:`, `refactor:`, `test:`, `docs:`).
- Match existing code style. No new dependencies without a reason in the commit body.
- Never commit secrets or `.env` files.
