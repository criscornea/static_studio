package project

import (
	"errors"
	"log/slog"
	"os"
	"path/filepath"
	"runtime"
	"sync"
	"testing"

	"github.com/criscornea/static_studio/internal/ssg"
)

// newHugoDir creates a minimal Hugo project and returns its path.
func newHugoDir(t *testing.T) string {
	t.Helper()

	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "hugo.toml"), nil, 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(dir, "content"), 0o750); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "content", "post.md"), []byte("# hi\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	return dir
}

func TestOpenAndCurrent(t *testing.T) {
	m := NewManager(slog.New(slog.DiscardHandler))
	t.Cleanup(func() { _ = m.Close() })

	if _, err := m.Current(); !errors.Is(err, ErrNoProject) {
		t.Fatalf("Current before Open: err = %v, want %v", err, ErrNoProject)
	}

	opened, err := m.Open(newHugoDir(t))
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	if opened.ID == "" {
		t.Error("ID is empty")
	}
	if opened.Info.Kind != ssg.KindHugo {
		t.Errorf("Kind = %q, want %q", opened.Info.Kind, ssg.KindHugo)
	}

	current, err := m.Current()
	if err != nil {
		t.Fatalf("Current: %v", err)
	}
	if current != opened {
		t.Error("Current returned a different project than Open")
	}
}

func TestRequireRejectsWrongID(t *testing.T) {
	m := NewManager(slog.New(slog.DiscardHandler))
	t.Cleanup(func() { _ = m.Close() })

	opened, err := m.Open(newHugoDir(t))
	if err != nil {
		t.Fatal(err)
	}

	if _, err := m.Require(opened.ID); err != nil {
		t.Errorf("Require with the correct id: %v", err)
	}
	if _, err := m.Require("wrong"); !errors.Is(err, ErrStaleID) {
		t.Errorf("Require with a wrong id: err = %v, want %v", err, ErrStaleID)
	}
	if _, err := m.Require(""); !errors.Is(err, ErrStaleID) {
		t.Errorf("Require with an empty id: err = %v, want %v", err, ErrStaleID)
	}
}

func TestReopenReplacesAndInvalidates(t *testing.T) {
	m := NewManager(slog.New(slog.DiscardHandler))
	t.Cleanup(func() { _ = m.Close() })

	first, err := m.Open(newHugoDir(t))
	if err != nil {
		t.Fatal(err)
	}
	second, err := m.Open(newHugoDir(t))
	if err != nil {
		t.Fatal(err)
	}

	if first.ID == second.ID {
		t.Error("reopening produced the same id; ids must be unique per open")
	}
	if _, err := m.Require(first.ID); !errors.Is(err, ErrStaleID) {
		t.Errorf("stale id still accepted: err = %v, want %v", err, ErrStaleID)
	}

	// The previous root must be closed, not leaked.
	if _, err := first.Root().Open("hugo.toml"); err == nil {
		t.Error("the replaced project's root is still usable; it was not closed")
	}
}

func TestOpenFailureLeavesPreviousProjectIntact(t *testing.T) {
	m := NewManager(slog.New(slog.DiscardHandler))
	t.Cleanup(func() { _ = m.Close() })

	good, err := m.Open(newHugoDir(t))
	if err != nil {
		t.Fatal(err)
	}

	if _, err := m.Open(t.TempDir()); !errors.Is(err, ssg.ErrNotDetected) {
		t.Fatalf("opening a non-project: err = %v, want %v", err, ssg.ErrNotDetected)
	}

	current, err := m.Current()
	if err != nil {
		t.Fatalf("Current after a failed Open: %v", err)
	}
	if current.ID != good.ID {
		t.Error("a failed Open replaced the working project")
	}
}

func TestRootConfinesAccess(t *testing.T) {
	dir := newHugoDir(t)

	// A symlink inside the project pointing at the parent directory.
	outside := filepath.Dir(dir)
	if err := os.Symlink(outside, filepath.Join(dir, "escape")); err != nil {
		if runtime.GOOS == "windows" {
			t.Skip("symlinks require privileges on Windows")
		}
		t.Fatal(err)
	}

	m := NewManager(slog.New(slog.DiscardHandler))
	t.Cleanup(func() { _ = m.Close() })

	opened, err := m.Open(dir)
	if err != nil {
		t.Fatal(err)
	}
	root := opened.Root()

	if _, err := root.Open("content/post.md"); err != nil {
		t.Fatalf("a legitimate path inside the project failed: %v", err)
	}

	escapes := []string{
		"../hugo.toml",
		"content/../../hugo.toml",
		"escape/",
		"escape/anything.md",
		"/etc/passwd",
		filepath.Join(outside, "hugo.toml"),
	}
	for _, name := range escapes {
		t.Run(name, func(t *testing.T) {
			f, err := root.Open(name)
			if err == nil {
				_ = f.Close()
				t.Errorf("escaped the project root via %q", name)
			}
		})
	}
}

func TestConcurrentOpenAndCurrent(t *testing.T) {
	m := NewManager(slog.New(slog.DiscardHandler))
	t.Cleanup(func() { _ = m.Close() })

	if _, err := m.Open(newHugoDir(t)); err != nil {
		t.Fatal(err)
	}
	dirs := []string{newHugoDir(t), newHugoDir(t), newHugoDir(t)}

	var wg sync.WaitGroup
	for i := range 8 {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for j := range 50 {
				if i%4 == 0 {
					if _, err := m.Open(dirs[j%len(dirs)]); err != nil {
						t.Errorf("Open: %v", err)
						return
					}
					continue
				}
				if op, err := m.Current(); err == nil {
					_ = op.Info.Root
					_ = op.ID
				}
			}
		}()
	}
	wg.Wait()
}
