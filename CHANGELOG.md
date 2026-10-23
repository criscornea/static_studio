# Changelog

All notable changes to this project are documented here.
Format follows [Keep a Changelog](https://keepachangelog.com/en/1.1.0/).

## [Unreleased]

### Added

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
- Hugo's `contentDir` and `staticDir` are read from the config file instead
  of being hardcoded. Missing, invalid or out-of-project values fall back to
  the defaults.
- The built frontend is embedded into the binary, so a release is a single
  file.
- Project skeleton: Makefile with a single `make check`, `golangci-lint`,
  GitHub Actions CI mirroring it, and structured logging throughout.
