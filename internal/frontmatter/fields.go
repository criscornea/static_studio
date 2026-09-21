package frontmatter

import (
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/goccy/go-yaml"
	"github.com/pelletier/go-toml/v2"
)

var ErrInvalid = errors.New("frontmatter is not valid")

// Fields are the frontmatter values the editor understands. Everything
// else stays untouched in Document.Meta.
type Fields struct {
	Title string     `json:"title"`
	Date  *time.Time `json:"date,omitempty"`
	Draft bool       `json:"draft"`
	Tags  []string   `json:"tags,omitempty"`
}

// Fields extracts the known values from the frontmatter block. A field
// that is missing or has an unexpected type is left at its zero value.
func (d *Document) Fields() (*Fields, error) {
	raw, err := d.decode()
	if err != nil {
		return nil, err
	}

	f := &Fields{}
	if s, ok := raw["title"].(string); ok {
		f.Title = s
	}
	if b, ok := raw["draft"].(bool); ok {
		f.Draft = b
	}
	f.Tags = stringList(raw["tags"])
	if t, ok := asTime(raw["date"]); ok {
		f.Date = &t
	}
	return f, nil
}

func (d *Document) decode() (map[string]any, error) {
	raw := map[string]any{}

	var err error
	switch d.Format {
	case FormatNone:
		return raw, nil
	case FormatYAML:
		err = yaml.Unmarshal(d.Meta, &raw)
	case FormatTOML:
		err = toml.Unmarshal(d.Meta, &raw)
	default:
		return nil, fmt.Errorf("unknown frontmatter format %q", d.Format)
	}
	if err != nil {
		return nil, fmt.Errorf("%w (%s): %w", ErrInvalid, d.Format, err)
	}

	out := make(map[string]any, len(raw))
	for k, v := range raw {
		out[strings.ToLower(k)] = v
	}
	return out, nil
}

func stringList(v any) []string {
	items, ok := v.([]any)
	if !ok {
		return nil
	}
	out := make([]string, 0, len(items))
	for _, item := range items {
		if s, ok := item.(string); ok {
			out = append(out, s)
		}
	}
	return out
}

var dateLayouts = []string{time.RFC3339, "2006-01-02T15:04:05", "2006-01-02"}

// asTime accepts every shape a date can arrive in from either parser.
func asTime(v any) (time.Time, bool) {
	switch t := v.(type) {
	case time.Time:
		return t, true
	case toml.LocalDateTime:
		return t.AsTime(time.UTC), true
	case toml.LocalDate:
		return t.AsTime(time.UTC), true
	case string:
		for _, layout := range dateLayouts {
			if parsed, err := time.Parse(layout, t); err == nil {
				return parsed, true
			}
		}
	}
	return time.Time{}, false
}
