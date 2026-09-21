package frontmatter

import (
	"errors"
	"testing"
)

func TestSplit(t *testing.T) {
	tests := []struct {
		name       string
		in         string
		wantFormat Format
		wantMeta   string
		wantBody   string
		wantErr    error
	}{
		{
			name:       "yaml",
			in:         "---\ntitle: Hello\n---\nBody text.\n",
			wantFormat: FormatYAML,
			wantMeta:   "title: Hello\n",
			wantBody:   "Body text.\n",
		},
		{
			name:       "toml",
			in:         "+++\ntitle = 'Hello'\n+++\nBody text.\n",
			wantFormat: FormatTOML,
			wantMeta:   "title = 'Hello'\n",
			wantBody:   "Body text.\n",
		},
		{
			name:       "no frontmatter",
			in:         "# Just markdown\n\nNo metadata here.\n",
			wantFormat: FormatNone,
			wantBody:   "# Just markdown\n\nNo metadata here.\n",
		},
		{
			name:       "empty file",
			in:         "",
			wantFormat: FormatNone,
		},
		{
			name:       "empty frontmatter block",
			in:         "---\n---\nBody.\n",
			wantFormat: FormatYAML,
			wantMeta:   "",
			wantBody:   "Body.\n",
		},
		{
			name:       "horizontal rule in body is not a fence",
			in:         "---\ntitle: x\n---\nAbove\n\n---\n\nBelow\n",
			wantFormat: FormatYAML,
			wantMeta:   "title: x\n",
			wantBody:   "Above\n\n---\n\nBelow\n",
		},
		{
			name:       "closing fence at end of file without newline",
			in:         "---\ntitle: x\n---",
			wantFormat: FormatYAML,
			wantMeta:   "title: x\n",
			wantBody:   "",
		},
		{
			name:       "windows line endings are matched but preserved",
			in:         "---\r\ntitle: x\r\n---\r\nBody.\r\n",
			wantFormat: FormatYAML,
			wantMeta:   "title: x\r\n",
			wantBody:   "Body.\r\n",
		},
		{
			name:       "byte order mark is skipped",
			in:         "\xEF\xBB\xBF---\ntitle: x\n---\nBody.\n",
			wantFormat: FormatYAML,
			wantMeta:   "title: x\n",
			wantBody:   "Body.\n",
		},
		{
			name:       "fence must be the first line",
			in:         "\n---\ntitle: x\n---\n",
			wantFormat: FormatNone,
			wantBody:   "\n---\ntitle: x\n---\n",
		},
		{
			name:       "longer dash line is not a fence",
			in:         "----\ntitle: x\n----\n",
			wantFormat: FormatNone,
			wantBody:   "----\ntitle: x\n----\n",
		},
		{
			name:       "fence with trailing spaces is not a fence",
			in:         "--- \ntitle: x\n---\n",
			wantFormat: FormatNone,
			wantBody:   "--- \ntitle: x\n---\n",
		},
		{
			name:    "unterminated yaml",
			in:      "---\ntitle: x\nBody without a closing fence.\n",
			wantErr: ErrUnterminated,
		},
		{
			name:    "unterminated toml",
			in:      "+++\ntitle = 'x'\n",
			wantErr: ErrUnterminated,
		},
		{
			name:    "mismatched fences",
			in:      "---\ntitle: x\n+++\nBody.\n",
			wantErr: ErrUnterminated,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			doc, err := Split([]byte(tt.in))

			if tt.wantErr != nil {
				if !errors.Is(err, tt.wantErr) {
					t.Fatalf("err = %v, want %v", err, tt.wantErr)
				}
				if doc != nil {
					t.Errorf("doc = %+v, want nil on error", doc)
				}
				return
			}

			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if doc.Format != tt.wantFormat {
				t.Errorf("Format = %q, want %q", doc.Format, tt.wantFormat)
			}
			if got := string(doc.Meta); got != tt.wantMeta {
				t.Errorf("Meta = %q, want %q", got, tt.wantMeta)
			}
			if got := string(doc.Body); got != tt.wantBody {
				t.Errorf("Body = %q, want %q", got, tt.wantBody)
			}
		})
	}
}
