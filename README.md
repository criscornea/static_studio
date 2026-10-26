# ✨ Static Studio

A local editor for static-site projects. Open a Hugo site folder and edit pages
with a markdown editor and a frontmatter form. Preview with the real Hugo
output, build, and publish to GitHub Pages or your own server. No terminal
needed.

## 🤔 Problem

Writing a post in a static site means juggling an editor, a terminal running
`hugo server`, and a git push. Static Studio puts those steps in one window.
It is one Go binary that serves a JSON API and an embedded React app on
`127.0.0.1:7777`. Nothing leaves your machine except what you publish.

## 🚀 Quickstart

Requirements: [Hugo](https://gohugo.io/) and `git` on your `PATH`.

**Install**

| Platform | Command |
| --- | --- |
| 🍺 macOS, Linux (Homebrew) | `brew install criscornea/tap/static-studio` |
| 🪟 Windows (Scoop) | `scoop bucket add criscornea https://github.com/criscornea/scoop-bucket` then `scoop install static-studio` |
| 🐧 Debian, Ubuntu | `sudo dpkg -i static-studio_<version>_linux_<arch>.deb` |
| 🎩 Fedora | `sudo rpm -i static-studio_<version>_linux_<arch>.rpm` |
| 📦 Any | archive from the Releases page |

Archives and Linux packages are on the Releases page. Releases are not signed,
so a browser download on macOS is blocked by Gatekeeper. Homebrew avoids this;
for an archive run `xattr -d com.apple.quarantine static-studio` once.

Or build from source (Go and Node 22):

```sh
make build
./bin/static_studio
```

Open http://127.0.0.1:7777, pick your site folder, and edit.

Flags:

| Flag | Default | Meaning |
| --- | --- | --- |
| `-addr` | `127.0.0.1:7777` | address to listen on |
| `-log-level` | `info` | `debug`, `info`, `warn` or `error` |
| `-log-format` | `text` | `text` or `json` |

🛠️ Development: `make run` starts the server and the Vite dev server together.
`make check` runs everything CI runs. `make sample` creates a test Hugo site.

## 🧰 What it does

- 🗂️ **Content tree** with search, sort and filter; create pages from
  archetypes; rename and delete.
- ✍️ **Markdown editor** (CodeMirror 6) with autosave, a frontmatter form, a
  floating formatting toolbar and live preview. Markup shows only on the line
  you are editing, and you can switch to plain source any time.
- 🖼️ **Images**: drag and drop into the static folder.
- 👀 **Preview** from a `hugo server` on its own port.
- 🔨 **Build** with streamed output and errors shown at the right file.
- 🌍 **Publish** `public/` to a `gh-pages` branch with the system `git`, or to
  your own server with the system `rsync` over SSH (optional delete and dry
  run).
- 🔐 **No stored credentials**: your SSH key, agent or credential helper signs
  in. For rsync, connect once with `ssh` in a terminal so the host key is in
  `known_hosts`.

## 🎯 Scope

**In scope**

- ✅ Hugo sites
- ✅ Two publish targets: GitHub Pages and rsync over SSH
- ✅ Raw markdown editing

**Out of scope for v1**

- ❌ Plugins
- ❌ Theme design
- ❌ General git integration
- ❌ Multi-user and cloud sync
- ❌ Astro editing
- ❌ Auth

📋 See [`ROADMAP.md`](ROADMAP.md) for known shortcuts and
[`CHANGELOG.md`](CHANGELOG.md) for releases.

## 📄 License

[MIT](LICENSE)
