package recent

import (
	"os"
	"path/filepath"
	"testing"
)

// A config dir that is really a file makes every write fail; the error must surface.
func TestAddFailsWhenConfigDirIsAFile(t *testing.T) {
	cfg := filepath.Join(t.TempDir(), "cfg")
	if err := os.WriteFile(cfg, []byte("x"), 0o600); err != nil {
		t.Fatal(err)
	}
	s := New(nil, cfg)
	if err := s.Add(t.TempDir(), "hugo"); err == nil {
		t.Fatal("want error")
	}
	if got := s.List(); len(got) != 0 {
		t.Fatalf("list = %v, want empty", got)
	}
}

func TestCorruptFileIsIgnoredThenReplaced(t *testing.T) {
	cfg := t.TempDir()
	s := New(nil, cfg)
	if err := os.MkdirAll(filepath.Join(cfg, "static-studio"), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(s.file, []byte("{not json"), 0o600); err != nil {
		t.Fatal(err)
	}
	if got := s.List(); len(got) != 0 {
		t.Fatalf("list = %v", got)
	}
	if err := s.Add(t.TempDir(), "hugo"); err != nil {
		t.Fatal(err)
	}
	if got := s.List(); len(got) != 1 {
		t.Fatalf("list = %v", got)
	}
}

func TestNewDefaultAndDisabled(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	if s := NewDefault(nil); s.file == "" {
		t.Error("default store disabled")
	}
	d := New(nil, "")
	if err := d.Add("x", "hugo"); err != nil || len(d.List()) != 0 {
		t.Error("disabled store should ignore writes")
	}
}
