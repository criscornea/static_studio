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
- [x] Vue 3 + TypeScript + Vite scaffold, dev proxy to the Go backend
- [x] Frontend embedded into the binary (`go:embed`), SPA fallback routing
- [x] Project detection: Hugo supported, Astro recognised but rejected
- [x] Path confinement via `os.Root` (traversal, symlinks, absolute paths)
- [x] Project lifecycle API: `open` / `get` / `close`, with a per-open id
- [x] Content tree walker (`internal/content`)
- [x] `GET /api/content` endpoint
- [x] File tree in the Vue UI
- [x] Folder picker in the UI

🎯 **Milestone:** choose a folder → see a tree of all content files.
📓 **Blog:** "Why Static Studio, aka: who needs an IDE anyway?"

## Content & frontmatter

**Goal:** manage files, not just look at them.

- [ ] Frontmatter parser, YAML and TOML
- [ ] Read Hugo config for the real `contentDir` (currently hardcoded)
- [ ] Show metadata in the UI: title, date, tags, draft status
- [ ] Sort, filter, full-text search across content
- [ ] Create a page from an archetype, rename, delete
- [ ] Atomic writes (temp file + rename)

🎯 **Milestone:** content management without a text editor.
📓 **Blog:** "All about the Frontmatter (current wip)"

## Editor & preview

**Goal:** the core. This is where it becomes real.

- [ ] CodeMirror 6 as the markdown editor
- [ ] Frontmatter as a **form**, not as text
- [ ] Live preview rendered by the backend
- [ ] Drag & drop images into `static/`, path inserted into the markdown
- [ ] Autosave with debounce, unsaved-changes state, editor undo

⚠️ Render markdown only. Template-accurate preview is a v2 topic.

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

- [ ] Pick the target: GitHub Pages *or* Netlify *or* rsync/SFTP
- [ ] Configure it in the UI
- [ ] Store credentials safely, never plaintext in the project folder
- [ ] Show progress and errors
- [ ] No terminal from "typing text" to "site is live"

🎯 **Milestone:** a real website published entirely from the tool.
📓 **Blog:** "On to version 1"

## Harden and ship

No new features. Zero.

- [ ] Raise coverage where failure hurts: writes, paths, processes
- [ ] Error messages readable by non-developers, end to end
- [ ] CSRF hardening: `Origin` header check on mutating endpoints
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

- `ContentDir` and `AssetDir` are hardcoded to Hugo defaults; the config is
  not parsed yet. Fix alongside the frontmatter work.
- The content tree only lists `.md`, `.markdown`, `.mdx` and `.html`.
- `Manager.Close` is not called on server shutdown; it becomes relevant
  alongside subprocess cleanup.
- `internal/server` writes responses via two `Server` methods. If that layer
  grows, extract an `internal/httpx` package.
- No `Origin` check on mutating endpoints yet.
