package server

import (
	"bytes"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/criscornea/static_studio/internal/content"
	"github.com/criscornea/static_studio/internal/frontmatter"
	"github.com/criscornea/static_studio/internal/ssg"
)

// newHugoDir creates a minimal Hugo project and returns its path.
func newHugoDir(t *testing.T) string {
	t.Helper()

	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "hugo.toml"), nil, 0o600); err != nil {
		t.Fatal(err)
	}
	return dir
}

// api drives one Server instance across several requests, so that state
// established by one call is visible to the next.
type api struct {
	t       *testing.T
	handler http.Handler
}

func newAPI(t *testing.T) *api {
	t.Helper()
	return &api{t: t, handler: New(slog.New(slog.DiscardHandler), nil).Routes()}
}

func (a *api) do(method, target, body string) *httptest.ResponseRecorder {
	a.t.Helper()

	var r io.Reader
	if body != "" {
		r = strings.NewReader(body)
	}
	req := httptest.NewRequest(method, target, r)
	rec := httptest.NewRecorder()
	a.handler.ServeHTTP(rec, req)
	return rec
}

func (a *api) open(path string) *httptest.ResponseRecorder {
	a.t.Helper()
	body, err := json.Marshal(openProjectRequest{Path: path})
	if err != nil {
		a.t.Fatal(err)
	}
	return a.do(http.MethodPost, "/api/project/open", string(body))
}

// decodeProject decodes a successful project response.
func decodeProject(t *testing.T, rec *httptest.ResponseRecorder) projectResponse {
	t.Helper()

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200 (body: %s)", rec.Code, rec.Body)
	}
	var got projectResponse
	if err := json.NewDecoder(rec.Body).Decode(&got); err != nil {
		t.Fatalf("decoding body: %v", err)
	}
	return got
}

// decodeError decodes an error body and checks it is fit for a human.
func decodeError(t *testing.T, rec *httptest.ResponseRecorder) apiError {
	t.Helper()

	if ct := rec.Header().Get("Content-Type"); ct != "application/json" {
		t.Errorf("Content-Type = %q, want application/json", ct)
	}
	var got apiError
	if err := json.NewDecoder(rec.Body).Decode(&got); err != nil {
		t.Fatalf("decoding error body: %v", err)
	}
	if got.Message == "" {
		t.Error("Message is empty; errors must be readable by a non-developer")
	}
	return got
}

func TestProjectLifecycle(t *testing.T) {
	a := newAPI(t)

	// Nothing open yet.
	rec := a.do(http.MethodGet, "/api/project", "")
	if rec.Code != http.StatusConflict {
		t.Fatalf("GET before open: status = %d, want 409", rec.Code)
	}
	if got := decodeError(t, rec); got.Code != "no_project" {
		t.Errorf("code = %q, want no_project", got.Code)
	}

	// Open.
	dir := newHugoDir(t)
	opened := decodeProject(t, a.open(dir))
	if opened.ID == "" {
		t.Error("id is empty")
	}
	if opened.Kind != ssg.KindHugo {
		t.Errorf("kind = %q, want %q", opened.Kind, ssg.KindHugo)
	}
	if !filepath.IsAbs(opened.Root) {
		t.Errorf("root = %q, want an absolute path", opened.Root)
	}

	// The same project is now current.
	current := decodeProject(t, a.do(http.MethodGet, "/api/project", ""))
	if current.ID != opened.ID {
		t.Errorf("current id = %q, want %q", current.ID, opened.ID)
	}

	// Close, twice: closing what is already closed is not an error.
	for i := range 2 {
		rec := a.do(http.MethodPost, "/api/project/close", "")
		if rec.Code != http.StatusNoContent {
			t.Fatalf("close #%d: status = %d, want 204", i+1, rec.Code)
		}
	}

	if rec := a.do(http.MethodGet, "/api/project", ""); rec.Code != http.StatusConflict {
		t.Errorf("GET after close: status = %d, want 409", rec.Code)
	}
}

func TestOpenProjectErrors(t *testing.T) {
	astroDir := t.TempDir()
	if err := os.WriteFile(filepath.Join(astroDir, "astro.config.mjs"), nil, 0o600); err != nil {
		t.Fatal(err)
	}

	tests := []struct {
		name       string
		body       string
		wantStatus int
		wantCode   string
	}{
		{"empty body", "", http.StatusBadRequest, "bad_request"},
		{"malformed json", "{", http.StatusBadRequest, "bad_request"},
		{"unknown field", `{"path":"/tmp","colour":"red"}`, http.StatusBadRequest, "bad_request"},
		{"empty path", `{"path":""}`, http.StatusBadRequest, "missing_path"},
		{"not a project", `{"path":"` + t.TempDir() + `"}`, http.StatusUnprocessableEntity, "not_a_project"},
		{"astro project", `{"path":"` + astroDir + `"}`, http.StatusNotImplemented, "unsupported_generator"},
		{"missing directory", `{"path":"/definitely/not/here"}`, http.StatusBadRequest, "cannot_open"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rec := newAPI(t).do(http.MethodPost, "/api/project/open", tt.body)

			if rec.Code != tt.wantStatus {
				t.Fatalf("status = %d, want %d (body: %s)", rec.Code, tt.wantStatus, rec.Body)
			}
			if got := decodeError(t, rec); got.Code != tt.wantCode {
				t.Errorf("code = %q, want %q", got.Code, tt.wantCode)
			}
		})
	}
}

func TestOpenProjectErrorDoesNotLeakInternals(t *testing.T) {
	missing := filepath.Join(t.TempDir(), "secret-folder-name")

	got := decodeError(t, newAPI(t).open(missing))

	if strings.Contains(got.Message, missing) {
		t.Errorf("message leaks the filesystem path: %q", got.Message)
	}
	if strings.Contains(strings.ToLower(got.Message), "no such file") {
		t.Errorf("message leaks the underlying OS error: %q", got.Message)
	}
}

func TestReopenIssuesANewID(t *testing.T) {
	a := newAPI(t)

	first := decodeProject(t, a.open(newHugoDir(t)))
	second := decodeProject(t, a.open(newHugoDir(t)))

	if first.ID == second.ID {
		t.Error("reopening returned the same id; ids must be unique per open")
	}
}

// withContent creates a Hugo project containing some content files.
func withContent(t *testing.T) string {
	t.Helper()

	dir := newHugoDir(t)
	files := map[string]string{
		"content/about.md":        "about",
		"content/posts/first.md":  "first",
		"content/posts/second.md": "second",
		"content/image.png":       "not markdown",
		"static/logo.png":         "outside content",
	}
	for name, body := range files {
		full := filepath.Join(dir, filepath.FromSlash(name))
		if err := os.MkdirAll(filepath.Dir(full), 0o750); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(full, []byte(body), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	return dir
}

func (a *api) content(id string) *httptest.ResponseRecorder {
	a.t.Helper()

	req := httptest.NewRequest(http.MethodGet, "/api/content", nil)
	if id != "" {
		req.Header.Set("X-Project-ID", id)
	}
	rec := httptest.NewRecorder()
	a.handler.ServeHTTP(rec, req)
	return rec
}

func TestContentTree(t *testing.T) {
	a := newAPI(t)
	opened := decodeProject(t, a.open(withContent(t)))

	rec := a.content(opened.ID)
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200 (body: %s)", rec.Code, rec.Body)
	}

	var got contentResponse
	if err := json.NewDecoder(rec.Body).Decode(&got); err != nil {
		t.Fatalf("decoding body: %v", err)
	}
	if got.Root == nil {
		t.Fatal("root is nil")
	}
	if got.Root.Path != "content" {
		t.Errorf("root path = %q, want content", got.Root.Path)
	}

	// Directories first, then files. Non-markdown and anything outside
	// the content directory must not appear.
	var found []string
	for _, c := range got.Root.Children {
		found = append(found, c.Path)
	}
	want := []string{"content/posts", "content/about.md"}
	if len(found) != len(want) {
		t.Fatalf("children = %v, want %v", found, want)
	}
	for i := range want {
		if found[i] != want[i] {
			t.Errorf("children[%d] = %q, want %q", i, found[i], want[i])
		}
	}
}

func TestContentAcceptsQueryParameterID(t *testing.T) {
	a := newAPI(t)
	opened := decodeProject(t, a.open(withContent(t)))

	target := "/api/content?projectId=" + url.QueryEscape(opened.ID)
	req := httptest.NewRequest(http.MethodGet, target, nil)
	rec := httptest.NewRecorder()
	a.handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200 (body: %s)", rec.Code, rec.Body)
	}
}

func TestContentRejectsBadID(t *testing.T) {
	tests := []struct {
		name     string
		id       func(openedID string) string
		wantCode string
	}{
		{"no id at all", func(string) string { return "" }, "stale_project"},
		{"wrong id", func(string) string { return "not-the-id" }, "stale_project"},
		{"id of a previous project", func(old string) string { return old }, "stale_project"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			a := newAPI(t)
			first := decodeProject(t, a.open(withContent(t)))
			// Reopen so that first.ID becomes stale.
			decodeProject(t, a.open(withContent(t)))

			rec := a.content(tt.id(first.ID))

			if rec.Code != http.StatusConflict {
				t.Fatalf("status = %d, want 409 (body: %s)", rec.Code, rec.Body)
			}
			if got := decodeError(t, rec); got.Code != tt.wantCode {
				t.Errorf("code = %q, want %q", got.Code, tt.wantCode)
			}
		})
	}
}

func TestContentWithNoProjectOpen(t *testing.T) {
	rec := newAPI(t).content("some-id")

	if rec.Code != http.StatusConflict {
		t.Fatalf("status = %d, want 409 (body: %s)", rec.Code, rec.Body)
	}
	if got := decodeError(t, rec); got.Code != "no_project" {
		t.Errorf("code = %q, want no_project", got.Code)
	}
}

func TestContentEmptyProject(t *testing.T) {
	a := newAPI(t)
	opened := decodeProject(t, a.open(newHugoDir(t))) // no content directory

	rec := a.content(opened.ID)
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200 (body: %s)", rec.Code, rec.Body)
	}

	var got contentResponse
	if err := json.NewDecoder(rec.Body).Decode(&got); err != nil {
		t.Fatal(err)
	}
	if got.Root == nil {
		t.Fatal("root is nil; an empty project must still return a root node")
	}
	if len(got.Root.Children) != 0 {
		t.Errorf("children = %d, want 0", len(got.Root.Children))
	}
}

func TestContentTreeJSONFieldNames(t *testing.T) {
	a := newAPI(t)
	opened := decodeProject(t, a.open(withContent(t)))

	body := a.content(opened.ID).Body.String()

	for _, key := range []string{`"name"`, `"path"`, `"isDir"`, `"children"`, `"ext"`, `"size"`} {
		if !strings.Contains(body, key) {
			t.Errorf("response is missing the %s field: %s", key, body)
		}
	}
	if strings.Contains(body, `"Path"`) {
		t.Error(`response contains "Path"; JSON field names must be lower camel case`)
	}
}

// withPages creates a Hugo project with a small set of pages.
func withPages(t *testing.T) string {
	t.Helper()

	dir := newHugoDir(t)
	files := map[string]string{
		"content/post.md":   "---\ntitle: A Post\ndraft: true\n---\nThe body.\n",
		"content/broken.md": "---\ntitle: [unclosed\n---\n",
		"static/x.md":       "---\ntitle: outside\n---\n",
	}
	for name, body := range files {
		full := filepath.Join(dir, filepath.FromSlash(name))
		if err := os.MkdirAll(filepath.Dir(full), 0o750); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(full, []byte(body), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	return dir
}

func (a *api) page(id, path string) *httptest.ResponseRecorder {
	a.t.Helper()

	req := httptest.NewRequest(http.MethodGet, "/api/page?path="+url.QueryEscape(path), nil)
	if id != "" {
		req.Header.Set("X-Project-ID", id)
	}
	rec := httptest.NewRecorder()
	a.handler.ServeHTTP(rec, req)
	return rec
}

func TestPage(t *testing.T) {
	a := newAPI(t)
	opened := decodeProject(t, a.open(withPages(t)))

	rec := a.page(opened.ID, "content/post.md")
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200 (body: %s)", rec.Code, rec.Body)
	}

	var got content.Page
	if err := json.NewDecoder(rec.Body).Decode(&got); err != nil {
		t.Fatalf("decoding body: %v", err)
	}
	if got.Path != "content/post.md" {
		t.Errorf("path = %q, want content/post.md", got.Path)
	}
	if got.Format != frontmatter.FormatYAML {
		t.Errorf("format = %q, want yaml", got.Format)
	}
	if got.Fields == nil || got.Fields.Title != "A Post" || !got.Fields.Draft {
		t.Errorf("fields = %+v, want title \"A Post\" and draft", got.Fields)
	}
	if got.Body != "The body.\n" {
		t.Errorf("body = %q, want %q", got.Body, "The body.\n")
	}
}

func TestPageJSONShape(t *testing.T) {
	a := newAPI(t)
	opened := decodeProject(t, a.open(withPages(t)))

	body := a.page(opened.ID, "content/post.md").Body.String()

	for _, key := range []string{`"path"`, `"format"`, `"fields"`, `"body"`, `"title"`, `"draft"`} {
		if !strings.Contains(body, key) {
			t.Errorf("response is missing %s: %s", key, body)
		}
	}
	// A []byte body would be base64-encoded; the text must arrive as text.
	if !strings.Contains(body, `"body":"The body.\n"`) {
		t.Errorf("body is not plain text: %s", body)
	}
}

func TestPageErrors(t *testing.T) {
	dir := withPages(t)
	huge := filepath.Join(dir, "content", "huge.md")
	// Comfortably above the content package's size limit.
	if err := os.WriteFile(huge, bytes.Repeat([]byte("a"), 3<<20), 0o600); err != nil {
		t.Fatal(err)
	}

	tests := []struct {
		name       string
		path       string
		wantStatus int
		wantCode   string
	}{
		{"missing path", "", http.StatusBadRequest, "missing_path"},
		{"config file", "hugo.toml", http.StatusBadRequest, "not_editable"},
		{"outside content", "static/x.md", http.StatusBadRequest, "not_editable"},
		{"traversal", "content/../hugo.toml", http.StatusBadRequest, "not_editable"},
		{"missing page", "content/nope.md", http.StatusNotFound, "page_not_found"},
		{"too large", "content/huge.md", http.StatusUnprocessableEntity, "page_too_large"},
		{"broken frontmatter", "content/broken.md", http.StatusUnprocessableEntity, "invalid_frontmatter"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			a := newAPI(t)
			opened := decodeProject(t, a.open(dir))

			rec := a.page(opened.ID, tt.path)

			if rec.Code != tt.wantStatus {
				t.Fatalf("status = %d, want %d (body: %s)", rec.Code, tt.wantStatus, rec.Body)
			}
			if got := decodeError(t, rec); got.Code != tt.wantCode {
				t.Errorf("code = %q, want %q", got.Code, tt.wantCode)
			}
		})
	}
}

func TestPageRequiresProjectID(t *testing.T) {
	a := newAPI(t)
	decodeProject(t, a.open(withPages(t)))

	rec := a.page("", "content/post.md")

	if rec.Code != http.StatusConflict {
		t.Fatalf("status = %d, want 409 (body: %s)", rec.Code, rec.Body)
	}
	if got := decodeError(t, rec); got.Code != "stale_project" {
		t.Errorf("code = %q, want stale_project", got.Code)
	}
}

func (a *api) search(id, query string) *httptest.ResponseRecorder {
	a.t.Helper()

	req := httptest.NewRequest(http.MethodGet, "/api/search?q="+url.QueryEscape(query), nil)
	if id != "" {
		req.Header.Set("X-Project-ID", id)
	}
	rec := httptest.NewRecorder()
	a.handler.ServeHTTP(rec, req)
	return rec
}

func TestSearch(t *testing.T) {
	a := newAPI(t)
	opened := decodeProject(t, a.open(withPages(t)))

	rec := a.search(opened.ID, "body")
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200 (body: %s)", rec.Code, rec.Body)
	}

	var got content.Results
	if err := json.NewDecoder(rec.Body).Decode(&got); err != nil {
		t.Fatalf("decoding body: %v", err)
	}
	if len(got.Hits) != 1 || got.Hits[0].Path != "content/post.md" || got.Hits[0].Title != "A Post" {
		t.Fatalf("hits = %+v, want only content/post.md (static/ is outside the content dir)", got.Hits)
	}
	if !got.Hits[0].Draft || got.Hits[0].Snippet != "The body." {
		t.Errorf("hit = %+v, want draft with snippet", got.Hits[0])
	}
}

func TestSearchJSONShape(t *testing.T) {
	a := newAPI(t)
	opened := decodeProject(t, a.open(withPages(t)))

	var raw map[string]any
	if err := json.NewDecoder(a.search(opened.ID, "nothing-matches").Body).Decode(&raw); err != nil {
		t.Fatal(err)
	}
	// The client relies on these exact names, and on [] rather than null.
	hits, ok := raw["hits"].([]any)
	if !ok || len(hits) != 0 {
		t.Errorf("hits = %#v, want []", raw["hits"])
	}
	if raw["truncated"] != false {
		t.Errorf("truncated = %#v, want false", raw["truncated"])
	}
}

func TestSearchErrors(t *testing.T) {
	a := newAPI(t)
	opened := decodeProject(t, a.open(withPages(t)))

	tests := []struct {
		name     string
		id       string
		query    string
		wantCode string
		status   int
	}{
		{"empty query", opened.ID, "", "missing_query", http.StatusBadRequest},
		{"blank query", opened.ID, "   ", "missing_query", http.StatusBadRequest},
		{"huge query", opened.ID, strings.Repeat("a", maxQueryLength+1), "bad_request", http.StatusBadRequest},
		{"no project id", "", "x", "stale_project", http.StatusConflict},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rec := a.search(tt.id, tt.query)
			if rec.Code != tt.status {
				t.Fatalf("status = %d, want %d (body: %s)", rec.Code, tt.status, rec.Body)
			}
			if got := decodeError(t, rec); got.Code != tt.wantCode {
				t.Errorf("code = %q, want %q", got.Code, tt.wantCode)
			}
		})
	}
}

func (a *api) createPage(id, body string) *httptest.ResponseRecorder {
	a.t.Helper()

	req := httptest.NewRequest(http.MethodPost, "/api/page", strings.NewReader(body))
	if id != "" {
		req.Header.Set("X-Project-ID", id)
	}
	rec := httptest.NewRecorder()
	a.handler.ServeHTTP(rec, req)
	return rec
}

func TestArchetypes(t *testing.T) {
	dir := newHugoDir(t)
	if err := os.MkdirAll(filepath.Join(dir, "archetypes"), 0o750); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "archetypes", "posts.md"), nil, 0o600); err != nil {
		t.Fatal(err)
	}
	a := newAPI(t)
	opened := decodeProject(t, a.open(dir))

	req := httptest.NewRequest(http.MethodGet, "/api/archetypes", nil)
	req.Header.Set("X-Project-ID", opened.ID)
	rec := httptest.NewRecorder()
	a.handler.ServeHTTP(rec, req)

	var got struct {
		Archetypes []struct {
			Name    string `json:"name"`
			Builtin bool   `json:"builtin"`
		} `json:"archetypes"`
	}
	if err := json.NewDecoder(rec.Body).Decode(&got); err != nil {
		t.Fatal(err)
	}
	if len(got.Archetypes) != 2 || got.Archetypes[0].Name != "default" || !got.Archetypes[0].Builtin ||
		got.Archetypes[1].Name != "posts" || got.Archetypes[1].Builtin {
		t.Errorf("archetypes = %+v, want builtin default, then posts", got.Archetypes)
	}
}

func TestCreatePage(t *testing.T) {
	dir := newHugoDir(t)
	a := newAPI(t)
	opened := decodeProject(t, a.open(dir))

	rec := a.createPage(opened.ID, `{"folder":"posts","name":"Hello World"}`)
	if rec.Code != http.StatusCreated {
		t.Fatalf("status = %d, want 201 (body: %s)", rec.Code, rec.Body)
	}

	var got content.Page
	if err := json.NewDecoder(rec.Body).Decode(&got); err != nil {
		t.Fatal(err)
	}
	if got.Path != "content/posts/hello-world.md" || got.Fields == nil || got.Fields.Title != "Hello World" {
		t.Errorf("page = %+v", got)
	}
	if _, err := os.Stat(filepath.Join(dir, "content", "posts", "hello-world.md")); err != nil {
		t.Errorf("file not written: %v", err)
	}

	// The new page shows up in the tree and can be opened.
	if rec := a.content(opened.ID); !strings.Contains(rec.Body.String(), "hello-world.md") {
		t.Errorf("tree does not list the new page: %s", rec.Body)
	}
	if rec := a.page(opened.ID, got.Path); rec.Code != http.StatusOK {
		t.Errorf("page status = %d, want 200", rec.Code)
	}
}

func TestCreatePageErrors(t *testing.T) {
	dir := newHugoDir(t)
	if err := os.MkdirAll(filepath.Join(dir, "content"), 0o750); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "content", "taken.md"), []byte("keep"), 0o600); err != nil {
		t.Fatal(err)
	}
	a := newAPI(t)
	opened := decodeProject(t, a.open(dir))

	tests := []struct {
		name     string
		id       string
		body     string
		status   int
		wantCode string
	}{
		{"bad json", opened.ID, `{`, http.StatusBadRequest, "bad_request"},
		{"unknown field", opened.ID, `{"name":"x","path":"../x"}`, http.StatusBadRequest, "bad_request"},
		{"no name", opened.ID, `{"name":"  "}`, http.StatusBadRequest, "invalid_name"},
		{"bad folder", opened.ID, `{"name":"x","folder":"../x"}`, http.StatusBadRequest, "invalid_folder"},
		{"unknown archetype", opened.ID, `{"name":"x","archetype":"nope"}`, http.StatusBadRequest, "unknown_archetype"},
		{"exists", opened.ID, `{"name":"Taken"}`, http.StatusConflict, "page_exists"},
		{"no project id", "", `{"name":"x"}`, http.StatusConflict, "stale_project"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rec := a.createPage(tt.id, tt.body)
			if rec.Code != tt.status {
				t.Fatalf("status = %d, want %d (body: %s)", rec.Code, tt.status, rec.Body)
			}
			if got := decodeError(t, rec); got.Code != tt.wantCode {
				t.Errorf("code = %q, want %q", got.Code, tt.wantCode)
			}
		})
	}

	if data, _ := os.ReadFile(filepath.Join(dir, "content", "taken.md")); string(data) != "keep" {
		t.Errorf("existing page was overwritten: %q", data)
	}
}
