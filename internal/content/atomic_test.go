package content

import (
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"syscall"
	"testing"
)

func openTestRoot(t *testing.T) (*os.Root, string) {
	t.Helper()
	dir := t.TempDir()
	root, err := os.OpenRoot(dir)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = root.Close() })
	return root, dir
}

func assertNoTemps(t *testing.T, dir string) {
	t.Helper()
	entries, _ := filepath.Glob(filepath.Join(dir, "*", ".*.tmp"))
	top, _ := filepath.Glob(filepath.Join(dir, ".*.tmp"))
	if len(entries)+len(top) != 0 {
		t.Fatalf("temp files left: %v %v", entries, top)
	}
}

func TestWriteFileAtomic(t *testing.T) {
	root, dir := openTestRoot(t)
	if err := root.Mkdir("content", 0o750); err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{"one", "two"} { // create, then replace
		if err := writeFileAtomic(root, "content/a.md", []byte(want), 0o600); err != nil {
			t.Fatal(err)
		}
		got, _ := os.ReadFile(filepath.Join(dir, "content/a.md"))
		if string(got) != want {
			t.Fatalf("got %q, want %q", got, want)
		}
	}
	assertNoTemps(t, dir)
}

func TestWriteFileAtomicFailureCleansUp(t *testing.T) {
	root, dir := openTestRoot(t)
	// Target is a non-empty directory: rename over it must fail.
	if err := os.MkdirAll(filepath.Join(dir, "a.md", "x"), 0o750); err != nil {
		t.Fatal(err)
	}
	if err := writeFileAtomic(root, "a.md", []byte("data"), 0o600); err == nil {
		t.Fatal("expected error")
	}
	assertNoTemps(t, dir)

	// Missing parent directory: no temp can be created.
	if err := writeFileAtomic(root, "missing/a.md", []byte("data"), 0o600); err == nil {
		t.Fatal("expected error")
	}
	assertNoTemps(t, dir)
}

func TestWriteFileAtomicKeepsMode(t *testing.T) {
	root, dir := openTestRoot(t)
	p := filepath.Join(dir, "a.md")
	if err := os.WriteFile(p, []byte("old"), 0o640); err != nil {
		t.Fatal(err)
	}
	if err := os.Chmod(p, 0o640); err != nil {
		t.Fatal(err)
	}
	if err := writeFileAtomic(root, "a.md", []byte("new"), 0o600); err != nil {
		t.Fatal(err)
	}
	info, _ := os.Stat(p)
	if info.Mode().Perm() != 0o640 {
		t.Fatalf("mode %v, want 0640", info.Mode().Perm())
	}
}

func TestWriteFileAtomicLongName(t *testing.T) {
	root, dir := openTestRoot(t)
	name := strings.Repeat("x", 250) + ".md"
	if err := writeFileAtomic(root, name, []byte("d"), 0o600); err != nil {
		t.Fatal(err)
	}
	assertNoTemps(t, dir)
}

func TestWriteFileAtomicNewLinkUnsupported(t *testing.T) {
	root, dir := openTestRoot(t)
	orig := linkFile
	linkFile = func(*os.Root, string, string) error { return syscall.EPERM }
	t.Cleanup(func() { linkFile = orig })

	if err := writeFileAtomicNew(root, "a.md", []byte("first"), 0o600); err != nil {
		t.Fatal(err)
	}
	got, _ := os.ReadFile(filepath.Join(dir, "a.md"))
	if string(got) != "first" {
		t.Fatalf("got %q", got)
	}
	err := writeFileAtomicNew(root, "a.md", []byte("second"), 0o600)
	if !errors.Is(err, fs.ErrExist) {
		t.Fatalf("got %v, want ErrExist", err)
	}
	got, _ = os.ReadFile(filepath.Join(dir, "a.md"))
	if string(got) != "first" {
		t.Fatalf("file replaced: %q", got)
	}
	assertNoTemps(t, dir)
}

func TestWriteFileAtomicNewFailureCleansUp(t *testing.T) {
	root, dir := openTestRoot(t)
	err := writeFileAtomicNew(root, "missing/a.md", []byte("x"), 0o600)
	if err == nil || errors.Is(err, fs.ErrExist) {
		t.Fatalf("got %v, want non-ErrExist error", err)
	}
	assertNoTemps(t, dir)
}

func TestWriteFileAtomicRefusesEscape(t *testing.T) {
	root, dir := openTestRoot(t)
	outside := t.TempDir()
	if err := os.Symlink(outside, filepath.Join(dir, "link")); err != nil {
		t.Skip("symlinks unavailable")
	}
	for _, name := range []string{"../evil.md", "link/evil.md", "/abs/evil.md"} {
		if err := writeFileAtomic(root, name, []byte("x"), 0o600); err == nil {
			t.Errorf("%s: expected error", name)
		}
		if err := writeFileAtomicNew(root, name, []byte("x"), 0o600); err == nil {
			t.Errorf("%s (new): expected error", name)
		}
	}
	if entries, _ := os.ReadDir(outside); len(entries) != 0 {
		t.Fatalf("escaped write: %v", entries)
	}
}

func TestWriteFileAtomicNewNeverOverwrites(t *testing.T) {
	root, dir := openTestRoot(t)
	if err := writeFileAtomicNew(root, "a.md", []byte("first"), 0o600); err != nil {
		t.Fatal(err)
	}
	err := writeFileAtomicNew(root, "a.md", []byte("second"), 0o600)
	if !errors.Is(err, fs.ErrExist) {
		t.Fatalf("got %v, want ErrExist", err)
	}
	got, _ := os.ReadFile(filepath.Join(dir, "a.md"))
	if string(got) != "first" {
		t.Fatalf("file replaced: %q", got)
	}
	assertNoTemps(t, dir)
}
