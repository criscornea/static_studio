# Plan: Static Studio
autopilot: off

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
Status: merged
- [x] M2.1 Add `writeFileAtomic(root, name, data, perm)` in `internal/content` with tests for success, temp cleanup on failure and traversal refusal
- [x] M2.2 Add a no-overwrite variant (link or exclusive-create semantics) with a test proving an existing file is never replaced
- [x] M2.3 Switch archetype page creation to the no-overwrite atomic helper
- [x] M2.4 Update ROADMAP.md, CHANGELOG.md and the known-shortcuts note

## Phase 3: Rename and delete a page
Branch: feat/p3-rename-delete
Goal: Users can rename (or move) and delete a page from the sidebar.
Done when: `POST /api/page/rename` and `DELETE /api/page` exist, require the project id, stay inside the content dir, refuse overwriting on rename and refuse non-editable files; the sidebar offers rename and delete with a confirm dialog; the tree and open page update; Go and vitest tests cover success and refusal paths.
Risk: high (deletion of data, moving user files)
Status: merged
- [x] M3.1 Add `content.RenamePage` with tests (no overwrite, confined, editable extensions only)
- [x] M3.2 Add `content.DeletePage` with tests (files only, confined, editable extensions only)
- [x] M3.3 Add rename and delete API handlers with handler tests
- [x] M3.4 Add client functions and types for rename and delete
- [x] M3.5 Add a rename dialog to the sidebar tree with a test
- [x] M3.6 Add a delete confirm dialog to the sidebar tree with a test

## Phase 4: Markdown editor with save
Branch: feat/p4-editor
Goal: A page opens in a CodeMirror 6 raw markdown editor and can be saved back to disk.
Done when: page load returns a sha256 hash of the file bytes; `PUT /api/page` takes that hash back, saves the body via the atomic helper keeping the existing frontmatter verbatim, and returns a conflict when the on-disk hash differs; `PageView` shows a CodeMirror 6 editor with markdown highlighting and a save action (button and Ctrl/Cmd+S); tests cover save, hash conflict and refusal.
Risk: high (overwrites user files; lost-update concurrency between app and external editors)
Status: merged
- [x] M4.1 Return a sha256 content hash of the file bytes from page load
- [x] M4.2 Add `content.SavePage` that writes body plus original frontmatter atomically after a hash check
- [x] M4.3 Add the `PUT /api/page` handler with tests for success, hash conflict and non-editable paths
- [x] M4.4 Add a CodeMirror 6 editor component to `PageView`
- [x] M4.5 Add save to the page store with a Ctrl/Cmd+S binding and a conflict message

## Phase 5: Frontmatter as a form
Branch: feat/p5-frontmatter-form
Goal: Users edit title, date, draft and tags in a form instead of raw frontmatter text.
Done when: the frontmatter package can re-serialise YAML and TOML with changed known fields while preserving unknown keys; save accepts form fields under the same hash check; `PageView` shows a form above the editor; round-trip tests for both formats pass.
Risk: low
Status: merged
- [x] M5.1 Add frontmatter field updates that preserve unknown keys, with YAML and TOML round-trip tests
- [x] M5.2 Extend `SavePage` and `PUT /api/page` to accept frontmatter fields
- [x] M5.3 Add a frontmatter form component (title, date, draft, tags) with a test
- [x] M5.4 Wire the form into `PageView` and the page store save

## Phase 6: Autosave and unsaved state
Branch: feat/p6-autosave
Goal: Edits save automatically after a pause, with a visible unsaved-changes state and editor undo.
Done when: the page store tracks a dirty flag and the last saved content hash; autosave fires after a debounce and on page switch, sending the hash and adopting the new hash from each response; a hash conflict stops autosave and shows the conflict; leaving with unsaved changes warns (beforeunload); CodeMirror history gives undo/redo; vitest covers debounce, dirty transitions and conflict.
Risk: high (concurrency: overlapping saves and conflicts can lose edits)
Status: merged
- [x] M6.1 Add a dirty flag and saved-hash tracking to the page store with tests
- [x] M6.2 Add debounced autosave with serialised in-flight saves and tests
- [x] M6.3 Stop autosave on a hash conflict and show it, with a test
- [x] M6.4 Show the unsaved/saving/saved state in `PageView`
- [x] M6.5 Warn on page switch or unload while saves are pending
- [x] M6.6 Enable CodeMirror history keybindings for undo and redo

## Phase 7: Image drag and drop
Branch: feat/p7-image-drop
Goal: Dropping an image into the editor stores it in the asset dir and inserts a markdown image link.
Done when: `POST /api/asset` accepts multipart uploads (size limit, image types only, no overwrite, confined to the asset dir) via the atomic helper; dropping onto the editor uploads and inserts `![](/path)` at the cursor; tests cover upload limits and refusal.
Risk: high (security: file upload into the project)
Status: merged
- [x] M7.1 Add `content.SaveAsset` with tests for type, size, name collision and confinement
- [x] M7.2 Add the `POST /api/asset` handler with tests
- [x] M7.3 Add editor drop handling that uploads and inserts the image link

## Phase 8: Build with streamed output
Branch: feat/p8-build
Goal: Users run `hugo` from the app and watch its output live.
Done when: an `internal/build` runner starts `hugo` in the project root with context cancellation and allows one build at a time; `POST /api/build` starts it and `GET /api/build/stream` streams output via SSE; the UI has a build button and a log panel; tests use a fake command.
Risk: high (concurrency and subprocess control)
Status: merged
- [x] M8.1 Add `internal/build` runner with cancellation and a single-build lock, tested with a fake binary
- [x] M8.2 Add an output broadcaster that fans lines out to subscribers, with tests
- [x] M8.3 Add build start and SSE stream handlers with tests
- [x] M8.4 Add a build store and log panel in the UI

## Phase 9: Dev server and Hugo preview
Branch: feat/p9-dev-server
Goal: Users start and stop `hugo server` and see the real Hugo-rendered page (theme and templates) in the app.
Done when: start/stop/status endpoints manage one `hugo server` bound to 127.0.0.1 on a free port; status returns its URL; the preview button opens an iframe loading directly from that port (separate origin, no proxy on the app origin, so site JS cannot reach `/api`) at the open page's URL; on project close and server shutdown the process is stopped and `Manager.Close` is called; tests cover lifecycle, 127.0.0.1 binding and no orphaned process.
Risk: high (concurrency, subprocess lifecycle, origin isolation)
Status: merged
- [x] M9.1 Add dev-server process management with 127.0.0.1 binding, free-port selection and tests
- [x] M9.2 Add start, stop and status handlers with tests
- [x] M9.3 Map a content path to its page URL on the hugo server, with tests
- [x] M9.4 Stop subprocesses and close the project on shutdown via signal handling in `cmd/studio`
- [x] M9.5 Add dev-server controls to the UI
- [x] M9.6 Add the preview button with an iframe on the hugo server origin

## Phase 10: Build errors at the right file
Branch: feat/p10-build-errors
Goal: Hugo build errors are parsed and shown next to the page they belong to.
Done when: a parser extracts file, line and message from Hugo error output; build status exposes structured errors; the tree marks affected pages and the editor jumps to the line; parser tests use real Hugo error samples.
Risk: low
Status: merged
- [x] M10.1 Add a Hugo error-line parser with sample-based tests
- [x] M10.2 Expose parsed errors in the build status API
- [x] M10.3 Mark pages with errors in the tree and jump to the line in the editor

## Phase 11: Recent projects
Branch: feat/p11-recent-projects
Goal: The open-project screen lists recently opened projects so the user opens one with a click instead of retyping the path.
Done when: an `internal/recent` store keeps up to 10 entries (path, name, kind, opened-at) in `<os.UserConfigDir()>/static-studio/recent.json`, never in the project dir, written via temp file plus rename; only successful `POST /api/project/open` calls record an entry, most recent first, deduped by the cleaned absolute path; a missing or corrupt file reads as an empty list (logged), and an unavailable config dir disables recording without breaking open; `GET /api/recent` returns the entries with an `available` flag (false when the directory no longer exists) and `DELETE /api/recent` removes one entry by path (covered by the existing Origin/Host guard); `web/src/api/types.ts` and `client.ts` mirror the shapes; `ProjectPicker` shows the list, clicking an available entry runs the existing open flow, unavailable entries are shown greyed out with a remove action only; Go tests (store with `t.TempDir()` as config dir, handlers) and vitest (picker list, click-to-open, remove, unavailable) pass; `make check` green; ROADMAP item ticked and moved to CHANGELOG.
Risk: low
Status: merged
- [x] M11.1 Add the `internal/recent` store (load, add, remove, dedupe, cap, atomic write, injectable dir) with tests
- [x] M11.2 Record successful project opens in the recent store with a handler test
- [x] M11.3 Add `GET /api/recent` and `DELETE /api/recent` handlers with tests, including the `available` flag
- [x] M11.4 Add recent-project types and client functions in `web/src/api`
- [x] M11.5 Show the recent list in `ProjectPicker` with open and remove actions, with a vitest test

## Phase 12: Publish to GitHub Pages
Branch: feat/p12-publish
Goal: Users publish the site to GitHub Pages without a terminal.
Done when: publish runs a hugo build, then pushes `public/` to the `gh-pages` branch of a configured remote using the system `git` binary; no credentials are stored by the app (the user's SSH key or git credential helper is used); the remote and branch are configurable in the UI; progress streams to the UI and errors are readable; tests push to a local bare repo as the fake remote.
Risk: high (external side effects, subprocess control, pushing to a user remote)
Status: merged
- [x] M12.1 Add publish settings (remote, branch) load and save with tests
- [x] M12.2 Add a git publisher that commits `public/` to `gh-pages` via system `git`, tested against a local bare repo
- [x] M12.3 Add publish (build then push) and progress-stream handlers with tests
- [x] M12.4 Add the publish settings and progress UI

## Phase 13: Harden and ship
Branch: feat/p13-harden-ship
Goal: v1.0.0 is releasable with readable errors, logging flags, docs and binaries.
Done when: `-log-level` and `-log-format=json` work; API errors map to non-developer messages in the UI; coverage raised for writes, paths and processes; README complete; a tagged release builds binaries for macOS, Linux and Windows in CI.
Risk: low
Status: merged
- [x] M13.1 Add `-log-level` and `-log-format` flags with tests
- [x] M13.2 Map API error codes to readable UI messages
- [x] M13.3 Add tests for uncovered write, path and process failure paths
- [x] M13.4 Write the README (problem, quickstart, architecture, scope)
- [x] M13.5 Add a release workflow building cross-platform binaries on tags

## Phase 14: Readable Hugo config errors
Branch: feat/p14-config-errors
Goal: An unreadable, oversized or invalid Hugo config stops the project from opening with a readable message instead of silently falling back to default directories.
Done when: `SSG.Dirs` returns an error; a missing optional config is still not an error, but a config that exists and cannot be read, exceeds `maxConfigSize` or fails to parse yields `ssg.ErrInvalidConfig` wrapping the file name and parser message (line number when the parser gives one); `POST /api/project/open` maps it to a 4xx JSON error with a dedicated code; the open screen shows the file and reason in plain language; Go tests cover TOML, YAML and JSON syntax errors, oversize and unreadable files; vitest covers the message mapping; `make check` green.
Risk: low
Status: merged
- [x] M14.1 Return an error from `readConfig` and `SSG.Dirs`, distinguishing missing from invalid, with tests
- [x] M14.2 Add `ssg.ErrInvalidConfig` with file and parser detail, propagated through `Detect`
- [x] M14.3 Map `ErrInvalidConfig` to a JSON error code in the open handler with a server test
- [x] M14.4 Show the config error on the open screen via the readable-message map
- [x] M14.5 Update ROADMAP.md, CHANGELOG.md and the known-shortcuts note

## Phase 15: App shell and theme
Branch: feat/p15-app-shell
Goal: The open project shows as a fixed-viewport desktop editor shell with dark and light themes that follow the OS.
Done when: the document never scrolls (`html`, `body` and `#root` are fixed to the viewport; only panes scroll internally); the shell is a grid of a title bar, an activity bar, a resizable sidebar, the main area and a status bar; a theme store applies the `.dark` class from `prefers-color-scheme`, reacts to OS changes and lets the user override with light, dark or system (persisted in `localStorage`); design tokens give compact density (13px UI text, 28px rows and controls, small radius, Geist UI font, a monospace font for code and paths) and tuned light and dark palettes; the sidebar width and collapsed state persist across reloads and Ctrl/Cmd+B toggles it; the status bar shows the project name, save state, preview-server state and build state; the project picker is a centered welcome screen with recent projects; vitest covers the theme store and the sidebar toggle; Playwright check in both themes once at the end; `make check` green.
Risk: low
Status: merged
- [x] M15.1 Add a theme store following `prefers-color-scheme` with a persisted override and tests
- [x] M15.2 Retune the design tokens in `index.css` for compact density and both palettes
- [x] M15.3 Replace the page layout in `App.tsx` with a fixed-viewport shell grid
- [x] M15.4 Persist the sidebar size and collapsed state with a Ctrl/Cmd+B toggle
- [x] M15.5 Add a status bar for project, save, preview-server and build state
- [x] M15.6 Restyle `ProjectPicker` as a welcome screen

## Phase 16: Explorer sidebar
Branch: feat/p16-explorer
Goal: The sidebar looks and behaves like an editor file explorer with icons, keyboard navigation and context menus.
Done when: the sidebar has a compact section header with icon buttons (new page, sort and filter menu, collapse all) carrying tooltips; tree rows are single-line, 22-24px high, with folder and file icons from `lucide-react`, indent guides, chevrons, a draft marker and an error marker, and a clear active-row highlight; arrow keys, Home, End and Enter move through and open tree rows (roving focus, `role="tree"` and `treeitem`); right-clicking a row opens a context menu (open, new page here, rename, delete, copy path) that reuses the existing dialogs, and the browser's own menu is suppressed only inside the tree; search sits in the header as a compact field and results use the same row style; vitest covers keyboard navigation and context-menu actions; `make check` green.
Risk: low
Status: merged
- [x] M16.1 Add the shadcn `tooltip`, `dropdown-menu` and `context-menu` components
- [x] M16.2 Replace the sidebar controls with an icon toolbar header
- [x] M16.3 Restyle `ContentTree` rows with icons, indent guides and state markers
- [x] M16.4 Add keyboard navigation to `ContentTree` with tests
- [x] M16.5 Add a tree row context menu reusing the page dialogs, with tests
- [x] M16.6 Restyle `SearchResults` to match the tree rows

## Phase 17: Editor chrome and split preview
Branch: feat/p17-editor-chrome
Goal: The main area is a full-height editor with a breadcrumb toolbar, a collapsible properties panel and a resizable side-by-side preview.
Done when: the main area has an editor header with a breadcrumb of the page path, the save state and icon actions (preview, build, publish, preview server) with tooltips and shortcut hints, and the old header buttons are gone; CodeMirror fills the remaining height, scrolls internally and uses a theme that follows the app theme (gutter, cursor, selection, markdown highlight colors, monospace font); frontmatter shows as a compact, collapsible properties panel above the editor whose open state persists; the preview opens in a resizable split pane to the right of the editor with a persisted size; the build log is a bottom panel inside the main area, resizable and toggled from the status bar and Ctrl/Cmd+J; existing autosave, conflict and error behaviour is unchanged and its tests still pass; vitest covers the header actions and panel toggles; `make check` green.
Risk: low
Status: merged
- [x] M17.1 Add an editor header with a path breadcrumb and save state
- [x] M17.2 Move build, publish and preview-server actions into header icon buttons
- [x] M17.3 Make the CodeMirror editor fill its pane with a theme following the app theme
- [x] M17.4 Turn `FrontmatterForm` into a collapsible properties panel
- [x] M17.5 Show the preview in a resizable split pane beside the editor
- [x] M17.6 Dock the build log as a resizable bottom panel toggled by Ctrl/Cmd+J

## Phase 18: Command palette, shortcuts and polish
Branch: feat/p18-commands
Goal: Every action is reachable from the keyboard through a command palette and shortcuts, and every empty or loading state looks finished.
Done when: a command registry lists each action (open page, new page, rename, delete, save, toggle sidebar, toggle preview, toggle build log, build, publish, start or stop the preview server, switch theme, close project) with its shortcut; one global key handler dispatches shortcuts and does not steal keys the CodeMirror editor or open dialogs handle; Ctrl/Cmd+Shift+P opens the command palette and Ctrl/Cmd+P opens quick-open over page titles and paths using the existing search; a shortcuts dialog lists all bindings; empty states (no page selected, empty content dir, no search hits, preview server stopped, empty build log) show an icon, one line of text and the relevant action; loading states use skeletons; app chrome is not text-selectable, uses the default cursor and shows focus rings only for keyboard focus; vitest covers the registry, shortcut dispatch and palette filtering; Playwright check once at the end; ROADMAP.md and CHANGELOG.md updated; `make check` green.
Risk: low
Status: merged
- [x] M18.1 Add a command registry with a global shortcut handler and tests
- [x] M18.2 Add the shadcn `command` component as the command palette on Ctrl/Cmd+Shift+P
- [x] M18.3 Add quick-open for pages on Ctrl/Cmd+P
- [x] M18.4 Add a keyboard shortcuts dialog
- [x] M18.5 Add polished empty and loading states across sidebar, editor, preview and build log
- [x] M18.6 Remove web-page behaviour from app chrome (selection, cursors, focus rings)
- [x] M18.7 Update ROADMAP.md and CHANGELOG.md

## Phase 19: Merged config directory and environment overrides
Branch: feat/p19-config-merge
Goal: `contentDir` and `staticDir` are resolved the way Hugo resolves them: root config, `config/_default/*`, `config/<environment>/*` and `HUGO_*` environment variables merged in Hugo's order.
Done when: a `config` loader in `internal/ssg` merges the root config file with every `config/_default/*.{toml,yaml,yml,json}` (files named after a top-level key such as `languages.toml` or `module.toml` nest under that key) following Hugo's documented precedence; `config/<env>/` is overlaid where env is `HUGO_ENVIRONMENT`, else `development` (the dev server's default); `HUGO_CONTENTDIR` and `HUGO_STATICDIR` override the merged values and still pass `configDir` confinement; the environment is injected so tests do not touch the process env; invalid files in the config dir surface `ErrInvalidConfig` from Phase 14; table tests cover each precedence step; `make check` green.
Risk: low
Status: todo
- [ ] M19.1 Add a case-insensitive deep merge of config maps with tests
- [ ] M19.2 Load and merge `config/_default/*` on top of the root config file
- [ ] M19.3 Overlay `config/<environment>/*` selected by `HUGO_ENVIRONMENT`
- [ ] M19.4 Apply `HUGO_CONTENTDIR` and `HUGO_STATICDIR` overrides from an injected environment
- [ ] M19.5 Update ROADMAP.md, CHANGELOG.md and the known-shortcuts note

## Phase 20: Resolve every content directory
Branch: feat/p20-content-dirs
Goal: Per-language `contentDir` values and project-local module mounts are discovered, and the tree and search show pages from all of them.
Done when: `ssg.Project` gains `ContentDirs []ContentDir{Path, Lang}` (primary first, deduplicated; `ContentDir` kept as the primary for compatibility) and `types.ts` mirrors it; `languages.<lang>.contentDir` entries are added; `module.mounts` with target `content` or `content/...` are added, and per Hugo, any content mount replaces the implicit `contentDir` mount; the first `static` mount source sets `AssetDir`; mount sources that are absolute, escape the root or belong to imported modules are skipped and logged, never followed; `GET /api/content` returns one root per content dir and the sidebar renders them with a language label; search walks every content dir; Go tests cover language dirs, mounts, the implicit-mount rule and refused sources; vitest covers the multi-root tree; `make check` green.
Risk: high (security: config-controlled paths feed path confinement; public API shape change)
Status: todo
- [ ] M20.1 Collect per-language `contentDir` values into a deduplicated `ContentDirs` list
- [ ] M20.2 Add content and static module mounts, skipping sources outside the project
- [ ] M20.3 Expose `contentDirs` in the project JSON and `types.ts`
- [ ] M20.4 Return one tree root per content dir from `GET /api/content`
- [ ] M20.5 Render multiple labelled roots in `ContentTree`
- [ ] M20.6 Search across every content dir

## Phase 21: Edit pages in every content directory
Branch: feat/p21-multi-dir-edit
Goal: Pages in any resolved content dir can be opened, saved, created, renamed and deleted with the same confinement guarantees as the primary dir.
Done when: a single `content.Dirs` check replaces the `contentDir` prefix checks in read, save, rename, delete and create, accepting a path only when it lies inside one resolved content dir (rename may move between dirs only if both are resolved); `POST /api/page` accepts a target content dir validated against the resolved set; `NewPageDialog` offers a content-dir picker when there is more than one; `RenamePageDialog` accepts a path in any content dir; Go tests prove paths outside every resolved dir, including sibling-prefix names like `content-old/`, are refused for each operation; vitest covers both dialogs; `make check` green; ROADMAP, CHANGELOG and known shortcuts updated (per-language preview URLs stay a separate shortcut).
Risk: high (security: write-path confinement; data writes)
Status: todo
- [ ] M21.1 Add a `content.Dirs` membership check with sibling-prefix tests
- [ ] M21.2 Use the dir-set check in read, save and delete
- [ ] M21.3 Use the dir-set check in rename, allowing moves between resolved dirs
- [ ] M21.4 Accept a target content dir in page creation and the new-page dialog
- [ ] M21.5 Accept any content dir in the rename dialog
- [ ] M21.6 Update ROADMAP.md, CHANGELOG.md and the known-shortcuts note

## Phase 22: Close the save, rename and delete race window
Branch: feat/p22-toctou
Goal: A file changed by an external editor between the check and the write is never overwritten or deleted unseen.
Done when: save and delete use grab-then-verify: the file is first moved atomically to a hidden sibling, its hash is checked on the moved copy, and on mismatch it is restored with a no-overwrite link and `ErrConflict` is returned; the new content is put in place with a no-overwrite link, so a file recreated at the name in between is kept and reported as a conflict; `DELETE /api/page` requires the hash the client last read (the client reads the page first when deleting from the sidebar); rename verifies after linking that the moved file is the one checked (`os.SameFile`) and undoes on mismatch; on filesystems without hard links the old behaviour remains and is documented; a test hook between check and write simulates an external writer for each operation under `-race`; no hidden temp or backup file is left after success or failure; `make check` green.
Risk: high (data writes, deletion of data, public API change, concurrency)
Status: todo
- [ ] M22.1 Add grab and restore helpers in `atomic.go` with no-overwrite tests
- [ ] M22.2 Verify the hash on the grabbed copy in `SavePage`
- [ ] M22.3 Require the page hash for `DELETE /api/page` in handler, client and sidebar
- [ ] M22.4 Delete only the grabbed copy after its hash is verified
- [ ] M22.5 Verify the moved file's identity in `RenamePage`
- [ ] M22.6 Update ROADMAP.md, CHANGELOG.md and the known-shortcuts note

## Phase 23: Cached page index for tree and search
Branch: feat/p23-page-index
Goal: Loading the tree and searching do not re-read unchanged pages, so a site with thousands of pages stays fast.
Done when: a `content.Index` owned by each open project caches per-page frontmatter fields and search text, keyed by path and validated by size, modification time and file identity on each lookup, so an external edit is picked up on the next load without a file watcher or new dependency; save, create, rename and delete update or drop their entries; the index is dropped on project close; it is safe under `-race` with concurrent tree, search and save calls; benchmarks on a generated 5,000-page site show warm tree and search at least 5x faster than the baseline; search semantics (case-insensitive substring, title and tag hits first, 50-hit cap) are unchanged; `make check` green.
Risk: high (concurrency; a stale cache shows wrong content)
Status: todo
- [ ] M23.1 Add tree and search benchmarks over a generated 5,000-page site
- [ ] M23.2 Add `content.Index` with stat-validated entries and staleness tests
- [ ] M23.3 Read tree frontmatter through the index
- [ ] M23.4 Match search queries against indexed text
- [ ] M23.5 Own one index per open project and update it on writes
- [ ] M23.6 Update ROADMAP.md, CHANGELOG.md and the known-shortcuts note
