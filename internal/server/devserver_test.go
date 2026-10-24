package server

import (
	"encoding/json"
	"log/slog"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/criscornea/static_studio/internal/devserver"
)

func newDevAPI(t *testing.T) (*api, *Server) {
	t.Helper()
	bin := filepath.Join(t.TempDir(), "fake")
	script := "#!/bin/sh\nexec sleep 30\n"
	if err := os.WriteFile(bin, []byte(script), 0o700); err != nil { //nolint:gosec // test script must be executable
		t.Fatal(err)
	}
	s := New(slog.New(slog.DiscardHandler), nil)
	s.dev = &devserver.Server{Bin: bin, Args: func(int) []string { return nil }}
	t.Cleanup(s.dev.Stop)

	return &api{t: t, handler: s.Routes()}, s
}

func decodeDev(t *testing.T, body string) devServerResponse {
	t.Helper()
	var got devServerResponse
	if err := json.NewDecoder(strings.NewReader(body)).Decode(&got); err != nil {
		t.Fatal(err)
	}

	return got
}

func TestDevServerLifecycle(t *testing.T) {
	a, _ := newDevAPI(t)
	id := decodeProject(t, a.open(newHugoDir(t))).ID

	if rec := a.send(http.MethodGet, "/api/devserver", id, ""); decodeDev(t, rec.Body.String()).Running {
		t.Fatal("running before start")
	}

	rec := a.send(http.MethodPost, "/api/devserver/start", id, "")
	if rec.Code != http.StatusAccepted {
		t.Fatalf("start = %d (%s)", rec.Code, rec.Body)
	}
	st := decodeDev(t, rec.Body.String())
	if !st.Running || !strings.HasPrefix(st.URL, "http://127.0.0.1:") {
		t.Fatalf("status = %+v", st)
	}

	if rec := a.send(http.MethodPost, "/api/devserver/start", id, ""); rec.Code != http.StatusConflict {
		t.Errorf("second start = %d", rec.Code)
	}

	rec = a.send(http.MethodGet, "/api/devserver?path=content/posts/a.md", id, "")
	if got := decodeDev(t, rec.Body.String()).PageURL; got != st.URL+"/posts/a/" {
		t.Errorf("pageUrl = %q", got)
	}

	if rec := a.send(http.MethodPost, "/api/devserver/stop", id, ""); rec.Code != http.StatusNoContent {
		t.Fatalf("stop = %d", rec.Code)
	}
	if rec := a.send(http.MethodGet, "/api/devserver", id, ""); decodeDev(t, rec.Body.String()).Running {
		t.Error("running after stop")
	}
}

func TestDevServerNeedsProject(t *testing.T) {
	a, _ := newDevAPI(t)
	for _, c := range []struct{ method, path string }{
		{http.MethodGet, "/api/devserver"},
		{http.MethodPost, "/api/devserver/start"},
		{http.MethodPost, "/api/devserver/stop"},
	} {
		if rec := a.send(c.method, c.path, "", ""); rec.Code != http.StatusConflict {
			t.Errorf("%s %s = %d", c.method, c.path, rec.Code)
		}
	}
}

func TestCloseStopsDevServerAndProject(t *testing.T) {
	a, s := newDevAPI(t)
	id := decodeProject(t, a.open(newHugoDir(t))).ID
	a.send(http.MethodPost, "/api/devserver/start", id, "")

	s.Close()

	if s.dev.Status().Running {
		t.Error("dev server still running")
	}
	if _, err := s.project.Current(); err == nil {
		t.Error("project still open")
	}
}

func TestProjectCloseStopsDevServer(t *testing.T) {
	a, s := newDevAPI(t)
	id := decodeProject(t, a.open(newHugoDir(t))).ID
	a.send(http.MethodPost, "/api/devserver/start", id, "")

	a.send(http.MethodPost, "/api/project/close", "", "")
	if s.dev.Status().Running {
		t.Error("dev server still running after project close")
	}
}
