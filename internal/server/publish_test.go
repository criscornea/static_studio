package server

import (
	"bufio"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/criscornea/static_studio/internal/build"
	"github.com/criscornea/static_studio/internal/publish"
)

func newPublishAPI(t *testing.T, script string) *api {
	t.Helper()
	bin := filepath.Join(t.TempDir(), "fake")
	if err := os.WriteFile(bin, []byte("#!/bin/sh\n"+script), 0o700); err != nil { //nolint:gosec // test script must be executable
		t.Fatal(err)
	}
	s := New(slog.New(slog.DiscardHandler), nil)
	s.build = &build.Runner{Bin: bin}
	s.UsePublishSettings(publish.NewStore(nil, t.TempDir()))
	return &api{t: t, handler: s.Routes()}
}

func TestPublishSettingsAPI(t *testing.T) {
	a := newPublishAPI(t, "true")
	if rec := a.do(http.MethodGet, "/api/publish/settings", ""); rec.Code != http.StatusConflict {
		t.Fatalf("no project: %d", rec.Code)
	}
	id := decodeProject(t, a.open(newHugoDir(t))).ID

	rec := a.send(http.MethodGet, "/api/publish/settings", id, "")
	if rec.Code != http.StatusOK || !strings.Contains(rec.Body.String(), `"branch":"gh-pages"`) {
		t.Fatalf("get = %d %s", rec.Code, rec.Body)
	}
	// Publishing without a remote fails before any build starts.
	if rec := a.send(http.MethodPost, "/api/publish", id, ""); rec.Code != http.StatusUnprocessableEntity {
		t.Fatalf("publish without settings = %d %s", rec.Code, rec.Body)
	}
	rec = a.send(http.MethodPut, "/api/publish/settings", id, `{"remote":"git@example.com:a/b.git","branch":""}`)
	if rec.Code != http.StatusOK {
		t.Fatalf("put = %d %s", rec.Code, rec.Body)
	}
	rec = a.send(http.MethodGet, "/api/publish/settings", id, "")
	if !strings.Contains(rec.Body.String(), "example.com") {
		t.Fatalf("get after put = %s", rec.Body)
	}
	rec = a.send(http.MethodPut, "/api/publish/settings", id, `{"remote":"--evil","branch":"x"}`)
	if rec.Code != http.StatusUnprocessableEntity || !strings.Contains(rec.Body.String(), "invalid_publish_settings") {
		t.Fatalf("invalid = %d %s", rec.Code, rec.Body)
	}
	rec = a.send(http.MethodPut, "/api/publish/settings", id, `{"nope":1}`)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("bad body = %d", rec.Code)
	}
}

func TestPublishStreamToBareRepo(t *testing.T) {
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git not installed")
	}
	remote := filepath.Join(t.TempDir(), "remote.git")
	if out, err := exec.Command("git", "init", "--bare", "--quiet", remote).CombinedOutput(); err != nil {
		t.Fatalf("%v: %s", err, out)
	}
	a := newPublishAPI(t, "mkdir -p public && echo hi > public/index.html")
	id := decodeProject(t, a.open(newHugoDir(t))).ID
	if rec := a.send(http.MethodPut, "/api/publish/settings", id, `{"remote":"`+remote+`"}`); rec.Code != http.StatusOK {
		t.Fatalf("put = %d %s", rec.Code, rec.Body)
	}

	srv := httptest.NewServer(a.handler)
	defer srv.Close()
	do := func(method, path string) *http.Response {
		req, _ := http.NewRequest(method, srv.URL+path, nil)
		req.Header.Set("X-Project-ID", id)
		res, err := http.DefaultClient.Do(req)
		if err != nil {
			t.Fatal(err)
		}
		return res
	}

	res := do(http.MethodPost, "/api/publish")
	_ = res.Body.Close()
	if res.StatusCode != http.StatusAccepted {
		t.Fatalf("start = %d", res.StatusCode)
	}
	res = do(http.MethodGet, "/api/publish/stream")
	defer func() { _ = res.Body.Close() }()

	var last string
	sc := bufio.NewScanner(res.Body)
	for sc.Scan() {
		if l, ok := strings.CutPrefix(sc.Text(), "data: "); ok {
			last = l
		}
	}
	if last != `{"kind":"done","ok":true}` {
		t.Fatalf("last event = %s", last)
	}
	out, err := exec.Command("git", "--git-dir", remote, "show", "gh-pages:index.html").Output()
	if err != nil || strings.TrimSpace(string(out)) != "hi" {
		t.Fatalf("remote index = %q, %v", out, err)
	}
}

func TestPublishPreflightAPI(t *testing.T) {
	a := newPublishAPI(t, "true")
	if rec := a.do(http.MethodGet, "/api/publish/preflight", ""); rec.Code != http.StatusConflict {
		t.Fatalf("no project: %d", rec.Code)
	}
	id := decodeProject(t, a.open(newHugoDir(t))).ID
	rec := a.send(http.MethodGet, "/api/publish/preflight", id, "")
	if rec.Code != http.StatusOK || !strings.Contains(rec.Body.String(), `"checks"`) {
		t.Fatalf("get = %d %s", rec.Code, rec.Body)
	}
}

func TestSiteAddressAPI(t *testing.T) {
	a := newPublishAPI(t, "true")
	dir := newHugoDir(t)
	if err := os.WriteFile(filepath.Join(dir, "hugo.toml"), []byte("title = 'x' # t\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(dir, "static"), 0o700); err != nil {
		t.Fatal(err)
	}
	cname := filepath.Join(dir, "static", "CNAME")
	if err := os.WriteFile(cname, []byte("blog.example.com\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	id := decodeProject(t, a.open(dir)).ID
	a.send(http.MethodPut, "/api/publish/settings", id, `{"target":"github-pages","remote":"git@github.com:alice/blog.git","branch":"gh-pages"}`)

	// Suggestion only; nothing applied.
	rec := a.send(http.MethodGet, "/api/publish/site-address", id, "")
	if rec.Code != http.StatusOK || !strings.Contains(rec.Body.String(), `"suggestion":"blog.example.com"`) ||
		!strings.Contains(rec.Body.String(), `"configFile":"hugo.toml"`) {
		t.Fatalf("get = %d %s", rec.Code, rec.Body)
	}

	// Project site writes baseURL.
	rec = a.send(http.MethodPut, "/api/publish/site-address", id, `{"siteKind":"project"}`)
	if rec.Code != http.StatusOK || !strings.Contains(rec.Body.String(), `"address":"https://alice.github.io/blog/"`) {
		t.Fatalf("put project = %d %s", rec.Code, rec.Body)
	}
	got, _ := os.ReadFile(filepath.Join(dir, "hugo.toml"))
	if string(got) != "baseURL = \"https://alice.github.io/blog/\"\ntitle = 'x' # t\n" {
		t.Fatalf("hugo.toml = %q", got)
	}

	// Conflict: 409 with both values, nothing saved or written.
	rec = a.send(http.MethodPut, "/api/publish/site-address", id, `{"siteKind":"custom","customDomain":"www.example.com"}`)
	if rec.Code != http.StatusConflict || !strings.Contains(rec.Body.String(), `"setting":"www.example.com"`) ||
		!strings.Contains(rec.Body.String(), `"static":"blog.example.com"`) {
		t.Fatalf("conflict = %d %s", rec.Code, rec.Body)
	}
	if rec := a.send(http.MethodGet, "/api/publish/site-address", id, ""); !strings.Contains(rec.Body.String(), `"siteKind":"project"`) {
		t.Fatalf("conflict saved: %s", rec.Body)
	}

	// Explicit choice adopts static/CNAME.
	rec = a.send(http.MethodPut, "/api/publish/site-address", id, `{"siteKind":"custom","customDomain":"www.example.com","cnameChoice":"static"}`)
	if rec.Code != http.StatusOK || !strings.Contains(rec.Body.String(), `"customDomain":"blog.example.com"`) {
		t.Fatalf("choice = %d %s", rec.Code, rec.Body)
	}
	got, _ = os.ReadFile(filepath.Join(dir, "hugo.toml"))
	if !strings.Contains(string(got), `baseURL = "https://blog.example.com/"`) {
		t.Fatalf("hugo.toml = %q", got)
	}
	if b, _ := os.ReadFile(cname); string(b) != "blog.example.com\n" {
		t.Fatalf("static/CNAME changed: %q", b)
	}

	// Saving the target settings keeps the site address.
	a.send(http.MethodPut, "/api/publish/settings", id, `{"target":"github-pages","remote":"git@github.com:alice/blog.git","branch":"gh-pages"}`)
	if rec := a.send(http.MethodGet, "/api/publish/site-address", id, ""); !strings.Contains(rec.Body.String(), `"siteKind":"custom"`) {
		t.Fatalf("site address lost: %s", rec.Body)
	}

	// Invalid input.
	if rec := a.send(http.MethodPut, "/api/publish/site-address", id, `{"siteKind":"user"}`); rec.Code != http.StatusUnprocessableEntity {
		t.Fatalf("user site on wrong repo = %d %s", rec.Code, rec.Body)
	}
	if rec := a.send(http.MethodPut, "/api/publish/site-address", id, `{"siteKind":"custom","customDomain":"https://x.com"}`); rec.Code != http.StatusUnprocessableEntity {
		t.Fatalf("bad domain = %d", rec.Code)
	}
}
