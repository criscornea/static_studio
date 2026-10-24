package server

import (
	"encoding/json"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"

	"github.com/criscornea/static_studio/internal/recent"
)

func newRecentAPI(t *testing.T) *api {
	t.Helper()
	s := New(slog.New(slog.DiscardHandler), nil)
	s.UseRecent(recent.New(nil, t.TempDir()))
	return &api{t: t, handler: s.Routes()}
}

func TestOpenRecordsRecent(t *testing.T) {
	a := newRecentAPI(t)
	dir := newHugoDir(t)

	if rec := a.open(t.TempDir()); rec.Code == http.StatusOK {
		t.Fatal("expected failed open")
	}
	if rec := a.open(dir); rec.Code != http.StatusOK {
		t.Fatalf("open: %d", rec.Code)
	}
	rec := a.do("GET", "/api/recent", "")
	var got recentResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &got); err != nil {
		t.Fatal(err)
	}
	if len(got.Recent) != 1 || !got.Recent[0].Available || got.Recent[0].Kind != "hugo" {
		t.Fatalf("got %+v", got)
	}
}

func TestRecentUnavailableAndRemove(t *testing.T) {
	a := newRecentAPI(t)
	dir := newHugoDir(t)
	a.open(dir)
	if err := os.RemoveAll(dir); err != nil {
		t.Fatal(err)
	}

	var got recentResponse
	_ = json.Unmarshal(a.do("GET", "/api/recent", "").Body.Bytes(), &got)
	if len(got.Recent) != 1 || got.Recent[0].Available {
		t.Fatalf("got %+v", got)
	}

	body := `{"path":` + string(mustJSON(t, got.Recent[0].Path)) + `}`
	if rec := a.do("DELETE", "/api/recent", body); rec.Code != http.StatusNoContent {
		t.Fatalf("delete: %d", rec.Code)
	}
	_ = json.Unmarshal(a.do("GET", "/api/recent", "").Body.Bytes(), &got)
	if len(got.Recent) != 0 {
		t.Fatalf("got %+v", got)
	}
	if rec := a.do("DELETE", "/api/recent", ""); rec.Code != http.StatusBadRequest {
		t.Fatalf("empty delete: %d", rec.Code)
	}
}

func TestRecentDeleteRejectsForeignOrigin(t *testing.T) {
	a := newRecentAPI(t)
	req := httptest.NewRequest("DELETE", "http://127.0.0.1:7777/api/recent", strings.NewReader(`{"path":"/x"}`))
	req.Header.Set("Origin", "http://evil.example")
	w := httptest.NewRecorder()
	a.handler.ServeHTTP(w, req)
	if w.Code != http.StatusForbidden {
		t.Fatalf("code %d", w.Code)
	}
}

func mustJSON(t *testing.T, v any) []byte {
	t.Helper()
	b, err := json.Marshal(v)
	if err != nil {
		t.Fatal(err)
	}
	return b
}
