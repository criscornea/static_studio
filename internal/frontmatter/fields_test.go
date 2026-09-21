package frontmatter

import (
	"errors"
	"slices"
	"testing"
	"time"
)

func date(y int, m time.Month, d, hh, mm int, loc *time.Location) *time.Time {
	t := time.Date(y, m, d, hh, mm, 0, 0, loc)
	return &t
}

func TestFields(t *testing.T) {
	plusOne := time.FixedZone("", 60*60)

	tests := []struct {
		name      string
		in        string
		wantTitle string
		wantDate  *time.Time
		wantDraft bool
		wantTags  []string
		wantErr   error
	}{
		{
			name: "yaml, all fields",
			in: "---\ntitle: First Post\ndate: 2026-02-01T09:30:00+01:00\n" +
				"draft: true\ntags: [go, hugo]\n---\nBody\n",
			wantTitle: "First Post",
			wantDate:  date(2026, 2, 1, 9, 30, plusOne),
			wantDraft: true,
			wantTags:  []string{"go", "hugo"},
		},
		{
			name: "toml, all fields",
			in: "+++\ntitle = 'Second Post'\ndate = 2026-03-12T14:00:00+01:00\n" +
				"draft = true\ntags = ['go']\n+++\nBody\n",
			wantTitle: "Second Post",
			wantDate:  date(2026, 3, 12, 14, 0, plusOne),
			wantDraft: true,
			wantTags:  []string{"go"},
		},

		// The date matrix: every shape a date can arrive in.
		{
			name:     "yaml date only",
			in:       "---\ndate: 2026-02-01\n---\n",
			wantDate: date(2026, 2, 1, 0, 0, time.UTC),
		},
		{
			name:     "yaml quoted date string",
			in:       "---\ndate: \"2026-02-01T09:30:00+01:00\"\n---\n",
			wantDate: date(2026, 2, 1, 9, 30, plusOne),
		},
		{
			name:     "toml local date",
			in:       "+++\ndate = 2026-02-01\n+++\n",
			wantDate: date(2026, 2, 1, 0, 0, time.UTC),
		},
		{
			name:     "toml local datetime",
			in:       "+++\ndate = 2026-02-01T09:30:00\n+++\n",
			wantDate: date(2026, 2, 1, 9, 30, time.UTC),
		},
		{
			name: "unparseable date is ignored",
			in:   "---\ndate: next tuesday\n---\n",
		},

		// Tolerance: odd input yields zero values, never an error.
		{
			name:      "capitalised keys",
			in:        "---\nTitle: Old Style\nDraft: true\n---\n",
			wantTitle: "Old Style",
			wantDraft: true,
		},
		{
			name: "mistyped fields fall back to zero values",
			in:   "---\ntitle: 42\ndraft: \"yes\"\ntags: go\n---\n",
		},
		{
			name:     "non-string tags are dropped",
			in:       "---\ntags: [go, 42, hugo]\n---\n",
			wantTags: []string{"go", "hugo"},
		},
		{
			name: "no frontmatter",
			in:   "# Just a heading\n",
		},
		{
			name: "empty yaml block",
			in:   "---\n---\nBody\n",
		},

		// Broken syntax is an error.
		{
			name:    "invalid yaml",
			in:      "---\ntitle: [unclosed\n---\n",
			wantErr: ErrInvalid,
		},
		{
			name:    "invalid toml",
			in:      "+++\ntitle = \n+++\n",
			wantErr: ErrInvalid,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			doc, err := Split([]byte(tt.in))
			if err != nil {
				t.Fatalf("Split: %v", err)
			}

			got, err := doc.Fields()

			if tt.wantErr != nil {
				if !errors.Is(err, tt.wantErr) {
					t.Fatalf("err = %v, want %v", err, tt.wantErr)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if got.Title != tt.wantTitle {
				t.Errorf("Title = %q, want %q", got.Title, tt.wantTitle)
			}
			if got.Draft != tt.wantDraft {
				t.Errorf("Draft = %v, want %v", got.Draft, tt.wantDraft)
			}
			if !slices.Equal(got.Tags, tt.wantTags) {
				t.Errorf("Tags = %q, want %q", got.Tags, tt.wantTags)
			}

			switch {
			case tt.wantDate == nil && got.Date != nil:
				t.Errorf("Date = %v, want none", *got.Date)
			case tt.wantDate != nil && got.Date == nil:
				t.Errorf("Date = none, want %v", *tt.wantDate)
			case tt.wantDate != nil && !got.Date.Equal(*tt.wantDate):
				t.Errorf("Date = %v, want %v", *got.Date, *tt.wantDate)
			}
		})
	}
}
