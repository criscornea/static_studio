package content

import (
	"os"
	"path/filepath"
	"sync"
	"testing"
	"time"
)

func indexSite(t *testing.T) (string, struct{}) {
	t.Helper()
	root := t.TempDir()
	if err := os.MkdirAll(filepath.Join(root, "content"), 0o755); err != nil {
		t.Fatal(err)
	}
	return root, struct{}{}
}

func write(t *testing.T, root, name, data string, mod time.Time) {
	t.Helper()
	p := filepath.Join(root, name)
	if err := os.WriteFile(p, []byte(data), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.Chtimes(p, mod, mod); err != nil {
		t.Fatal(err)
	}
}

func titleOf(t *testing.T, ix *Index, root string) string {
	t.Helper()
	tree, err := ix.Tree(os.DirFS(root), "content")
	if err != nil {
		t.Fatal(err)
	}
	if len(tree.Children) != 1 || tree.Children[0].Fields == nil {
		t.Fatalf("unexpected tree: %+v", tree.Children)
	}
	return tree.Children[0].Fields.Title
}

func TestIndexReusesUnchangedPage(t *testing.T) {
	root, _ := indexSite(t)
	ix := NewIndex()
	t0 := time.Now().Add(-time.Hour)
	write(t, root, "content/a.md", "---\ntitle: One\n---\nbody\n", t0)
	if got := titleOf(t, ix, root); got != "One" {
		t.Fatalf("title = %q", got)
	}
	first := ix.entries["content/a.md"]
	titleOf(t, ix, root)
	if ix.entries["content/a.md"] != first {
		t.Fatal("unchanged page was reloaded")
	}
	if ix.Len() != 1 {
		t.Fatalf("Len = %d", ix.Len())
	}
}

func TestIndexSeesExternalEdits(t *testing.T) {
	root, _ := indexSite(t)
	ix := NewIndex()
	t0 := time.Now().Add(-time.Hour)
	write(t, root, "content/a.md", "---\ntitle: One\n---\nbody\n", t0)
	titleOf(t, ix, root)

	t.Run("same size new mtime", func(t *testing.T) {
		write(t, root, "content/a.md", "---\ntitle: Two\n---\nbody\n", t0.Add(time.Minute))
		if got := titleOf(t, ix, root); got != "Two" {
			t.Fatalf("title = %q", got)
		}
	})

	t.Run("same size same mtime new file", func(t *testing.T) {
		p := filepath.Join(root, "content/a.md")
		info, _ := os.Stat(p)
		tmp := filepath.Join(root, "content/tmp")
		write(t, root, "content/tmp", "---\ntitle: Tri\n---\nbody\n", info.ModTime())
		if err := os.Rename(tmp, p); err != nil {
			t.Fatal(err)
		}
		if got := titleOf(t, ix, root); got != "Tri" {
			t.Fatalf("title = %q", got)
		}
	})

	t.Run("deleted", func(t *testing.T) {
		if err := os.Remove(filepath.Join(root, "content/a.md")); err != nil {
			t.Fatal(err)
		}
		tree, err := ix.Tree(os.DirFS(root), "content")
		if err != nil || len(tree.Children) != 0 {
			t.Fatalf("tree = %+v, err = %v", tree, err)
		}
	})
}

func TestIndexSearchSeesExternalEdits(t *testing.T) {
	root, _ := indexSite(t)
	ix := NewIndex()
	fsys := os.DirFS(root)
	t0 := time.Now().Add(-time.Hour)
	write(t, root, "content/a.md", "---\ntitle: One\n---\nalpha\n", t0)
	res, err := ix.Search(fsys, []string{"content"}, "alpha")
	if err != nil || len(res.Hits) != 1 {
		t.Fatalf("res = %+v, err = %v", res, err)
	}
	write(t, root, "content/a.md", "---\ntitle: One\n---\nbravo\n", t0.Add(time.Minute))
	res, _ = ix.Search(fsys, []string{"content"}, "alpha")
	if len(res.Hits) != 0 {
		t.Fatalf("stale hit: %+v", res.Hits)
	}
	res, _ = ix.Search(fsys, []string{"content"}, "bravo")
	if len(res.Hits) != 1 {
		t.Fatalf("missing hit: %+v", res.Hits)
	}
}

func TestIndexDropAndNil(t *testing.T) {
	root, _ := indexSite(t)
	ix := NewIndex()
	write(t, root, "content/a.md", "---\ntitle: One\n---\n", time.Now())
	titleOf(t, ix, root)
	ix.Drop("content/a.md")
	if ix.Len() != 0 {
		t.Fatal("Drop kept the entry")
	}
	titleOf(t, ix, root)
	ix.Reset()
	if ix.Len() != 0 {
		t.Fatal("Reset kept entries")
	}

	var none *Index
	none.Drop("x")
	none.Reset()
	if none.Len() != 0 || titleOf(t, none, root) != "One" {
		t.Fatal("nil index misbehaves")
	}
}

func TestIndexConcurrent(t *testing.T) {
	root, _ := indexSite(t)
	ix := NewIndex()
	fsys := os.DirFS(root)
	t0 := time.Now().Add(-time.Hour)
	for _, n := range []string{"a", "b", "c"} {
		write(t, root, "content/"+n+".md", "---\ntitle: "+n+"\n---\nneedle\n", t0)
	}
	var wg sync.WaitGroup
	for i := range 8 {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for j := range 50 {
				switch i % 3 {
				case 0:
					_, _ = ix.Tree(fsys, "content")
				case 1:
					_, _ = ix.Search(fsys, []string{"content"}, "needle")
				default:
					write(t, root, "content/a.md", "---\ntitle: a\n---\nneedle\n", t0.Add(time.Duration(j)*time.Second))
					ix.Drop("content/a.md")
				}
			}
		}()
	}
	wg.Wait()
}

func TestIndexFollowsSymlinkedPage(t *testing.T) {
	root, _ := indexSite(t)
	ix := NewIndex()
	t0 := time.Now().Add(-time.Hour)
	write(t, root, "a.md", "---\ntitle: One\n---\nbody\n", t0)
	if err := os.Symlink("../a.md", filepath.Join(root, "content/link.md")); err != nil {
		t.Skip(err)
	}
	if got := titleOf(t, ix, root); got != "One" {
		t.Fatalf("title = %q", got)
	}
	write(t, root, "a.md", "---\ntitle: Two\n---\nbody\n", t0.Add(time.Minute))
	if got := titleOf(t, ix, root); got != "Two" {
		t.Fatalf("title after target edit = %q", got)
	}
}

func TestIndexPrunesVanishedPages(t *testing.T) {
	root, _ := indexSite(t)
	ix := NewIndex()
	t0 := time.Now().Add(-time.Hour)
	write(t, root, "content/a.md", "---\ntitle: One\n---\n", t0)
	write(t, root, "content/b.md", "---\ntitle: Two\n---\n", t0)
	if _, err := ix.Tree(os.DirFS(root), "content"); err != nil {
		t.Fatal(err)
	}
	if err := os.Remove(filepath.Join(root, "content/b.md")); err != nil {
		t.Fatal(err)
	}
	if _, err := ix.Tree(os.DirFS(root), "content"); err != nil {
		t.Fatal(err)
	}
	if ix.Len() != 1 {
		t.Fatalf("Len = %d, want 1", ix.Len())
	}
}
