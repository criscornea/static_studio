package content

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

const benchPages = 5000

// benchSite writes a generated site of benchPages pages across 50 sections
// and returns its root.
func benchSite(tb testing.TB) string {
	tb.Helper()
	root := tb.TempDir()
	body := strings.Repeat("Lorem ipsum dolor sit amet, consectetur adipiscing elit. ", 40)
	for i := range benchPages {
		dir := filepath.Join(root, "content", fmt.Sprintf("section-%02d", i%50))
		if err := os.MkdirAll(dir, 0o755); err != nil {
			tb.Fatal(err)
		}
		page := fmt.Sprintf("---\ntitle: Page %d\ndate: 2024-01-02\ndraft: %t\ntags: [alpha, beta%d]\n---\n%s\nneedle%d\n", i, i%7 == 0, i%10, body, i)
		if err := os.WriteFile(filepath.Join(dir, fmt.Sprintf("page-%d.md", i)), []byte(page), 0o644); err != nil {
			tb.Fatal(err)
		}
	}
	return root
}

func BenchmarkTree(b *testing.B) {
	fsys := os.DirFS(benchSite(b))
	b.ResetTimer()
	for range b.N {
		if _, err := Tree(fsys, "content"); err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkSearch(b *testing.B) {
	fsys := os.DirFS(benchSite(b))
	b.ResetTimer()
	for range b.N {
		if _, err := Search(fsys, []string{"content"}, "needle4999"); err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkTreeWarm(b *testing.B) {
	fsys := os.DirFS(benchSite(b))
	ix := NewIndex()
	if _, err := ix.Tree(fsys, "content"); err != nil {
		b.Fatal(err)
	}
	b.ResetTimer()
	for range b.N {
		if _, err := ix.Tree(fsys, "content"); err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkSearchWarm(b *testing.B) {
	fsys := os.DirFS(benchSite(b))
	ix := NewIndex()
	if _, err := ix.Search(fsys, []string{"content"}, "needle4999"); err != nil {
		b.Fatal(err)
	}
	b.ResetTimer()
	for range b.N {
		if _, err := ix.Search(fsys, []string{"content"}, "needle4999"); err != nil {
			b.Fatal(err)
		}
	}
}
