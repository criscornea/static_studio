# Changelog

All notable changes to this project are documented here.
Format follows [Keep a Changelog](https://keepachangelog.com/en/1.1.0/).

## [Unreleased]

### Fixed

- Generated UI components no longer depend on the third-party `cn` package;
  they use the local helper in `web/src/lib/utils.ts`.

### Added

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
