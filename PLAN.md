# Plan: Static Studio - Editor & preview
autopilot: off

Source: ROADMAP.md section "Editor & preview". Render markdown only; template-accurate
preview is v2. Undo across sessions is out of scope. Page creation (archetypes,
NewPageDialog) is built on the `create-page` branch and is not part of this plan.

## Phase 1: Save a page body
Branch: feat/p1-save-body
Goal: edit a page body in CodeMirror 6 and save it to disk safely with Ctrl/Cmd+S.
Done when: `PUT /api/page` writes body changes atomically (temp file + rename inside
`os.Root`), keeps the frontmatter bytes unchanged, rejects a stale version with 409
`conflict`; the UI shows the body in CodeMirror and Ctrl/Cmd+S saves it; Go and
vitest tests cover the write, the conflict and the path checks; `make check` is green.
Risk: high (writes user files; overwrite of external changes)
Status: todo
- [ ] M1.1 `content.WriteFileAtomic`: temp file in the target dir + rename via `os.Root`, keeps file mode, cleans up on error
- [ ] M1.2 Page `version` (content hash) returned by `GET /api/page`
- [ ] M1.3 `content.WritePage`: reassemble original fences + meta + new body, refuse on version mismatch
- [ ] M1.4 `PUT /api/page` handler with path checks, size cap, 409 on conflict; client and types updated
- [ ] M1.5 `MarkdownEditor` component on CodeMirror 6 (`@codemirror/lang-markdown`) replacing the `<pre>` body
- [ ] M1.6 Page store tracks the edited body, dirty flag and Ctrl/Cmd+S save

## Phase 2: Frontmatter as a form
Branch: feat/p2-frontmatter-form
Goal: edit title, date, draft and tags in a form and save them into the page's own frontmatter format.
Done when: saving form changes updates only those keys; unknown keys survive in YAML and
TOML; YAML comments and key order survive; a page without frontmatter gets a new YAML
block; the tree and search reflect the saved values; round-trip tests cover each case.
Risk: high (rewrites user frontmatter; data loss if keys are dropped)
Status: todo
- [ ] M2.1 `frontmatter.UpdateYAML`: set known fields through the goccy/go-yaml AST, keep comments, order and unknown keys
- [ ] M2.2 `frontmatter.UpdateTOML`: set known fields, keep unknown keys (document comment loss as a known shortcut if unavoidable)
- [ ] M2.3 Create a YAML frontmatter block for pages that have none
- [ ] M2.4 `PUT /api/page` accepts optional `fields` and applies them through the updaters
- [ ] M2.5 `FrontmatterForm` component (title, date, draft toggle, tag input) replacing the read-only header
- [ ] M2.6 Refresh the content tree entry after a save

## Phase 3: Live markdown preview
Branch: feat/p3-preview
Goal: show a backend-rendered HTML preview next to the editor that follows typing.
Done when: `POST /api/preview` returns goldmark HTML (GFM, raw HTML escaped, like Hugo's
default `unsafe: false`); root-relative image URLs load from the project's static dir;
the preview runs in a sandboxed iframe without scripts; a split view updates within a
debounce; tests cover rendering, escaping and static-file confinement.
Risk: high (renders user HTML in the app; serves project files over HTTP)
Status: todo
- [ ] M3.1 `internal/render` on goldmark (Hugo's renderer, reason in commit body) with GFM and escaped raw HTML
- [ ] M3.2 Rewrite root-relative `img src` to `/api/static/{id}/{path}` during rendering
- [ ] M3.3 `GET /api/static/{id}/{path...}` serving files from `AssetDir` via `os.Root`, with image content types only
- [ ] M3.4 `POST /api/preview` handler with a body size cap
- [ ] M3.5 `Preview` pane in a sandboxed `srcdoc` iframe, debounced, in a resizable split with the editor

## Phase 4: Drag & drop images
Branch: feat/p4-image-drop
Goal: drop or paste an image into the editor, store it under `static/`, and insert its markdown link.
Done when: the image lands in `<AssetDir>/images/` with a sanitized, collision-free name;
only image types under a size cap are accepted; the request requires `X-Project-ID`;
`![alt](/images/<name>)` is inserted at the drop position and shows in the preview.
Risk: high (file upload writes into the project; security)
Status: todo
- [ ] M4.1 `content.SaveAsset`: sanitize the name, add a numeric suffix on collision, atomic write under `AssetDir/images`
- [ ] M4.2 `POST /api/static` multipart handler with type allowlist, size cap and required project id header
- [ ] M4.3 CodeMirror drop handler uploads files and inserts the markdown link at the drop position
- [ ] M4.4 CodeMirror paste handler for clipboard images

## Phase 5: Autosave and unsaved state
Branch: feat/p5-autosave
Goal: changes save themselves, unsaved work is visible and protected, and undo works across autosaves.
Done when: edits save after an idle debounce with at most one save in flight; the header
and tree mark unsaved pages; switching pages or closing the project flushes pending
changes; leaving the tab warns while dirty; a 409 conflict offers reload or overwrite;
CodeMirror undo history survives autosaves within the session; vitest covers the store logic.
Risk: high (concurrency between saves, page switches and external edits)
Status: todo
- [ ] M5.1 Debounced autosave in the page store with a single in-flight save and a queued latest version
- [ ] M5.2 Flush pending changes before switching pages or closing the project
- [ ] M5.3 Unsaved/saving/saved indicator in the page header and the tree
- [ ] M5.4 `beforeunload` warning while dirty or saving
- [ ] M5.5 Conflict dialog on 409: reload from disk or overwrite
- [ ] M5.6 Keep the CodeMirror undo history when a save response updates the version
