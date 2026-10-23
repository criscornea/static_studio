package content

import (
	"errors"
	"strings"
	"testing"
	"testing/fstest"
)

func searchFS() fstest.MapFS {
	return fstest.MapFS{
		"content/a-body.md":      {Data: []byte("---\ntitle: Alpha\n---\nSome text about GOLANG here.\n")},
		"content/b-title.md":     {Data: []byte("---\ntitle: Learning Golang\ndraft: true\n---\nNothing.\n")},
		"content/c-tag.md":       {Data: []byte("+++\ntitle = 'Gamma'\ntags = ['golang']\n+++\nNothing.\n")},
		"content/d-none.md":      {Data: []byte("# plain golang page\n")},
		"content/e-broken.md":    {Data: []byte("---\ntitle: [unclosed\n---\ngolang survives\n")},
		"content/f-miss.md":      {Data: []byte("---\ntitle: Other\n---\nunrelated\n")},
		"content/sub/g-deep.md":  {Data: []byte("deep golang\n")},
		"content/.hidden/h.md":   {Data: []byte("golang hidden\n")},
		"content/image.png":      {Data: []byte("golang")},
		"content/i-fm-only.md":   {Data: []byte("---\ntitle: Other\nsummary: golang\n---\nbody\n")},
		"static/outside.md":      {Data: []byte("golang")},
		"content/j-unicode.md":   {Data: []byte("Über die Größe von Straße — golang — ünïcödé\n")},
		"content/k-multiline.md": {Data: []byte("first line\n\n\n  golang\n\tlast   line\n")},
	}
}

func paths(r *Results) []string {
	out := make([]string, len(r.Hits))
	for i, h := range r.Hits {
		out[i] = h.Path
	}
	return out
}

func TestSearchFindsTitleAndTagsFirstThenBodiesInPathOrder(t *testing.T) {
	res, err := Search(searchFS(), "content", "golang")
	if err != nil {
		t.Fatal(err)
	}

	want := []string{
		"content/b-title.md", "content/c-tag.md", // title and tag matches
		"content/a-body.md", "content/d-none.md", "content/e-broken.md",
		"content/j-unicode.md", "content/k-multiline.md", "content/sub/g-deep.md",
	}
	if got := paths(res); !equal(got, want) {
		t.Errorf("hits = %v, want %v", got, want)
	}
	if res.Truncated {
		t.Error("Truncated = true, want false")
	}
}

func TestSearchIgnoresCaseAndTrimsTheQuery(t *testing.T) {
	res, err := Search(searchFS(), "content", "  LEARNING gol")
	if err != nil {
		t.Fatal(err)
	}
	if got, want := paths(res), []string{"content/b-title.md"}; !equal(got, want) {
		t.Errorf("hits = %v, want %v", got, want)
	}
}

func TestSearchTreatsTheQueryAsLiteralText(t *testing.T) {
	fsys := fstest.MapFS{"content/a.md": {Data: []byte("price is $5 (approx.)\nabc\n")}}

	for _, q := range []string{"(approx.)", "$5", ".*"} {
		res, err := Search(fsys, "content", q)
		if err != nil {
			t.Fatalf("%q: %v", q, err)
		}
		wantHits := 1
		if q == ".*" {
			wantHits = 0
		}
		if len(res.Hits) != wantHits {
			t.Errorf("%q: hits = %d, want %d", q, len(res.Hits), wantHits)
		}
	}
}

func TestSearchHitDetails(t *testing.T) {
	res, err := Search(searchFS(), "content", "golang")
	if err != nil {
		t.Fatal(err)
	}
	byPath := map[string]Hit{}
	for _, h := range res.Hits {
		byPath[h.Path] = h
	}

	if h := byPath["content/b-title.md"]; h.Title != "Learning Golang" || !h.Draft || h.Snippet != "" {
		t.Errorf("title hit = %+v, want title, draft and no snippet (the body has no match)", h)
	}
	if h := byPath["content/a-body.md"]; h.Title != "Alpha" || h.Snippet != "Some text about GOLANG here." {
		t.Errorf("body hit = %+v", h)
	}
	if h := byPath["content/k-multiline.md"]; h.Snippet != "first line golang last line" {
		t.Errorf("snippet = %q, want whitespace collapsed", h.Snippet)
	}
}

func TestSearchDoesNotMatchFrontmatterThatIsNotTitleOrTags(t *testing.T) {
	res, err := Search(searchFS(), "content", "golang")
	if err != nil {
		t.Fatal(err)
	}
	for _, p := range paths(res) {
		if p == "content/i-fm-only.md" {
			t.Error("a match inside other frontmatter settings must not count")
		}
	}
}

func TestSearchSnippetKeepsRunesWhole(t *testing.T) {
	res, err := Search(searchFS(), "content", "golang")
	if err != nil {
		t.Fatal(err)
	}
	for _, h := range res.Hits {
		if h.Path == "content/j-unicode.md" && !strings.Contains(h.Snippet, "golang") {
			t.Errorf("snippet = %q", h.Snippet)
		}
		for _, r := range h.Snippet {
			if r == '�' {
				t.Errorf("snippet %q contains a broken rune", h.Snippet)
			}
		}
	}

	// A long body forces cuts on both sides of the match.
	long := strings.Repeat("ä", 100) + "needle" + strings.Repeat("ö", 100)
	res, err = Search(fstest.MapFS{"content/l.md": {Data: []byte(long)}}, "content", "needle")
	if err != nil {
		t.Fatal(err)
	}
	s := res.Hits[0].Snippet
	if !strings.HasPrefix(s, "…") || !strings.HasSuffix(s, "…") || !strings.Contains(s, "needle") {
		t.Errorf("snippet = %q, want ellipses around the match", s)
	}
	if strings.ContainsRune(s, '�') {
		t.Errorf("snippet %q contains a broken rune", s)
	}
}

func TestSearchTruncates(t *testing.T) {
	fsys := fstest.MapFS{}
	for i := range SearchLimit + 5 {
		fsys["content/p"+string(rune('a'+i/26))+string(rune('a'+i%26))+".md"] = &fstest.MapFile{Data: []byte("needle")}
	}

	res, err := Search(fsys, "content", "needle")
	if err != nil {
		t.Fatal(err)
	}
	if len(res.Hits) != SearchLimit || !res.Truncated {
		t.Errorf("hits = %d, truncated = %v, want %d and true", len(res.Hits), res.Truncated, SearchLimit)
	}
}

func TestSearchEmptyQuery(t *testing.T) {
	for _, q := range []string{"", "   "} {
		if _, err := Search(searchFS(), "content", q); !errors.Is(err, ErrEmptyQuery) {
			t.Errorf("%q: err = %v, want ErrEmptyQuery", q, err)
		}
	}
}

func TestSearchMissingContentDirFindsNothing(t *testing.T) {
	res, err := Search(fstest.MapFS{}, "content", "x")
	if err != nil {
		t.Fatal(err)
	}
	if len(res.Hits) != 0 || res.Hits == nil {
		t.Errorf("hits = %#v, want an empty non-nil slice (it must encode as [])", res.Hits)
	}
}
