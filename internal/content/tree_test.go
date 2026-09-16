package content

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"
	"testing/fstest"
	"time"
)

// names returns the child names of n, marking directories with a trailing slash.
func names(n *Node) []string {
	out := make([]string, 0, len(n.Children))
	for _, c := range n.Children {
		if c.IsDir {
			out = append(out, c.Name+"/")
			continue
		}
		out = append(out, c.Name)
	}
	return out
}

func equal(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

func TestTree(t *testing.T) {
	fsys := fstest.MapFS{
		"content/_index.md":              {Data: []byte("root")},
		"content/about.md":               {},
		"content/posts/second.md":        {},
		"content/posts/first.md":         {},
		"content/posts/draft/hidden.md":  {},
		"content/Notes.md":               {},
		"content/image.png":              {},
		"content/.DS_Store":              {},
		"content/.hidden/secret.md":      {},
		"content/empty/readme.txt":       {},
		"content/node_modules/pkg/at.md": {},
		"static/logo.png":                {},
	}

	root, err := Tree(fsys, "content")
	if err != nil {
		t.Fatalf("Tree: %v", err)
	}

	// Directories first, then files, each alphabetically and case-insensitively.
	want := []string{"posts/", "_index.md", "about.md", "Notes.md"}
	if got := names(root); !equal(got, want) {
		t.Errorf("children = %v, want %v", got, want)
	}

	var posts *Node
	for _, c := range root.Children {
		if c.Name == "posts" {
			posts = c
		}
	}
	if posts == nil {
		t.Fatal("posts directory missing")
	}
	if got, want := names(posts), []string{"draft/", "first.md", "second.md"}; !equal(got, want) {
		t.Errorf("posts children = %v, want %v", got, want)
	}
	if posts.Path != "content/posts" {
		t.Errorf("posts path = %q, want content/posts", posts.Path)
	}
}

func TestTreeMetadata(t *testing.T) {
	fsys := fstest.MapFS{"content/post.MD": {Data: []byte("hello")}}

	root, err := Tree(fsys, "content")
	if err != nil {
		t.Fatal(err)
	}
	if len(root.Children) != 1 {
		t.Fatalf("children = %v, want one file", names(root))
	}

	got := root.Children[0]
	if got.Ext != ".md" {
		t.Errorf("ext = %q, want .md (extensions are normalised to lower case)", got.Ext)
	}
	if got.Size != 5 {
		t.Errorf("size = %d, want 5", got.Size)
	}
	if got.IsDir {
		t.Error("IsDir = true, want false")
	}
}

func TestTreeMissingContentDirIsEmptyNotAnError(t *testing.T) {
	root, err := Tree(fstest.MapFS{"hugo.toml": {}}, "content")
	if err != nil {
		t.Fatalf("Tree: %v", err)
	}
	if root == nil {
		t.Fatal("root is nil")
	}
	if len(root.Children) != 0 {
		t.Errorf("children = %v, want none", names(root))
	}
}

// TestTreeSurvivesSymlinkLoop checks that a symlink pointing at an ancestor
// directory does not send the walk into infinite recursion. It runs against
// a real os.Root, because fstest.MapFS has no symlinks.
func TestTreeSurvivesSymlinkLoop(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("symlinks require privileges on Windows")
	}

	dir := t.TempDir()
	posts := filepath.Join(dir, "content", "posts")
	if err := os.MkdirAll(posts, 0o750); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(posts, "first.md"), []byte("x"), 0o600); err != nil {
		t.Fatal(err)
	}
	// content/posts/loop -> content, an ancestor of itself.
	if err := os.Symlink(filepath.Join(dir, "content"), filepath.Join(posts, "loop")); err != nil {
		t.Fatal(err)
	}

	root, err := os.OpenRoot(dir)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = root.Close() })

	done := make(chan struct{})
	go func() {
		defer close(done)
		if _, err := Tree(root.FS(), "content"); err != nil {
			t.Errorf("Tree: %v", err)
		}
	}()

	select {
	case <-done:
	case <-time.After(5 * time.Second):
		t.Fatal("Tree did not finish: the symlink loop was followed")
	}
}
