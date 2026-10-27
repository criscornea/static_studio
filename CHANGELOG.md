# Changelog

All notable changes to this project are documented here.
Format follows [Keep a Changelog](https://keepachangelog.com/en/1.1.0/).

## [Unreleased]

### Added

- rsync over SSH publish target alongside GitHub Pages. The publish dialog has
  a target picker with destination (`user@host:path` or a full local path),
  port, optional identity file, a delete option (with a warning) and a dry
  run. The system `rsync` runs with an argv only, no shell, from `public/`
  without following symlinks. SSH uses `BatchMode`, your agent, config and
  `known_hosts`; host key checking is never turned off and no credentials are
  stored. Unknown host keys, refused logins, missing programs and unreachable
  hosts give readable messages. Existing settings load as GitHub Pages.
- Removed the live preview mode before release. The editor always shows plain
  markdown source; the switch in the header, status bar and command palette is
  gone.
- Floating formatting toolbar at the bottom center of the editor: H1-H3, bold,
  italic, strikethrough, inline code, link, bullet and numbered lists and
  quote. Active formats are highlighted from the markdown syntax tree. It shows
  only while the editor has focus. Ctrl/Cmd+B, I and K format inside the
  editor; outside it Mod+B still toggles the sidebar and Mod+K focuses search.
  Every action is also in the command palette (group Format) and the shortcuts
  dialog marks the editor-only bindings. The editor now parses GFM markdown.

### Changed

- The content tree and search read pages through a per-project cached index
  validated by size, mtime and file identity, so unchanged pages are not read
  again. On a generated 5,000-page site a warm tree load is about 35x faster
  and a warm search about 30x faster. External edits are still picked up on
  the next load.
- Save and delete move the file aside, verify its hash there and only then
  replace or remove it, so a change made by an external editor in between is
  reported as a conflict instead of being lost. A file recreated at the name
  is kept. Rename verifies the moved file is the one it checked.
- `DELETE /api/page` requires `hash`, the hash of the page as last read; the
  delete dialog reads the page first when it is not open.

### Added

- Pages in every resolved content dir can be opened, saved, created, renamed
  (also between dirs) and deleted; paths outside all of them, including
  sibling-prefix names like `content-old/`, are refused. `POST /api/page`
  takes an optional `dir`, and the new-page dialog has a content folder picker.
- Every Hugo content dir is resolved (per-language `contentDir`, project-local
  content and static module mounts; unsafe sources are skipped). The sidebar
  shows one labelled tree per dir and search covers all of them.
- Desktop-style app shell: fixed viewport, title bar, activity bar,
  resizable sidebar, status bar (project, save, preview server, build).
- Light and dark themes following the OS, with a persisted override;
  compact design tokens.
- Sidebar width and collapsed state persist; Ctrl/Cmd+B toggles it.
- Project picker restyled as a centered welcome screen.
- Explorer sidebar: icon toolbar with tooltips, sort and filter menu,
  collapse all, compact search, single-line tree rows with icons, indent
  guides, draft and error markers, keyboard navigation and a right-click
  menu (open, new page here, rename, delete, copy path).
- Editor chrome: header with path breadcrumb, save state and icon actions
  (preview, preview server, build, publish) with tooltips; full-height
  CodeMirror editor following the app theme; collapsible properties panel;
  resizable side-by-side preview; resizable build log panel toggled from
  the status bar or Ctrl/Cmd+J. Panel sizes and open states persist.
- Command palette (Ctrl/Cmd+Shift+P) and quick-open for pages (Ctrl/Cmd+P),
  both backed by one command registry and one global shortcut handler that
  leaves keys to the editor and open dialogs; a keyboard shortcuts dialog
  (Ctrl/Cmd+/) lists every binding.
- Finished empty and loading states (icon, one line, action; skeletons)
  in the sidebar, search, editor, preview and build log.

### Changed

- `contentDir` and `staticDir` are resolved from the merged Hugo config:
  root file, `config/_default/*`, `config/<environment>/*` (from
  `HUGO_ENVIRONMENT`, else `development`) and the `HUGO_CONTENTDIR` and
  `HUGO_STATICDIR` environment variables.

- App chrome is not text-selectable, uses the default cursor and shows
  focus rings only for keyboard focus; editor text, logs and inputs stay
  selectable.

### Fixed

- An unreadable, oversized or invalid Hugo config now stops the project
  from opening and names the file and reason (`invalid_config`) instead of
  silently using default directories.
- Generated UI components no longer depend on the third-party `cn` package;
  they use the local helper in `web/src/lib/utils.ts`.

### Added

- `-log-level` and `-log-format` (`text` or `json`) flags.
- Readable UI messages for API error codes (`web/src/lib/errors.ts`).
- Tests for store write failures and opening a missing project folder.
- README: problem, quickstart, architecture and scope.
- Release workflow: a `v*` tag builds Linux, macOS and Windows binaries and
  attaches them to a GitHub release.
- Publish to GitHub Pages: the Publish button saves a remote and branch (kept in the
  user config dir, per project; default `gh-pages`), runs a Hugo build, then commits
  `public/` in a temporary repository and force-pushes it with the system `git`. No
  credentials are stored; the user's SSH key or credential helper signs in. Progress
  streams over `GET /api/publish/stream`; settings use `GET`/`PUT /api/publish/settings`,
  and `POST /api/publish` and `/api/publish/cancel` control the run.
- Recent projects: the open screen lists the last 10 opened projects (kept in the
  user config dir) with one-click open, remove, and greyed-out missing folders.
- Build errors: Hugo error lines are parsed into file, line and message and sent
  as `problems` on the build stream's `done` event. The tree marks affected pages,
  the build panel lists them, and clicking one opens the page and moves the editor
  to the line. Pages now also report `bodyLine`.
- Preview: start/stop a `hugo server` bound to 127.0.0.1 on a free port
  (`GET /api/devserver`, `POST /api/devserver/start`, `POST /api/devserver/stop`).
  The Preview button shows the open page in an iframe loaded directly from the
  server's own port, so site scripts cannot reach `/api`. The server is stopped
  on project close, project switch and shutdown, which also closes the project root.
- Build button runs `hugo` in the project root (one build at a time, cancellable)
  and streams its output live to a log panel over SSE
  (`POST /api/build`, `POST /api/build/cancel`, `GET /api/build/stream`).
- Drop an image onto the editor to upload it to `<assetDir>/images/` and insert
  `![](/images/name.png)`. `POST /api/asset` accepts PNG, JPEG, GIF and WebP up to
  10 MB, checks the bytes, never overwrites (a clash adds `-1`, `-2`, ...; a name
  with no usable characters gets a generated one) and stays inside the project.
- Autosave after a 1.5 s pause and on page switch, with saves serialised,
  a Saved/Unsaved/Saving/Conflict indicator, a beforeunload warning and
  editor undo/redo. A hash conflict stops autosave.
- Frontmatter form (title, date, draft, tags) above the editor. `PUT /api/page`
  accepts optional `fields`; unknown keys are kept, comments and TOML key order
  are not.

- CodeMirror 6 markdown editor with save (button and Ctrl/Cmd+S). Pages carry a
  sha256 hash; `PUT /api/page` keeps frontmatter verbatim, writes atomically and
  returns 409 when the file changed on disk.

- Rename/move and delete a page: `POST /api/page/rename` and `DELETE /api/page`
  (confined to the content dir, never overwrite, editable files only) with
  sidebar rename and delete-confirm dialogs.

- Atomic, root-confined file writes (temp file, fsync, rename); new pages
  are published by hard link so an existing file is never replaced.
- `/api` requests with a non-loopback `Host` (DNS rebinding) or a mutating
  request with a foreign `Origin` are refused with 403; the Vite dev origin
  works because `Origin` must match `Host`.
- Project detection for Hugo, covering both the `hugo.*` and legacy
  `config.*` file names and the `config/_default/` layout. Astro projects
  are recognised and rejected with a clear message rather than being
  reported as unrecognised.
- Project lifecycle API: open, get and close. The backend holds one open
  project at a time and issues an id per open, which clients must send with
  every request that touches project files.
- All filesystem access is confined to the open project's directory via
  `os.Root`, rejecting parent traversal, absolute paths and symlinks that
  point outside it.
- Content tree endpoint returning the editable files of the open project,
  with directories sorted before files and empty directories pruned.
- React 19 frontend (Vite, Tailwind v4, shadcn/ui) with a typed API client,
  Zustand stores for the open project and page, a folder picker, a file
  tree and a resizable sidebar. A page reload reconnects to the project the
  backend already has open. Migrated from an initial Vue 3 scaffold.
- Content tree entries carry their frontmatter (title, date, draft, tags),
  and the file tree shows the title, filename, date and a draft badge.
- Sort the file tree by name, title or date, filter it by draft status and
  tag, and search page titles, tags and bodies from a box in the sidebar
  (Ctrl/Cmd+K focuses it). Search is served by `GET /api/search`.
- Create a page from the sidebar's plus button: choose a name, a folder and a
  template (Hugo archetype, or a built-in default). The page is written
  without ever overwriting an existing file, and opens straight away.
  Served by `GET /api/archetypes` and `POST /api/page`.
- Hugo's `contentDir` and `staticDir` are read from the config file instead
  of being hardcoded. Missing, invalid or out-of-project values fall back to
  the defaults.
- The built frontend is embedded into the binary, so a release is a single
  file.
- Project skeleton: Makefile with a single `make check`, `golangci-lint`,
  GitHub Actions CI mirroring it, and structured logging throughout.
