package frontmatter

import (
	"slices"
	"strings"
	"testing"
	"time"
)

func TestUpdateRoundTrip(t *testing.T) {
	when := time.Date(2027, 1, 2, 3, 4, 5, 0, time.UTC)
	tests := []struct {
		name string
		in   string
	}{
		{"yaml", "---\ntitle: Old\nweight: 4\nauthor:\n  name: Ann\n  links: [a, b]\ndraft: false\n---\nBody\n"},
		{"toml", "+++\ntitle = 'Old'\nweight = 4\ndraft = false\n[author]\nname = 'Ann'\n+++\nBody\n"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			doc, err := Split([]byte(tt.in))
			if err != nil {
				t.Fatal(err)
			}
			want := Fields{Title: "New: \"title\"", Date: &when, Draft: true, Tags: []string{"x", "y z"}}
			meta, err := doc.Update(want)
			if err != nil {
				t.Fatal(err)
			}
			fence := "---\n"
			if doc.Format == FormatTOML {
				fence = "+++\n"
			}
			re, err := Split([]byte(fence + string(meta) + fence + "Body\n"))
			if err != nil {
				t.Fatal(err)
			}
			got, err := re.Fields()
			if err != nil {
				t.Fatal(err)
			}
			if got.Title != want.Title || !got.Draft || !slices.Equal(got.Tags, want.Tags) ||
				got.Date == nil || !got.Date.Equal(when) {
				t.Fatalf("fields: %+v", got)
			}
			s := string(meta)
			for _, k := range []string{"weight", "Ann"} {
				if !strings.Contains(s, k) {
					t.Errorf("unknown key lost (%s): %s", k, s)
				}
			}
			if string(re.Body) != "Body\n" {
				t.Fatal("body changed")
			}
		})
	}
}

func TestUpdateRemovesAndSkips(t *testing.T) {
	doc, _ := Split([]byte("---\ntitle: T\ndate: 2026-01-01\ntags: [a]\nkeep: 1\n---\n"))
	meta, err := doc.Update(Fields{Title: "T"})
	if err != nil {
		t.Fatal(err)
	}
	s := string(meta)
	if strings.Contains(s, "date") || strings.Contains(s, "tags") || strings.Contains(s, "draft") || !strings.Contains(s, "keep") {
		t.Fatalf("got %q", s)
	}
	bad, _ := Split([]byte("---\n---\n"))
	if _, err := bad.Update(Fields{Title: "x"}); err != nil {
		t.Fatal(err)
	}
	none, _ := Split([]byte("x"))
	if _, err := none.Update(Fields{}); err == nil {
		t.Fatal("want error for no frontmatter")
	}
}
