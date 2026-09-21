package content

import (
	"bytes"
	"errors"
	"testing"
	"testing/fstest"

	"github.com/criscornea/static_studio/internal/frontmatter"
)

func pageFS() fstest.MapFS {
	return fstest.MapFS{
		"hugo.toml": {Data: []byte("title = 'site'\n")},

		"content/yaml.md":     {Data: []byte("---\ntitle: YAML Page\n---\nYAML body.\n")},
		"content/toml.md":     {Data: []byte("+++\ntitle = 'TOML Page'\n+++\nTOML body.\n")},
		"content/plain.md":    {Data: []byte("# No frontmatter\n")},
		"content/Loud.MD":     {Data: []byte("---\ntitle: Loud\n---\n")},
		"content/broken.md":   {Data: []byte("---\ntitle: [unclosed\n---\n")},
		"content/unclosed.md": {Data: []byte("---\ntitle: x\nno closing fence\n")},
		"content/image.png":   {Data: []byte("\x89PNG")},
		"content/posts/a.md":  {Data: []byte("---\ntitle: Nested\n---\n")},

		"content/exact.md": {Data: bytes.Repeat([]byte("a"), maxPageSize)},
		"content/huge.md":  {Data: bytes.Repeat([]byte("a"), maxPageSize+1)},

		// Exists, so a rejection proves the path check fired, not a missing file.
		"contentfoo/x.md": {Data: []byte("---\ntitle: sneaky\n---\n")},
		"static/x.md":     {Data: []byte("---\ntitle: static\n---\n")},
	}
}

func TestReadPage(t *testing.T) {
	tests := []struct {
		name       string
		path       string
		wantFormat frontmatter.Format
		wantTitle  string
		wantBody   string
		wantErr    error
	}{
		{name: "yaml page", path: "content/yaml.md",
			wantFormat: frontmatter.FormatYAML, wantTitle: "YAML Page", wantBody: "YAML body.\n"},
		{name: "toml page", path: "content/toml.md",
			wantFormat: frontmatter.FormatTOML, wantTitle: "TOML Page", wantBody: "TOML body.\n"},
		{name: "no frontmatter", path: "content/plain.md",
			wantFormat: frontmatter.FormatNone, wantBody: "# No frontmatter\n"},
		{name: "nested page", path: "content/posts/a.md",
			wantFormat: frontmatter.FormatYAML, wantTitle: "Nested"},
		{name: "upper-case extension", path: "content/Loud.MD",
			wantFormat: frontmatter.FormatYAML, wantTitle: "Loud"},
		{name: "exactly at the size limit", path: "content/exact.md",
			wantFormat: frontmatter.FormatNone, wantBody: string(bytes.Repeat([]byte("a"), maxPageSize))},

		{name: "missing file", path: "content/nope.md", wantErr: ErrNotFound},
		{name: "over the size limit", path: "content/huge.md", wantErr: ErrTooLarge},

		{name: "directory", path: "content/posts", wantErr: ErrNotEditable},
		{name: "non-content extension", path: "content/image.png", wantErr: ErrNotEditable},
		{name: "config file", path: "hugo.toml", wantErr: ErrNotEditable},
		{name: "outside content dir", path: "static/x.md", wantErr: ErrNotEditable},
		{name: "prefix without separator", path: "contentfoo/x.md", wantErr: ErrNotEditable},
		{name: "parent traversal", path: "content/../static/x.md", wantErr: ErrNotEditable},
		{name: "dot segment", path: "content/./yaml.md", wantErr: ErrNotEditable},
		{name: "absolute path", path: "/content/yaml.md", wantErr: ErrNotEditable},
		{name: "empty path", path: "", wantErr: ErrNotEditable},

		{name: "invalid frontmatter", path: "content/broken.md", wantErr: frontmatter.ErrInvalid},
		{name: "unclosed frontmatter", path: "content/unclosed.md", wantErr: frontmatter.ErrInvalid},
	}

	fsys := pageFS()

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ReadPage(fsys, "content", tt.path)

			if tt.wantErr != nil {
				if !errors.Is(err, tt.wantErr) {
					t.Fatalf("err = %v, want %v", err, tt.wantErr)
				}
				if got != nil {
					t.Errorf("page = %+v, want nil on error", got)
				}
				return
			}

			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if got.Path != tt.path {
				t.Errorf("Path = %q, want %q", got.Path, tt.path)
			}
			if got.Format != tt.wantFormat {
				t.Errorf("Format = %q, want %q", got.Format, tt.wantFormat)
			}
			if got.Fields == nil {
				t.Fatal("Fields is nil")
			}
			if got.Fields.Title != tt.wantTitle {
				t.Errorf("Title = %q, want %q", got.Fields.Title, tt.wantTitle)
			}
			if tt.wantBody != "" && got.Body != tt.wantBody {
				t.Errorf("Body = %q, want %q", truncate(got.Body), truncate(tt.wantBody))
			}
		})
	}
}

// truncate keeps failure output readable for the large-file cases.
func truncate(s string) string {
	if len(s) > 60 {
		return s[:60] + "…"
	}
	return s
}
