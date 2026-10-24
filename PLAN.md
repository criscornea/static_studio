# Plan: Static Studio
autopilot: on

## Phase 1: Origin and Host check on mutating endpoints
Branch: feat/p1-origin-check
Goal: Mutating `/api` requests from foreign origins or rebound hostnames are refused before more write features land.
Done when: a middleware on POST/PUT/DELETE `/api` routes rejects requests whose `Origin` is not the app's own (`http://127.0.0.1:<port>` or `http://localhost:<port>`), allowing the Vite dev proxy origin `http://localhost:5173` in dev; `/api` requests whose `Host` is not 127.0.0.1/localhost:<port> are rejected to block DNS rebinding; tests cover allowed, missing and foreign `Origin` and foreign `Host`; ROADMAP item ticked and the known shortcut removed.
Risk: high (security)
Status: merged
- [x] M1.1 Add a `Host` check middleware against 127.0.0.1/localhost:<port> with tests
- [x] M1.2 Add an `Origin` check middleware for mutating methods with tests, allowing localhost:5173 in dev
- [x] M1.3 Apply both middlewares to `/api` routes in `server.go`
- [x] M1.4 Update ROADMAP.md, CHANGELOG.md and the known-shortcuts note

## Phase 2: Atomic writes
Branch: feat/p2-atomic-writes
Goal: Every write to a project file goes through a confined temp-file-plus-rename helper, so a crash never leaves a half-written page.
Done when: `internal/content` has a tested atomic write helper on `os.Root` (temp file in the same dir, fsync, rename, cleanup on failure, refuses escapes); page creation uses it while still never overwriting an existing file; `make check` green; ROADMAP item ticked and the matching "Known shortcut" removed.
Risk: high (data writes; a bug can corrupt or lose user pages)
Status: todo
- [ ] M2.1 Add `writeFileAtomic(root, name, data, perm)` in `internal/content` with tests for success, temp cleanup on failure and traversal refusal
- [ ] M2.2 Add a no-overwrite variant (link or exclusive-create semantics) with a test proving an existing file is never replaced
- [ ] M2.3 Switch archetype page creation to the no-overwrite atomic helper
- [ ] M2.4 Update ROADMAP.md, CHANGELOG.md and the known-shortcuts note

## Phase 3: Rename and delete a page
Branch: feat/p3-rename-delete
Goal: Users can rename (or move) and delete a page from the sidebar.
Done when: `POST /api/page/rename` and `DELETE /api/page` exist, require the project id, stay inside the content dir, refuse overwriting on rename and refuse non-editable files; the sidebar offers rename and delete with a confirm dialog; the tree and open page update; Go and vitest tests cover success and refusal paths.
Risk: high (deletion of data, moving user files)
Status: todo
- [ ] M3.1 Add `content.RenamePage` with tests (no overwrite, confined, editable extensions only)
- [ ] M3.2 Add `content.DeletePage` with tests (files only, confined, editable extensions only)
- [ ] M3.3 Add rename and delete API handlers with handler tests
- [ ] M3.4 Add client functions and types for rename and delete
- [ ] M3.5 Add a rename dialog to the sidebar tree with a test
- [ ] M3.6 Add a delete confirm dialog to the sidebar tree with a test

## Phase 4: Markdown editor with save
Branch: feat/p4-editor
Goal: A page opens in a CodeMirror 6 raw markdown editor and can be saved back to disk.
Done when: page load returns a sha256 hash of the file bytes; `PUT /api/page` takes that hash back, saves the body via the atomic helper keeping the existing frontmatter verbatim, and returns a conflict when the on-disk hash differs; `PageView` shows a CodeMirror 6 editor with markdown highlighting and a save action (button and Ctrl/Cmd+S); tests cover save, hash conflict and refusal.
Risk: high (overwrites user files; lost-update concurrency between app and external editors)
Status: todo
- [ ] M4.1 Return a sha256 content hash of the file bytes from page load
- [ ] M4.2 Add `content.SavePage` that writes body plus original frontmatter atomically after a hash check
- [ ] M4.3 Add the `PUT /api/page` handler with tests for success, hash conflict and non-editable paths
- [ ] M4.4 Add a CodeMirror 6 editor component to `PageView`
- [ ] M4.5 Add save to the page store with a Ctrl/Cmd+S binding and a conflict message

## Phase 5: Frontmatter as a form
Branch: feat/p5-frontmatter-form
Goal: Users edit title, date, draft and tags in a form instead of raw frontmatter text.
Done when: the frontmatter package can re-serialise YAML and TOML with changed known fields while preserving unknown keys; save accepts form fields under the same hash check; `PageView` shows a form above the editor; round-trip tests for both formats pass.
Risk: low
Status: todo
- [ ] M5.1 Add frontmatter field updates that preserve unknown keys, with YAML and TOML round-trip tests
- [ ] M5.2 Extend `SavePage` and `PUT /api/page` to accept frontmatter fields
- [ ] M5.3 Add a frontmatter form component (title, date, draft, tags) with a test
- [ ] M5.4 Wire the form into `PageView` and the page store save

## Phase 6: Autosave and unsaved state
Branch: feat/p6-autosave
Goal: Edits save automatically after a pause, with a visible unsaved-changes state and editor undo.
Done when: the page store tracks a dirty flag and the last saved content hash; autosave fires after a debounce and on page switch, sending the hash and adopting the new hash from each response; a hash conflict stops autosave and shows the conflict; leaving with unsaved changes warns (beforeunload); CodeMirror history gives undo/redo; vitest covers debounce, dirty transitions and conflict.
Risk: high (concurrency: overlapping saves and conflicts can lose edits)
Status: todo
- [ ] M6.1 Add a dirty flag and saved-hash tracking to the page store with tests
- [ ] M6.2 Add debounced autosave with serialised in-flight saves and tests
- [ ] M6.3 Stop autosave on a hash conflict and show it, with a test
- [ ] M6.4 Show the unsaved/saving/saved state in `PageView`
- [ ] M6.5 Warn on page switch or unload while saves are pending
- [ ] M6.6 Enable CodeMirror history keybindings for undo and redo

## Phase 7: Image drag and drop
Branch: feat/p7-image-drop
Goal: Dropping an image into the editor stores it in the asset dir and inserts a markdown image link.
Done when: `POST /api/asset` accepts multipart uploads (size limit, image types only, no overwrite, confined to the asset dir) via the atomic helper; dropping onto the editor uploads and inserts `![](/path)` at the cursor; tests cover upload limits and refusal.
Risk: high (security: file upload into the project)
Status: todo
- [ ] M7.1 Add `content.SaveAsset` with tests for type, size, name collision and confinement
- [ ] M7.2 Add the `POST /api/asset` handler with tests
- [ ] M7.3 Add editor drop handling that uploads and inserts the image link

## Phase 8: Build with streamed output
Branch: feat/p8-build
Goal: Users run `hugo` from the app and watch its output live.
Done when: an `internal/build` runner starts `hugo` in the project root with context cancellation and allows one build at a time; `POST /api/build` starts it and `GET /api/build/stream` streams output via SSE; the UI has a build button and a log panel; tests use a fake command.
Risk: high (concurrency and subprocess control)
Status: todo
- [ ] M8.1 Add `internal/build` runner with cancellation and a single-build lock, tested with a fake binary
- [ ] M8.2 Add an output broadcaster that fans lines out to subscribers, with tests
- [ ] M8.3 Add build start and SSE stream handlers with tests
- [ ] M8.4 Add a build store and log panel in the UI

## Phase 9: Dev server and Hugo preview
Branch: feat/p9-dev-server
Goal: Users start and stop `hugo server` and see the real Hugo-rendered page (theme and templates) in the app.
Done when: start/stop/status endpoints manage one `hugo server` bound to 127.0.0.1 on a free port; status returns its URL; the preview button opens an iframe loading directly from that port (separate origin, no proxy on the app origin, so site JS cannot reach `/api`) at the open page's URL; on project close and server shutdown the process is stopped and `Manager.Close` is called; tests cover lifecycle, 127.0.0.1 binding and no orphaned process.
Risk: high (concurrency, subprocess lifecycle, origin isolation)
Status: todo
- [ ] M9.1 Add dev-server process management with 127.0.0.1 binding, free-port selection and tests
- [ ] M9.2 Add start, stop and status handlers with tests
- [ ] M9.3 Map a content path to its page URL on the hugo server, with tests
- [ ] M9.4 Stop subprocesses and close the project on shutdown via signal handling in `cmd/studio`
- [ ] M9.5 Add dev-server controls to the UI
- [ ] M9.6 Add the preview button with an iframe on the hugo server origin

## Phase 10: Build errors at the right file
Branch: feat/p10-build-errors
Goal: Hugo build errors are parsed and shown next to the page they belong to.
Done when: a parser extracts file, line and message from Hugo error output; build status exposes structured errors; the tree marks affected pages and the editor jumps to the line; parser tests use real Hugo error samples.
Risk: low
Status: todo
- [ ] M10.1 Add a Hugo error-line parser with sample-based tests
- [ ] M10.2 Expose parsed errors in the build status API
- [ ] M10.3 Mark pages with errors in the tree and jump to the line in the editor

## Phase 11: Publish to GitHub Pages
Branch: feat/p11-publish
Goal: Users publish the site to GitHub Pages without a terminal.
Done when: publish runs a hugo build, then pushes `public/` to the `gh-pages` branch of a configured remote using the system `git` binary; no credentials are stored by the app (the user's SSH key or git credential helper is used); the remote and branch are configurable in the UI; progress streams to the UI and errors are readable; tests push to a local bare repo as the fake remote.
Risk: high (external side effects, subprocess control, pushing to a user remote)
Status: todo
- [ ] M11.1 Add publish settings (remote, branch) load and save with tests
- [ ] M11.2 Add a git publisher that commits `public/` to `gh-pages` via system `git`, tested against a local bare repo
- [ ] M11.3 Add publish (build then push) and progress-stream handlers with tests
- [ ] M11.4 Add the publish settings and progress UI

## Phase 12: Harden and ship
Branch: feat/p12-harden
Goal: v1.0.0 is releasable with readable errors, logging flags, docs and binaries.
Done when: `-log-level` and `-log-format=json` work; API errors map to non-developer messages in the UI; coverage raised for writes, paths and processes; README complete; a tagged release builds binaries for macOS, Linux and Windows in CI.
Risk: low
Status: todo
- [ ] M12.1 Add `-log-level` and `-log-format` flags with tests
- [ ] M12.2 Map API error codes to readable UI messages
- [ ] M12.3 Add tests for uncovered write, path and process failure paths
- [ ] M12.4 Write the README (problem, quickstart, architecture, scope)
- [ ] M12.5 Add a release workflow building cross-platform binaries on tags
