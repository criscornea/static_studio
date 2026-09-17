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
- Vue 3 frontend with a typed API client, a Pinia store for the open
  project, a folder picker and a file tree. A page reload reconnects to the
  project the backend already has open.
- The built frontend is embedded into the binary, so a release is a single
  file.
- Project skeleton: Makefile with a single `make check`, `golangci-lint`,
  GitHub Actions CI mirroring it, and structured logging throughout.
