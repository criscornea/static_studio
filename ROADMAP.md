# Roadmap

What's built, what's next, and what's deliberately out of scope.
Shipped items move to `CHANGELOG.md`.

Legend: `[x]` done · `[~]` in progress · `[ ]` not started

## Foundation

**Goal:** open a project and see its content.

- [x] Repo, module layout, `.gitignore`
- [x] Makefile with a single `make check`
- [x] `golangci-lint` configured and green
- [x] GitHub Actions CI mirroring `make check`
- [x] `CHANGELOG.md`
- [x] Structured logging (`log/slog`) from day one
- [x] React + TypeScript + Vite scaffold (migrated from Vue), dev proxy to the Go backend
- [x] Frontend embedded into the binary (`go:embed`), SPA fallback routing
- [x] Project detection: Hugo supported, Astro recognised but rejected
- [x] Path confinement via `os.Root` (traversal, symlinks, absolute paths)
- [x] Project lifecycle API: `open` / `get` / `close`, with a per-open id
- [x] Content tree walker (`internal/content`)
- [x] `GET /api/content` endpoint
- [x] File tree in the UI
- [x] Folder picker in the UI

🎯 **Milestone:** choose a folder → see a tree of all content files.
📓 **Blog:** "Why Static Studio, aka: who needs an IDE anyway?"

## Content & frontmatter

**Goal:** manage files, not just look at them.

- [x] Frontmatter parser, YAML and TOML
- [x] Read Hugo config for the real `contentDir` (and `staticDir`)
- [x] Show metadata in the UI: title, date, tags, draft status
- [x] Sort, filter, full-text search across content
- [x] Create a page from an archetype
- [ ] CSRF hardening: `Origin` check on mutating endpoints, `Host` check
  against DNS rebinding
- [ ] Rename and delete a page
- [ ] Atomic writes (temp file + rename)

🎯 **Milestone:** content management without a text editor.
📓 **Blog:** "All about the Frontmatter (current wip)"

## Editor & preview

**Goal:** the core. This is where it becomes real.

- [ ] CodeMirror 6 as the markdown editor
- [ ] Frontmatter as a **form**, not as text
- [ ] Preview: the real Hugo-rendered page from `hugo server`, loaded in an
  iframe from its own port (separate origin from the app)
- [ ] Drag & drop images into `static/`, path inserted into the markdown
- [ ] Autosave with debounce, unsaved-changes state, editor undo

⚠️ The editor stays raw markdown. Preview is template-accurate because Hugo
renders it, not the app.

🎯 **Milestone:** a post can be written end to end without a terminal.
📓 **Blog:** "The editor comes to live"

## Build & serve

**Goal:** the hardest part.

- [ ] Run `hugo` as a subprocess with context cancellation
- [ ] Stream build output live to the UI (SSE or WebSocket)
- [ ] Start/stop the dev server, show status, proxy the preview
- [ ] Parse build errors and surface them at the right file
- [ ] Clean shutdown: no orphaned processes, project root closed

🎯 **Milestone:** edit → build → see the result, all in the app.
📓 **Blog:** "Subprozesse, Streaming und Context — Go at work"

## Publish

**Goal:** one way out. **One.**

- [ ] Target: GitHub Pages, pushing `public/` to `gh-pages` via the system `git`
- [ ] Configure it in the UI
- [ ] No stored credentials: rely on the user's SSH key or git credential helper
- [ ] Show progress and errors
- [ ] No terminal from "typing text" to "site is live"

🎯 **Milestone:** a real website published entirely from the tool.
📓 **Blog:** "On to version 1"

## Harden and ship

No new features. Zero.

- [ ] Raise coverage where failure hurts: writes, paths, processes
- [ ] Error messages readable by non-developers, end to end
- [ ] `-log-level` and `-log-format=json` flags
- [ ] README: problem, screenshots, quickstart, architecture, scope decisions
- [ ] **Screencast, 2–3 minutes**
- [ ] Release binaries for macOS/Linux/Windows, tag `v1.0.0`
- [ ] Closing blog post

🎯 **Milestone:** v1.0.0 released, screencast online.

## Explicitly out of scope for v1.0

Plugin system · theme designer · Git integration · multi-user · cloud sync ·
custom markdown parser · **Astro editing support** · mobile · collaboration ·
auth · undo across sessions

## Known shortcuts to revisit

Deliberate simplifications, recorded so they don't become invisible.

- `ContentDir` and `AssetDir` come from the top level of the detected Hugo
  config file only. Per-language `contentDir`, module mounts, merged
  `config/_default/*` files and `HUGO_*` environment overrides are ignored,
  and an unreadable or invalid config silently falls back to the defaults.
- Search reads every page on each query, and the content tree reads every
  page's frontmatter on each load, with no index or cache. Search is a
  case-insensitive substring match over title, tags and body (no stemming,
  no ranking beyond title-and-tag matches first, capped at 50 hits).
- The content tree reads every page's frontmatter on each load. Fine for a
  personal site; cache it if a project with thousands of pages gets slow.
- Archetypes are rendered by a built-in `text/template` with `.Name`, `.Type`,
  `.Date`, `.File.ContentBaseName` and the functions `replace`, `title`,
  `lower`, `upper` and `now`, not by Hugo. A template using anything else is
  refused with a readable message. `title` capitalises every word instead of
  following Hugo's title-case style. Archetype folders (page bundles) and a
  custom `archetypeDir` are not supported. A new page's title comes from the
  archetype (the file name, title-cased), not from what the user typed.
- New pages are created with `O_EXCL` and removed again if the write fails,
  but not yet via temp file + rename; that is the atomic-writes item.
- The content tree only lists `.md`, `.markdown`, `.mdx` and `.html`.
- `Manager.Close` is not called on server shutdown; it becomes relevant
  alongside subprocess cleanup.
- `internal/server` writes responses via two `Server` methods. If that layer
  grows, extract an `internal/httpx` package.
- No `Origin` check on mutating endpoints yet.
