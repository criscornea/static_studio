package ssg

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"path"
	"strings"

	"github.com/goccy/go-yaml"
	"github.com/pelletier/go-toml/v2"
)

const (
	hugoDefaultContentDir = "content"
	hugoDefaultAssetDir   = "static"

	// maxConfigSize caps how much of a config file is read into memory.
	maxConfigSize = 1 << 20 // 1 MiB
)

type hugo struct{}

func (hugo) Kind() Kind      { return KindHugo }
func (hugo) Supported() bool { return true }

// Dirs reads contentDir and staticDir from the top level of the Hugo config.
// Per-language overrides and module mounts are not followed.
func (hugo) Dirs(fsys fs.FS, configFile string) (string, string, error) {
	raw, err := readConfig(fsys, configFile)
	if err != nil {
		return "", "", err
	}

	contentDir := configDir(raw["contentdir"], hugoDefaultContentDir)
	// staticDir may be a list of directories. The first one is the primary.
	assetDir := configDir(firstOf(raw["staticdir"]), hugoDefaultAssetDir)

	return contentDir, assetDir, nil
}

// hugoConfigs are checked in order. Hugo itself accepts all of these;
// the newer hugo.* names take precedence over the legacy config.* names.
var hugoConfigs = []string{
	"hugo.toml", "hugo.yaml", "hugo.yml", "hugo.json",
	"config.toml", "config.yaml", "config.yml", "config.json",
	"config/_default/hugo.toml", "config/_default/hugo.yaml",
	"config/_default/config.toml", "config/_default/config.yaml",
}

func (hugo) Detect(fsys fs.FS) (string, bool) {
	for _, name := range hugoConfigs {
		if info, err := fs.Stat(fsys, name); err == nil && !info.IsDir() {
			return name, true
		}
	}
	return "", false
}

// readConfig decodes a config file into a map with lower-cased top-level
// keys, because Hugo treats config keys case-insensitively. A missing file
// yields an empty map and no error: the config is optional. A file that
// exists but cannot be read, is too large or does not parse yields a
// *ConfigError.
func readConfig(fsys fs.FS, name string) (map[string]any, error) {
	info, err := fs.Stat(fsys, name)
	if errors.Is(err, fs.ErrNotExist) {
		return nil, nil
	}
	if err != nil {
		return nil, &ConfigError{File: name, Reason: "the file cannot be read (" + reasonOf(err) + ")"}
	}
	if info.Size() > maxConfigSize {
		return nil, &ConfigError{File: name, Reason: fmt.Sprintf("the file is larger than %d MiB", maxConfigSize>>20)}
	}
	data, err := fs.ReadFile(fsys, name)
	if errors.Is(err, fs.ErrNotExist) {
		return nil, nil
	}
	if err != nil {
		return nil, &ConfigError{File: name, Reason: "the file cannot be read (" + reasonOf(err) + ")"}
	}

	var raw map[string]any
	switch strings.ToLower(path.Ext(name)) {
	case ".toml":
		err = toml.Unmarshal(data, &raw)
		var de *toml.DecodeError
		if errors.As(err, &de) {
			row, _ := de.Position()
			return nil, &ConfigError{File: name, Reason: fmt.Sprintf("line %d: %s", row, de.Error())}
		}
	case ".yaml", ".yml":
		err = yaml.Unmarshal(data, &raw)
	case ".json":
		err = json.Unmarshal(data, &raw)
		var se *json.SyntaxError
		if errors.As(err, &se) {
			line := 1 + bytes.Count(data[:min(int(se.Offset), len(data))], []byte("\n"))
			return nil, &ConfigError{File: name, Reason: fmt.Sprintf("line %d: %s", line, se.Error())}
		}
	default:
		return nil, nil
	}
	if err != nil {
		return nil, &ConfigError{File: name, Reason: reasonOf(err)}
	}

	out := make(map[string]any, len(raw))
	for k, v := range raw {
		out[strings.ToLower(k)] = v
	}
	return out, nil
}

// reasonOf returns the error text without a leading path, which the caller
// reports separately.
func reasonOf(err error) string {
	var pe *fs.PathError
	if errors.As(err, &pe) {
		return pe.Err.Error()
	}
	return err.Error()
}

// firstOf returns v itself, or its first element when v is a list.
func firstOf(v any) any {
	if items, ok := v.([]any); ok {
		if len(items) == 0 {
			return nil
		}
		return items[0]
	}
	return v
}

// configDir turns a configured directory into a clean path relative to the
// project root. Anything that is not a plain string naming a directory
// strictly inside the project (empty, absolute, "." or escaping with "..")
// gives fallback, so a config can never point the editor outside the project.
func configDir(v any, fallback string) string {
	s, ok := v.(string)
	if !ok {
		return fallback
	}
	clean := path.Clean(strings.ReplaceAll(strings.TrimSpace(s), `\`, "/"))
	if clean == "." || !fs.ValidPath(clean) {
		return fallback
	}
	return clean
}
