package content

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"testing/fstest"
	"time"
)

var fixedNow = time.Date(2024, 3, 1, 10, 30, 0, 0, time.UTC)

// newSite writes files (slash-separated names) into a temp dir and returns
// a confined root on it.
func newSite(t *testing.T, files map[string]string) *os.Root {
	t.Helper()

	dir := t.TempDir()
	for name, body := range files {
		full := filepath.Join(dir, filepath.FromSlash(name))
		if err := os.MkdirAll(filepath.Dir(full), 0o750); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(full, []byte(body), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	root, err := os.OpenRoot(dir)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = root.Close() })
	return root
}

func TestArchetypes(t *testing.T) {
	fsys := fstest.MapFS{
		"archetypes/posts.md":      {},
		"archetypes/Notes.md":      {},
		"archetypes/default.md":    {},
		"archetypes/readme.txt":    {},
		"archetypes/.hidden.md":    {},
		"archetypes/bundle/foo.md": {},
	}

	var got []string
	for _, a := range Archetypes(fsys) {
		if a.Builtin {
			t.Errorf("%q is built in, but the project defines default.md", a.Name)
		}
		got = append(got, a.Name)
	}
	if want := []string{"default", "Notes", "posts"}; !equal(got, want) {
		t.Errorf("archetypes = %v, want %v (default first, folders and other files skipped)", got, want)
	}
}

func TestArchetypesFallsBackToTheBuiltinDefault(t *testing.T) {
	for name, fsys := range map[string]fstest.MapFS{
		"no folder":     {},
		"no default.md": {"archetypes/posts.md": {}},
	} {
		got := Archetypes(fsys)
		if len(got) == 0 || got[0].Name != "default" || !got[0].Builtin {
			t.Errorf("%s: archetypes = %+v, want the builtin default first", name, got)
		}
	}
}

func TestSlugify(t *testing.T) {
	tests := map[string]string{
		"Hello, World!":          "hello-world",
		"  spaced   out  ":       "spaced-out",
		"Über die Größe":         "über-die-größe",
		"2024: A Year":           "2024-a-year",
		"a/b\\c..d":              "a-b-c-d",
		"---":                    "",
		"":                       "",
		"!?":                     "",
		"UPPER":                  "upper",
		strings.Repeat("a", 200): strings.Repeat("a", maxSlugLength),
	}
	for in, want := range tests {
		if got := Slugify(in); got != want {
			t.Errorf("Slugify(%q) = %q, want %q", in, got, want)
		}
	}
}

func TestSlugifyCapsTheLengthWithoutATrailingHyphen(t *testing.T) {
	got := Slugify(strings.Repeat("ab ", 60))

	if len(got) > maxSlugLength || strings.HasSuffix(got, "-") || strings.HasPrefix(got, "-") {
		t.Errorf("Slugify = %q (%d), want at most %d characters and no edge hyphens", got, len(got), maxSlugLength)
	}
}

func TestCreatePageFromTheBuiltinDefault(t *testing.T) {
	root := newSite(t, map[string]string{"hugo.toml": ""})

	page, err := CreatePage(root, "content", NewPage{Folder: "posts", Name: "My First Post!"}, fixedNow)
	if err != nil {
		t.Fatal(err)
	}

	if page.Path != "content/posts/my-first-post.md" {
		t.Errorf("path = %q", page.Path)
	}
	if page.Fields.Title != "My First Post" || !page.Fields.Draft {
		t.Errorf("fields = %+v, want title \"My First Post\" and draft", page.Fields)
	}
	if page.Fields.Date == nil || !page.Fields.Date.Equal(fixedNow) {
		t.Errorf("date = %v, want %v", page.Fields.Date, fixedNow)
	}
	if _, err := root.Stat("content/posts/my-first-post.md"); err != nil {
		t.Errorf("file not on disk: %v", err)
	}
}

func TestCreatePageUsesTheProjectsArchetypes(t *testing.T) {
	root := newSite(t, map[string]string{
		"archetypes/default.md": "---\ntitle: \"{{ .Name }}\"\ntype: {{ .Type }}\n---\ndefault body\n",
		"archetypes/posts.md": "---\n" +
			"title: \"{{ replace .File.ContentBaseName \"-\" \" \" | title }}\"\n" +
			"date: {{ .Date }}\n" +
			"stamp: {{ now.Format \"2006\" }}\n" +
			"tags: [{{ upper .Type }}]\n" +
			"---\nposts body\n",
	})

	// Folder named after an archetype picks it when none is requested.
	page, err := CreatePage(root, "content", NewPage{Folder: "posts/2024", Name: "hello world"}, fixedNow)
	if err != nil {
		t.Fatal(err)
	}
	if page.Fields.Title != "Hello World" || page.Body != "posts body\n" {
		t.Errorf("page = %+v, want the posts archetype", page)
	}
	if len(page.Fields.Tags) != 1 || page.Fields.Tags[0] != "POSTS" {
		t.Errorf("tags = %v, want [POSTS]", page.Fields.Tags)
	}

	// Elsewhere the default is used, and .Type is the folder's section.
	page, err = CreatePage(root, "content", NewPage{Folder: "docs", Name: "intro"}, fixedNow)
	if err != nil {
		t.Fatal(err)
	}
	if page.Body != "default body\n" || page.Fields.Title != "intro" {
		t.Errorf("page = %+v, want the default archetype", page)
	}

	// An explicit choice wins over the folder.
	page, err = CreatePage(root, "content", NewPage{Archetype: "posts", Name: "at the root"}, fixedNow)
	if err != nil {
		t.Fatal(err)
	}
	if page.Path != "content/at-the-root.md" || page.Body != "posts body\n" {
		t.Errorf("page = %+v, want posts archetype at the content root", page)
	}
}

func TestCreatePageNeverOverwrites(t *testing.T) {
	root := newSite(t, map[string]string{"content/a.md": "precious"})

	_, err := CreatePage(root, "content", NewPage{Name: "A"}, fixedNow)

	if !errors.Is(err, ErrPageExists) {
		t.Fatalf("err = %v, want ErrPageExists", err)
	}
	if data, _ := root.ReadFile("content/a.md"); string(data) != "precious" {
		t.Errorf("existing file was changed to %q", data)
	}
}

func TestCreatePageErrors(t *testing.T) {
	root := newSite(t, map[string]string{
		"archetypes/unsupported.md": "---\ntitle: {{ .Nope }}\n---\n",
		"archetypes/badfunc.md":     "---\ntitle: {{ readFile \"x\" }}\n---\n",
		"archetypes/broken.md":      "---\ntitle: [unclosed\n---\n",
		"archetypes/syntax.md":      "{{ if }}",
	})

	tests := []struct {
		name string
		req  NewPage
		want error
	}{
		{"no usable name", NewPage{Name: " !? "}, ErrInvalidName},
		{"parent folder", NewPage{Folder: "../out", Name: "x"}, ErrInvalidFolder},
		{"parent inside", NewPage{Folder: "a/../../out", Name: "x"}, ErrInvalidFolder},
		{"hidden folder", NewPage{Folder: ".git", Name: "x"}, ErrInvalidFolder},
		{"hidden nested", NewPage{Folder: "a/.b", Name: "x"}, ErrInvalidFolder},
		{"node_modules", NewPage{Folder: "node_modules", Name: "x"}, ErrInvalidFolder},
		{"unknown archetype", NewPage{Archetype: "nope", Name: "x"}, ErrUnknownArchetype},
		{"path as archetype", NewPage{Archetype: "../hugo", Name: "x"}, ErrUnknownArchetype},
		{"unknown field", NewPage{Archetype: "unsupported", Name: "x"}, ErrUnsupportedArchetype},
		{"unknown function", NewPage{Archetype: "badfunc", Name: "x"}, ErrUnsupportedArchetype},
		{"template syntax", NewPage{Archetype: "syntax", Name: "x"}, ErrUnsupportedArchetype},
		{"broken frontmatter", NewPage{Archetype: "broken", Name: "x"}, ErrInvalidArchetype},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if _, err := CreatePage(root, "content", tt.req, fixedNow); !errors.Is(err, tt.want) {
				t.Errorf("err = %v, want %v", err, tt.want)
			}
		})
	}

	// None of the failures may leave anything behind.
	if _, err := root.Stat("content"); err == nil {
		t.Error("a failed request created the content folder")
	}
}

func TestCreatePageStaysInsideTheProject(t *testing.T) {
	outside := t.TempDir()
	root := newSite(t, map[string]string{"hugo.toml": ""})
	if err := os.Symlink(outside, filepath.Join(root.Name(), "content")); err != nil {
		t.Skipf("symlinks unavailable: %v", err)
	}

	if _, err := CreatePage(root, "content", NewPage{Name: "escape"}, fixedNow); err == nil {
		t.Fatal("created a page through a symlink that leaves the project")
	}
	if entries, _ := os.ReadDir(outside); len(entries) != 0 {
		t.Errorf("wrote outside the project: %v", entries)
	}
}
