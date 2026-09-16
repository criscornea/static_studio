package server

import (
	"encoding/json"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/criscornea/static_studio/internal/ssg"
)

func getProject(t *testing.T, dir string) *httptest.ResponseRecorder {
	t.Helper()

	target := "/api/project?path=" + url.QueryEscape(dir)
	req := httptest.NewRequest(http.MethodGet, target, nil)
	rec := httptest.NewRecorder()

	New(slog.New(slog.DiscardHandler), nil).Routes().ServeHTTP(rec, req)

	return rec
}

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
		t.Error("Message is empty; errors must be readable by a non-dev")
	}

	return got
}

func TestOpenProjectSuccess(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "hugo.toml"), nil, 0o600); err != nil {
		t.Fatal(err)
	}

	rec := getProject(t, dir)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d (body: %s)", rec.Code, http.StatusOK, rec.Body)
	}

	var got ssg.Project
	if err := json.NewDecoder(rec.Body).Decode(&got); err != nil {
		t.Fatalf("decoding body: %v", err)
	}
	if got.Kind != ssg.KindHugo {
		t.Errorf("kind = %q, want %q", got.Kind, ssg.KindHugo)
	}
	if got.ConfigFile != "hugo.toml" {
		t.Errorf("configFile = %q, want hugo.toml", got.ConfigFile)
	}
	if !filepath.IsAbs(got.Root) {
		t.Errorf("root = %q, want an absolute path", got.Root)
	}
}

func TestOpenProjectErrors(t *testing.T) {
	emptyDir := t.TempDir()

	astroDir := t.TempDir()
	if err := os.WriteFile(filepath.Join(astroDir, "astro.config.mjs"), nil, 0o600); err != nil {
		t.Fatal(err)
	}

	missing := filepath.Join(t.TempDir(), "nope")

	tests := []struct {
		name       string
		dir        string
		wantStatus int
		wantCode   string
	}{
		{"empty path", "", http.StatusBadRequest, "missing_path"},
		{"not a project", emptyDir, http.StatusUnprocessableEntity, "not_a_project"},
		{"astro project", astroDir, http.StatusNotImplemented, "unsupported_generator"},
		{"missing directory", missing, http.StatusBadRequest, "cannot_open"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rec := getProject(t, tt.dir)

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

	got := decodeError(t, getProject(t, missing))

	if strings.Contains(got.Message, missing) {
		t.Errorf("message leaks the filesystem path: %q", got.Message)
	}
	if strings.Contains(strings.ToLower(got.Message), "no such file") {
		t.Errorf("message leaks the underlying OS err: %q", got.Message)
	}
}
