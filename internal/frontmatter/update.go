package frontmatter

import (
	"fmt"
	"strings"

	"github.com/goccy/go-yaml"
	"github.com/pelletier/go-toml/v2"
)

// Update returns the frontmatter block (without fences) with the known
// fields set from f. Keys the editor does not know are kept with their
// values. Comments and, for TOML, key order are not preserved. A known key
// that is absent and has a zero value in f stays absent; a nil Date or
// empty Tags removes the key.
func (d *Document) Update(f Fields) ([]byte, error) {
	switch d.Format {
	case FormatYAML:
		return d.updateYAML(f)
	case FormatTOML:
		return d.updateTOML(f)
	default:
		return nil, fmt.Errorf("cannot update frontmatter of format %q", d.Format)
	}
}

// knownValues maps each known key to its new value, or nil to remove it.
func knownValues(f Fields) map[string]any {
	v := map[string]any{
		"title": f.Title,
		"draft": f.Draft,
		"tags":  nil,
		"date":  nil,
	}
	if len(f.Tags) > 0 {
		v["tags"] = f.Tags
	}
	if f.Date != nil {
		v["date"] = *f.Date
	}
	return v
}

func (d *Document) updateYAML(f Fields) ([]byte, error) {
	var items yaml.MapSlice
	if err := yaml.Unmarshal(d.Meta, &items); err != nil {
		return nil, fmt.Errorf("%w (%s): %w", ErrInvalid, d.Format, err)
	}

	values := knownValues(f)
	seen := map[string]bool{}
	out := make(yaml.MapSlice, 0, len(items)+4)
	for _, item := range items {
		key, _ := item.Key.(string)
		lower := strings.ToLower(key)
		nv, known := values[lower]
		if !known || seen[lower] {
			out = append(out, item)
			continue
		}
		seen[lower] = true
		if nv != nil {
			out = append(out, yaml.MapItem{Key: item.Key, Value: nv})
		}
	}
	for _, k := range []string{"title", "date", "draft", "tags"} {
		if seen[k] || values[k] == nil {
			continue
		}
		if k == "title" && f.Title == "" || k == "draft" && !f.Draft {
			continue
		}
		out = append(out, yaml.MapItem{Key: k, Value: values[k]})
	}
	if len(out) == 0 {
		return nil, nil
	}
	b, err := yaml.Marshal(out)
	if err != nil {
		return nil, fmt.Errorf("encoding frontmatter: %w", err)
	}
	return b, nil
}

func (d *Document) updateTOML(f Fields) ([]byte, error) {
	raw := map[string]any{}
	if err := toml.Unmarshal(d.Meta, &raw); err != nil {
		return nil, fmt.Errorf("%w (%s): %w", ErrInvalid, d.Format, err)
	}

	values := knownValues(f)
	seen := map[string]bool{}
	for key := range raw {
		lower := strings.ToLower(key)
		nv, known := values[lower]
		if !known || seen[lower] {
			continue
		}
		seen[lower] = true
		if nv == nil {
			delete(raw, key)
		} else {
			raw[key] = nv
		}
	}
	for _, k := range []string{"title", "date", "draft", "tags"} {
		if seen[k] || values[k] == nil {
			continue
		}
		if k == "title" && f.Title == "" || k == "draft" && !f.Draft {
			continue
		}
		raw[k] = values[k]
	}
	if len(raw) == 0 {
		return nil, nil
	}
	b, err := toml.Marshal(raw)
	if err != nil {
		return nil, fmt.Errorf("encoding frontmatter: %w", err)
	}
	return b, nil
}
