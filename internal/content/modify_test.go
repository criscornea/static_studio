package content

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/criscornea/static_studio/internal/frontmatter"
)

func writeTest(t *testing.T, dir, name, data string) {
	t.Helper()
	p := filepath.Join(dir, name)
	if err := os.MkdirAll(filepath.Dir(p), 0o750); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(p, []byte(data), 0o600); err != nil {
		t.Fatal(err)
	}
}

func TestRenamePage(t *testing.T) {
	root, dir := openTestRoot(t)
	writeTest(t, dir, "content/a.md", "A")
	writeTest(t, dir, "content/b.md", "B")
	writeTest(t, dir, "content/img.png", "x")
	if err := os.Symlink("a.md", filepath.Join(dir, "content/link.md")); err != nil {
		t.Fatal(err)
	}
	if err := os.Mkdir(filepath.Join(dir, "content/d.md"), 0o750); err != nil {
		t.Fatal(err)
	}

	refusals := []struct {
		name, from, to string
		want           error
	}{
		{"exists", "content/a.md", "content/b.md", ErrPageExists},
		{"same", "content/a.md", "content/a.md", ErrPageExists},
		{"missing", "content/nope.md", "content/z.md", ErrNotFound},
		{"escape to", "content/a.md", "content/../x.md", ErrNotEditable},
		{"outside to", "content/a.md", "other/x.md", ErrNotEditable},
		{"outside from", "other/a.md", "content/x.md", ErrNotEditable},
		{"absolute", "/etc/passwd", "content/x.md", ErrNotEditable},
		{"bad ext to", "content/a.md", "content/a.png", ErrNotEditable},
		{"bad ext from", "content/img.png", "content/i.md", ErrNotEditable},
		{"hidden folder", "content/a.md", "content/.h/a.md", ErrNotEditable},
		{"symlink", "content/link.md", "content/x.md", ErrNotEditable},
		{"dir", "content/d.md", "content/x.md", ErrNotEditable},
	}
	for _, tc := range refusals {
		t.Run(tc.name, func(t *testing.T) {
			if _, err := RenamePage(root, "content", tc.from, tc.to); !errors.Is(err, tc.want) {
				t.Fatalf("got %v, want %v", err, tc.want)
			}
		})
	}
	if got, _ := os.ReadFile(filepath.Join(dir, "content/b.md")); string(got) != "B" {
		t.Fatalf("b.md overwritten: %q", got)
	}

	to, err := RenamePage(root, "content", "content/a.md", "content/sub/c.md")
	if err != nil || to != "content/sub/c.md" {
		t.Fatalf("got %q, %v", to, err)
	}
	if got, _ := os.ReadFile(filepath.Join(dir, "content/sub/c.md")); string(got) != "A" {
		t.Fatalf("got %q", got)
	}
	if _, err := os.Stat(filepath.Join(dir, "content/a.md")); !os.IsNotExist(err) {
		t.Fatal("old file still exists")
	}
}

func TestRenamePageWithoutHardLinks(t *testing.T) {
	root, dir := openTestRoot(t)
	writeTest(t, dir, "content/a.md", "A")
	writeTest(t, dir, "content/b.md", "B")
	old := linkFile
	linkFile = func(*os.Root, string, string) error { return errors.New("unsupported") }
	t.Cleanup(func() { linkFile = old })

	if _, err := RenamePage(root, "content", "content/a.md", "content/b.md"); !errors.Is(err, ErrPageExists) {
		t.Fatalf("got %v", err)
	}
	if _, err := RenamePage(root, "content", "content/a.md", "content/c.md"); err != nil {
		t.Fatal(err)
	}
	if got, _ := os.ReadFile(filepath.Join(dir, "content/c.md")); string(got) != "A" {
		t.Fatalf("got %q", got)
	}
}

func TestDeletePage(t *testing.T) {
	root, dir := openTestRoot(t)
	writeTest(t, dir, "content/a.md", "A")
	writeTest(t, dir, "content/img.png", "x")
	writeTest(t, dir, "outside.md", "o")
	if err := os.Mkdir(filepath.Join(dir, "content/d.md"), 0o750); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink("../outside.md", filepath.Join(dir, "content/link.md")); err != nil {
		t.Fatal(err)
	}

	for name, want := range map[string]error{
		"content/nope.md":       ErrNotFound,
		"content/img.png":       ErrNotEditable,
		"content/d.md":          ErrNotEditable,
		"content/link.md":       ErrNotEditable,
		"content/../outside.md": ErrNotEditable,
		"outside.md":            ErrNotEditable,
		"/etc/passwd":           ErrNotEditable,
		"content":               ErrNotEditable,
	} {
		if err := DeletePage(root, "content", name); !errors.Is(err, want) {
			t.Errorf("%s: got %v, want %v", name, err, want)
		}
	}
	for _, keep := range []string{"content/img.png", "content/d.md", "outside.md"} {
		if _, err := os.Lstat(filepath.Join(dir, keep)); err != nil {
			t.Errorf("%s removed", keep)
		}
	}
	if err := DeletePage(root, "content", "content/a.md"); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(dir, "content/a.md")); !os.IsNotExist(err) {
		t.Fatal("not deleted")
	}
}

func TestPageSymlinkedFolder(t *testing.T) {
	root, dir := openTestRoot(t)
	writeTest(t, dir, "layouts/x.html", "L")
	writeTest(t, dir, "content/a.md", "A")
	if err := os.Symlink("../layouts", filepath.Join(dir, "content/lnk")); err != nil {
		t.Fatal(err)
	}

	if err := DeletePage(root, "content", "content/lnk/x.html"); !errors.Is(err, ErrNotEditable) {
		t.Fatalf("delete: %v", err)
	}
	if _, err := RenamePage(root, "content", "content/lnk/x.html", "content/y.html"); !errors.Is(err, ErrNotEditable) {
		t.Fatalf("rename from: %v", err)
	}
	if _, err := RenamePage(root, "content", "content/a.md", "content/lnk/sub/a.md"); !errors.Is(err, ErrNotEditable) {
		t.Fatalf("rename to: %v", err)
	}
	if _, err := os.Stat(filepath.Join(dir, "layouts/x.html")); err != nil {
		t.Fatal("layouts/x.html touched")
	}
	if _, err := os.Stat(filepath.Join(dir, "layouts/sub")); !os.IsNotExist(err) {
		t.Fatal("layouts/sub created")
	}
}

func TestPageHiddenSegments(t *testing.T) {
	root, dir := openTestRoot(t)
	writeTest(t, dir, "content/.h/a.md", "A")
	writeTest(t, dir, "content/node_modules/b.md", "B")
	writeTest(t, dir, "content/c.md", "C")

	for _, name := range []string{"content/.h/a.md", "content/node_modules/b.md"} {
		if err := DeletePage(root, "content", name); !errors.Is(err, ErrNotEditable) {
			t.Errorf("delete %s: %v", name, err)
		}
		if _, err := RenamePage(root, "content", name, "content/z.md"); !errors.Is(err, ErrNotEditable) {
			t.Errorf("rename %s: %v", name, err)
		}
		if _, err := os.Stat(filepath.Join(dir, name)); err != nil {
			t.Errorf("%s removed", name)
		}
	}
}

func TestRenamePageCaseOnly(t *testing.T) {
	root, dir := openTestRoot(t)
	writeTest(t, dir, "content/a.md", "A")
	if _, err := os.Stat(filepath.Join(dir, "content/A.md")); err != nil {
		t.Skip("case-sensitive filesystem")
	}
	if _, err := RenamePage(root, "content", "content/a.md", "content/A.md"); err != nil {
		t.Fatal(err)
	}
}

func TestSavePage(t *testing.T) {
	root, dir := openTestRoot(t)
	writeTest(t, dir, "content/a.md", "---\ntitle: A\n---\nold\n")
	writeTest(t, dir, "content/plain.md", "old\n")
	writeTest(t, dir, "content/img.png", "x")
	if err := os.Mkdir(filepath.Join(dir, "content/d.md"), 0o750); err != nil {
		t.Fatal(err)
	}

	read := func(name string) []byte {
		b, err := os.ReadFile(filepath.Join(dir, name))
		if err != nil {
			t.Fatal(err)
		}
		return b
	}

	data := read("content/a.md")
	h, err := SavePage(root, "content", "content/a.md", "new\n", hashBytes(data), nil)
	if err != nil {
		t.Fatal(err)
	}
	got := read("content/a.md")
	if string(got) != "---\ntitle: A\n---\nnew\n" {
		t.Fatalf("got %q", got)
	}
	if h != hashBytes(got) {
		t.Fatal("returned hash does not match file")
	}

	// Stale hash: file untouched.
	if _, err := SavePage(root, "content", "content/a.md", "x", hashBytes(data), nil); !errors.Is(err, ErrConflict) {
		t.Fatalf("got %v, want ErrConflict", err)
	}
	if string(read("content/a.md")) != string(got) {
		t.Fatal("file changed on conflict")
	}

	// No frontmatter.
	p := read("content/plain.md")
	if _, err := SavePage(root, "content", "content/plain.md", "n", hashBytes(p), nil); err != nil {
		t.Fatal(err)
	}
	if string(read("content/plain.md")) != "n" {
		t.Fatal("plain page not saved")
	}

	for _, name := range []string{"content/img.png", "content/d.md", "static/x.md", "../x.md", "content/.h/a.md"} {
		if _, err := SavePage(root, "content", name, "x", "", nil); !errors.Is(err, ErrNotEditable) {
			t.Errorf("%s: got %v, want ErrNotEditable", name, err)
		}
	}
	if _, err := SavePage(root, "content", "content/missing.md", "x", "", nil); !errors.Is(err, ErrNotFound) {
		t.Errorf("missing: got %v", err)
	}
}

func TestSavePageFields(t *testing.T) {
	root, dir := openTestRoot(t)
	writeTest(t, dir, "content/y.md", "---\ntitle: Y\nweight: 3\n---\nold\n")
	writeTest(t, dir, "content/t.md", "+++\ntitle = 'T'\nweight = 3\n+++\nold\n")
	writeTest(t, dir, "content/p.md", "plain\n")

	for _, tc := range []struct{ name, fence string }{
		{"content/y.md", "---\n"}, {"content/t.md", "+++\n"}, {"content/p.md", "---\n"},
	} {
		data, err := os.ReadFile(filepath.Join(dir, tc.name))
		if err != nil {
			t.Fatal(err)
		}
		f := &frontmatter.Fields{Title: "New", Draft: true, Tags: []string{"a"}}
		if _, err := SavePage(root, "content", tc.name, "body\n", hashBytes(data), f); err != nil {
			t.Fatalf("%s: %v", tc.name, err)
		}
		got, _ := os.ReadFile(filepath.Join(dir, tc.name))
		s := string(got)
		if !strings.HasPrefix(s, tc.fence) || !strings.HasSuffix(s, tc.fence+"body\n") ||
			!strings.Contains(s, "New") || (tc.name != "content/p.md" && !strings.Contains(s, "weight")) {
			t.Errorf("%s: got %q", tc.name, s)
		}
	}
}

func TestSavePageFieldsCRLF(t *testing.T) {
	root, dir := openTestRoot(t)
	writeTest(t, dir, "content/y.md", "---\r\ntitle: Y\r\nweight: 3\r\n---\r\nold\r\n")
	writeTest(t, dir, "content/t.md", "+++\r\ntitle = 'T'\r\nweight = 3\r\n+++\r\nold\r\n")

	for _, tc := range []struct{ name, fence string }{
		{"content/y.md", "---\r\n"}, {"content/t.md", "+++\r\n"},
	} {
		data, err := os.ReadFile(filepath.Join(dir, tc.name))
		if err != nil {
			t.Fatal(err)
		}
		f := &frontmatter.Fields{Title: "New", Draft: true, Tags: []string{"a"}}
		if _, err := SavePage(root, "content", tc.name, "body\r\n", hashBytes(data), f); err != nil {
			t.Fatalf("%s: %v", tc.name, err)
		}
		got, _ := os.ReadFile(filepath.Join(dir, tc.name))
		s := string(got)
		head := strings.TrimSuffix(s, "body\r\n")
		if !strings.HasPrefix(s, tc.fence) || !strings.HasSuffix(head, tc.fence) ||
			strings.Contains(strings.ReplaceAll(head, "\r\n", ""), "\n") || !strings.Contains(s, "New") {
			t.Errorf("%s: got %q", tc.name, s)
		}
	}
}
