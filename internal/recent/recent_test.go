package recent

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"
)

func TestAddDedupeOrderCap(t *testing.T) {
	s := New(nil, t.TempDir())
	for i := range 12 {
		if err := s.Add(filepath.Join(t.TempDir(), fmt.Sprint("p", i)), "hugo"); err != nil {
			t.Fatal(err)
		}
	}
	list := s.List()
	if len(list) != Max {
		t.Fatalf("len = %d, want %d", len(list), Max)
	}
	if list[0].Name != "p11" {
		t.Errorf("first = %q, want p11", list[0].Name)
	}
	if err := s.Add(list[3].Path, "hugo"); err != nil {
		t.Fatal(err)
	}
	again := s.List()
	if len(again) != Max || again[0].Path != list[3].Path {
		t.Errorf("dedupe/move-to-front failed: %v", again)
	}
}

func TestRemove(t *testing.T) {
	s := New(nil, t.TempDir())
	p := filepath.Join(t.TempDir(), "site")
	_ = s.Add(p, "hugo")
	if err := s.Remove(p); err != nil {
		t.Fatal(err)
	}
	if got := s.List(); len(got) != 0 {
		t.Errorf("got %v", got)
	}
}

func TestMissingAndCorrupt(t *testing.T) {
	dir := t.TempDir()
	s := New(nil, dir)
	if got := s.List(); got == nil || len(got) != 0 {
		t.Errorf("missing file: %v", got)
	}
	f := filepath.Join(dir, "static-studio", "recent.json")
	_ = os.MkdirAll(filepath.Dir(f), 0o700)
	_ = os.WriteFile(f, []byte("{nope"), 0o600)
	if got := s.List(); len(got) != 0 {
		t.Errorf("corrupt file: %v", got)
	}
	if err := s.Add(t.TempDir(), "hugo"); err != nil {
		t.Fatal(err)
	}
	if len(s.List()) != 1 {
		t.Error("add after corrupt should recover")
	}
}

func TestDisabled(t *testing.T) {
	s := New(nil, "")
	if err := s.Add(t.TempDir(), "hugo"); err != nil {
		t.Fatal(err)
	}
	if len(s.List()) != 0 {
		t.Error("disabled store should be empty")
	}
}

func TestNoTempLeftBehind(t *testing.T) {
	dir := t.TempDir()
	s := New(nil, dir)
	_ = s.Add(t.TempDir(), "hugo")
	ents, _ := os.ReadDir(filepath.Join(dir, "static-studio"))
	if len(ents) != 1 {
		t.Errorf("entries = %d, want 1", len(ents))
	}
}
