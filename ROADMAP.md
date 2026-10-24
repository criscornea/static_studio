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
- [x] Recent projects on the open screen, stored in the user config dir

🎯 **Milestone:** choose a folder → see a tree of all content files.
📓 **Blog:** "Why Static Studio, aka: who needs an IDE anyway?"

## Content & frontmatter

**Goal:** manage files, not just look at them.

- [x] Frontmatter parser, YAML and TOML
- [x] Read Hugo config for the real `contentDir` (and `staticDir`)
- [x] Show metadata in the UI: title, date, tags, draft status
- [x] Sort, filter, full-text search across content
- [x] Create a page from an archetype
- [x] CSRF hardening: `Origin` check on mutating endpoints, `Host` check
  against DNS rebinding
- [x] Rename and delete a page
- [x] Atomic writes (temp file + rename)

🎯 **Milestone:** content management without a text editor.
📓 **Blog:** "All about the Frontmatter (current wip)"

## Editor & preview

**Goal:** the core. This is where it becomes real.

- [x] CodeMirror 6 as the markdown editor
- [x] Frontmatter as a **form**, not as text
- [x] Preview: the real Hugo-rendered page from `hugo server`, loaded in an
  iframe from its own port (separate origin from the app)
- [x] Drag & drop images into `static/`, path inserted into the markdown
- [x] Autosave with debounce, unsaved-changes state, editor undo

⚠️ The editor stays raw markdown. Preview is template-accurate because Hugo
renders it, not the app.

🎯 **Milestone:** a post can be written end to end without a terminal.
📓 **Blog:** "The editor comes to live"

## Build & serve

**Goal:** the hardest part.

- [x] Run `hugo` as a subprocess with context cancellation
- [x] Stream build output live to the UI (SSE or WebSocket)
- [x] Start/stop the dev server, show status (preview loads from its own port, no proxy)
- [x] Parse build errors and surface them at the right file
- [x] Clean shutdown: no orphaned processes, project root closed

🎯 **Milestone:** edit → build → see the result, all in the app.
📓 **Blog:** "Subprozesse, Streaming und Context — Go at work"

## Publish

**Goal:** one way out. **One.**

- [x] Target: GitHub Pages, pushing `public/` to `gh-pages` via the system `git`
- [x] Configure it in the UI
- [x] No stored credentials: rely on the user's SSH key or git credential helper
- [x] Show progress and errors
- [x] No terminal from "typing text" to "site is live"

🎯 **Milestone:** a real website published entirely from the tool.
📓 **Blog:** "On to version 1"

## Harden and ship

No new features. Zero.

- [x] Raise coverage where failure hurts: writes, paths, processes
- [x] Error messages readable by non-developers, end to end
- [x] `-log-level` and `-log-format=json` flags
- [x] README: problem, screenshots, quickstart, architecture, scope decisions
- [ ] **Screencast, 2–3 minutes**
- [x] Release binaries for macOS/Linux/Windows, tag `v1.0.0`
- [ ] Closing blog post

🎯 **Milestone:** v1.0.0 released, screencast online.

## v1.2

**Goal:** write comfortably in the editor and publish to any server.

- [x] Floating formatting toolbar (Obsidian style): bottom-center bar with
  H1-H3, bold, italic, strikethrough, inline code, link, bullet and numbered
  list, quote; built on CodeMirror commands; active-state highlighting;
  Ctrl/Cmd+B, I, K in the editor; reachable from the command palette; themed
  for light and dark; hidden when the editor is unfocused
- [x] Live preview in the editor (Obsidian-style hybrid): markup shown only on
  the cursor line or selection, headings, emphasis, links, inline code, lists,
  quotes and horizontal rules rendered inline via CodeMirror decorations;
  persisted toggle between live preview and source mode; the document text is
  never altered
- [x] rsync/SFTP publish target alongside GitHub Pages: target setting
  (`github-pages` | `rsync`), destination `user@host:path`, port, optional
  identity file, delete and dry-run flags; system `rsync`/`ssh` with argv only
  (no shell); no stored credentials (ssh-agent and keys); progress through the
  existing publish stream; readable errors; host keys verified, never
  `StrictHostKeyChecking=no`

🎯 **Milestone:** format, preview and publish a post to your own server without leaving the app.

## Explicitly out of scope for v1.0

Plugin system · theme designer · Git integration · multi-user · cloud sync ·
custom markdown parser · **Astro editing support** · mobile · collaboration ·
auth · undo across sessions

## Known shortcuts to revisit

Deliberate simplifications, recorded so they don't become invisible.

- The rsync publish target needs the system `rsync` (and `ssh` for remote
  destinations) on the PATH. Fields are limited to a safe character set, so
  paths with spaces or special characters, identity files with spaces and
  `~` in remote paths are refused. SFTP-only servers are not supported, and
  the host key must already be in `known_hosts` (connect once with `ssh`).
- Preview URLs are derived from the primary content dir only; pages in
  other content dirs (per-language) get no preview link until per-language
  URL mapping lands.

- `ContentDir` and `AssetDir` come from the top level of the merged Hugo
  config (root file, `config/_default/*`, `config/<environment>/*`,
  `HUGO_CONTENTDIR`, `HUGO_STATICDIR`). Per-language `contentDir` and
  project-local `module.mounts` add content dirs (shown in the tree and
  searched) and pages in all of them can be edited, created, renamed and
  deleted; imported-module mounts are ignored. An unreadable, oversized or invalid config now stops the project from
  opening with a readable error.
- The `/api` host guard accepts a loopback Host (`127.0.0.1`, `::1`,
  `localhost`) on any port, not only the server's own. Rebinding still fails
  because a foreign name is rejected, and the Vite dev proxy forwards
  `Host: localhost:5173`, which a fixed-port check would break.
- Saving, renaming and deleting are serialised by an in-process mutex, and
  save and delete verify the hash on a file moved aside (grab-then-verify), so
  an external edit is never overwritten or deleted unseen. The remaining
  gap: a sub-millisecond window in rename between the identity check and
  removing the old name, and no protection on filesystems where renaming a
  file that another process holds open fails.
- The page index (`content.Index`, one per open project) caches frontmatter
  fields and search text per page, validated by size, mtime and file identity
  on every lookup (a stat per page; no file watcher). A write that keeps the
  size and mtime and replaces the file in place is missed. Search is a
  case-insensitive substring match over title, tags and body (no stemming,
  no ranking beyond title-and-tag matches first, capped at 50 hits). The
  index holds page text in memory, so very large sites use proportional RAM.
- Archetypes are rendered by a built-in `text/template` with `.Name`, `.Type`,
  `.Date`, `.File.ContentBaseName` and the functions `replace`, `title`,
  `lower`, `upper` and `now`, not by Hugo. A template using anything else is
  refused with a readable message. `title` capitalises every word instead of
  following Hugo's title-case style. Archetype folders (page bundles) and a
  custom `archetypeDir` are not supported. A new page's title comes from the
  archetype (the file name, title-cased), not from what the user typed.
- The content tree only lists `.md`, `.markdown`, `.mdx` and `.html`.
- On a filesystem without hard links, `RenamePage` falls back to Lstat then
  rename, and a restore after a conflicting save or delete falls back the
  same way, so a file created at the target in between could be overwritten
  (TOCTOU). With hard links nothing is overwritten.
- The preview URL for a page follows Hugo's default permalinks and path
  cleaning (lowercase, spaces to hyphens) only. Front matter `url` and
  `slug`, language subdirectories or suffixes, custom `permalinks`, and
  `disablePathToLower` from the config are not read.
- The dev server port is picked by binding and releasing a free port, then
  passed to Hugo; another process can grab it in between (TOCTOU). Hugo then
  exits and its output is shown as an error.
- `internal/server` writes responses via two `Server` methods. If that layer
  grows, extract an `internal/httpx` package.
