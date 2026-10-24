package server

import (
	"bufio"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/criscornea/static_studio/internal/build"
)

func newBuildAPI(t *testing.T, script string) *api {
	t.Helper()
	bin := filepath.Join(t.TempDir(), "fake")
	if err := os.WriteFile(bin, []byte("#!/bin/sh\n"+script), 0o700); err != nil { //nolint:gosec // test script must be executable
		t.Fatal(err)
	}
	s := New(slog.New(slog.DiscardHandler), nil)
	s.build = &build.Runner{Bin: bin}
	return &api{t: t, handler: s.Routes()}
}

func TestBuildStream(t *testing.T) {
	a := newBuildAPI(t, "echo first; echo second")
	opened := decodeProject(t, a.open(newHugoDir(t)))

	srv := httptest.NewServer(a.handler)
	defer srv.Close()

	do := func(method, path string) *http.Response {
		req, _ := http.NewRequest(method, srv.URL+path, nil)
		req.Header.Set("X-Project-ID", opened.ID)
		res, err := http.DefaultClient.Do(req)
		if err != nil {
			t.Fatal(err)
		}
		return res
	}

	res := do(http.MethodPost, "/api/build")
	_ = res.Body.Close()
	if res.StatusCode != http.StatusAccepted {
		t.Fatalf("start status = %d", res.StatusCode)
	}

	res = do(http.MethodGet, "/api/build/stream")
	defer func() { _ = res.Body.Close() }()
	if ct := res.Header.Get("Content-Type"); ct != "text/event-stream" {
		t.Fatalf("content type = %q", ct)
	}

	var data []string
	sc := bufio.NewScanner(res.Body)
	for sc.Scan() {
		if l, ok := strings.CutPrefix(sc.Text(), "data: "); ok {
			data = append(data, l)
		}
	}
	want := []string{
		`{"kind":"line","line":"first"}`,
		`{"kind":"line","line":"second"}`,
		`{"kind":"done","ok":true}`,
	}
	if strings.Join(data, "\n") != strings.Join(want, "\n") {
		t.Errorf("events = %q", data)
	}
}

func TestBuildBusyAndGuards(t *testing.T) {
	a := newBuildAPI(t, "exec sleep 30")
	if rec := a.do(http.MethodPost, "/api/build", ""); rec.Code != http.StatusConflict {
		t.Fatalf("no project: status = %d", rec.Code)
	}
	opened := decodeProject(t, a.open(newHugoDir(t)))

	post := func(path string) *httptest.ResponseRecorder {
		req := httptest.NewRequest(http.MethodPost, "http://127.0.0.1:7777"+path, nil)
		req.Header.Set("X-Project-ID", opened.ID)
		rec := httptest.NewRecorder()
		a.handler.ServeHTTP(rec, req)
		return rec
	}

	if rec := post("/api/build"); rec.Code != http.StatusAccepted {
		t.Fatalf("start = %d", rec.Code)
	}
	rec := post("/api/build")
	if rec.Code != http.StatusConflict || !strings.Contains(rec.Body.String(), "build_running") {
		t.Fatalf("second start = %d %s", rec.Code, rec.Body)
	}
	if rec := post("/api/build/cancel"); rec.Code != http.StatusNoContent {
		t.Fatalf("cancel = %d", rec.Code)
	}

	deadline := time.Now().Add(5 * time.Second)
	for post("/api/build").Code == http.StatusConflict {
		if time.Now().After(deadline) {
			t.Fatal("build never stopped")
		}
		time.Sleep(10 * time.Millisecond)
	}
	post("/api/build/cancel")
}

func TestBuildStreamWithoutBuild(t *testing.T) {
	a := newBuildAPI(t, "true")
	opened := decodeProject(t, a.open(newHugoDir(t)))

	req := httptest.NewRequest(http.MethodGet, "http://127.0.0.1:7777/api/build/stream", nil)
	req.Header.Set("X-Project-ID", opened.ID)
	rec := httptest.NewRecorder()
	a.handler.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK || rec.Body.Len() != 0 {
		t.Errorf("status = %d body = %q", rec.Code, rec.Body)
	}
}

func TestBuildStreamOutlivesWriteTimeout(t *testing.T) {
	a := newBuildAPI(t, "echo first; sleep 1; echo second")
	opened := decodeProject(t, a.open(newHugoDir(t)))

	srv := httptest.NewUnstartedServer(a.handler)
	srv.Config.WriteTimeout = 200 * time.Millisecond
	srv.Start()
	defer srv.Close()

	do := func(method, path string) *http.Response {
		req, _ := http.NewRequest(method, srv.URL+path, nil)
		req.Header.Set("X-Project-ID", opened.ID)
		res, err := http.DefaultClient.Do(req)
		if err != nil {
			t.Fatal(err)
		}
		return res
	}

	_ = do(http.MethodPost, "/api/build").Body.Close()
	res := do(http.MethodGet, "/api/build/stream")
	defer func() { _ = res.Body.Close() }()

	var data []string
	sc := bufio.NewScanner(res.Body)
	for sc.Scan() {
		if l, ok := strings.CutPrefix(sc.Text(), "data: "); ok {
			data = append(data, l)
		}
	}
	if len(data) != 3 || !strings.Contains(data[2], `"done"`) {
		t.Errorf("events = %q", data)
	}
}
