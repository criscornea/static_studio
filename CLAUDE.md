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
- `make run` — run the server (`go run ./cmd/studio`, flag `-addr`)
- `make web-install` (`npm ci`), `make web-dev` (Vite on :5173, proxies `/api` to :7777), `make web-build`, `make web-check` (type-check + lint + vitest)
- Single frontend test: `cd web && npx vitest --run src/stores/__tests__/page.spec.ts`
- `make build` embeds the real frontend; `make build-dev` builds without it.

Go targets depend on `web-stub`, which writes a placeholder `web/dist/index.html` so `//go:embed all:dist` in `web/embed.go` resolves without a Node build. Dev workflow: run `make run` and `make web-dev` side by side.

Note: the frontend `lint` script runs with `--fix`, so it modifies files.

## Architecture

Backend (`internal/`), wired in `cmd/studio/main.go` → `server.New(logger, web.Assets()).Routes()`:

- `ssg` — `SSG` interface (Hugo/Astro implementations) and `Detect(dir)`, which yields an `ssg.Project` (root, kind, config file, content/asset dirs). Unsupported-but-recognised generators return `ErrUnsupported`.
- `project` — `Manager` owns the single open project (mutex-guarded). Opening creates an `os.Root`; **all filesystem access to project files must go through `Open.Root()`** so traversal, absolute paths and symlink escapes are refused. Each open gets a random ID; API calls carry it and mismatches return `ErrStaleID`.
- `content` — walks the content dir into a tree (`tree.go`) and loads a single page (`page.go`), operating on the confined root.
- `frontmatter` — splits and parses YAML/TOML frontmatter from markdown.
- `server` — `net/http` mux with method-prefixed patterns (`api.go` handlers, `server.go` routing). Unknown `/api/*` returns a JSON 404; everything else is an SPA handler over the embedded assets with index.html fallback (or a "no UI" message when assets are nil).

Frontend (`web/`, React 19 + TS + Zustand + Vite + Tailwind v4 + shadcn/ui): `src/api/client.ts` and `types.ts` mirror the Go JSON shapes (keep in sync when changing API structs); state lives in `src/stores/{project,page}.ts`; components are `ProjectPicker`, `ContentTree`, `PageView` (`src/components/ui/` is generated shadcn code; add more with `npx shadcn@latest add <name>`). `web/embed.go` exposes `dist/` to the Go binary.

CI (`.github/workflows/ci.yml`) has a Go job (tidy/gofmt diff, vet, race tests, golangci-lint) and a web job (type-check, lint, build, then `go build ./...` with the real embed).

## Planning

- `ROADMAP.md` is the long-term list (built, next, out of scope, known shortcuts). `PLAN.md` breaks the current ROADMAP section into phases and milestones.
- When a phase is done, tick its items in `ROADMAP.md` and move shipped items to `CHANGELOG.md`.

## Workflow

- `PLAN.md` is the source of truth for the current work: phases, milestones, status. Keep it current.
- Planning: Opus (`planner` agent). Coding: Sonnet (`implementer` agent). Review: Opus (`code-reviewer` agent), only when the user agrees.
- One branch per phase: `feat/p<N>-<slug>`. One commit per milestone. Push after each commit if a remote exists.
- Merge into `main` only with a local squash merge, after the optional review.
- Use `/run-plan` to execute PLAN.md.
- Need the user? Write `BLOCKED: <reason>` in PLAN.md and stop.

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
