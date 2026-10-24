# Static Studio

A local editor for static-site projects. Open a Hugo site folder and edit pages
with a markdown editor and a frontmatter form. Preview with the real Hugo
output, build, and publish to GitHub Pages. No terminal needed.

## Problem

Writing a post in a static site means juggling an editor, a terminal running
`hugo server`, and a git push. Static Studio puts those steps in one window.
It is one Go binary that serves a JSON API and an embedded React app on
`127.0.0.1:7777`. Nothing leaves your machine except the publish push.

## Quickstart

Requirements: [Hugo](https://gohugo.io/) and `git` on your `PATH`.

Download a binary from the Releases page, or build from source (Go and Node 22):

```sh
make build
./bin/studio
```

Open http://127.0.0.1:7777, pick your site folder, and edit.

Flags:

| Flag | Default | Meaning |
| --- | --- | --- |
| `-addr` | `127.0.0.1:7777` | address to listen on |
| `-log-level` | `info` | `debug`, `info`, `warn` or `error` |
| `-log-format` | `text` | `text` or `json` |

Development: `make run` starts the server and the Vite dev server together.
`make check` runs everything CI runs. `make sample` creates a test Hugo site.

## What it does

- Content tree with search, sort and filter; create pages from archetypes;
  rename and delete.
- Markdown editor (CodeMirror 6) with autosave and a frontmatter form.
- Drag and drop images into the static folder.
- Preview from a `hugo server` on its own port; build with streamed output and
  errors shown at the right file.
- Publish `public/` to a `gh-pages` branch with the system `git`. No
  credentials are stored; your SSH key or credential helper signs in.

## Architecture

Backend in `internal/`:

- `ssg` detects the generator (Hugo supported, Astro rejected).
- `project` owns the one open project. All file access goes through an
  `os.Root`, so traversal and symlink escapes are refused.
- `content`, `frontmatter` read, search and write pages (atomic writes).
- `build`, `devserver`, `publish` run subprocesses with cancellation.
- `recent` and `publish` settings live in the user config dir.
- `server` is the `net/http` API with Origin and Host checks.

Frontend in `web/`: React 19, TypeScript, Zustand, Vite, Tailwind v4 and
shadcn/ui. `web/embed.go` embeds the built app into the binary.

## Scope

Hugo only. One publish target (GitHub Pages). The editor stays raw markdown.
Out of scope for v1: plugins, theme design, general git integration,
multi-user, cloud sync, Astro editing, auth. See `ROADMAP.md` for known
shortcuts and `CHANGELOG.md` for releases.
