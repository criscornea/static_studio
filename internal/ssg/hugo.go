package ssg

import (
	"encoding/json"
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
func (hugo) Dirs(fsys fs.FS, configFile string) (string, string) {
	raw := readConfig(fsys, configFile)

	contentDir := configDir(raw["contentdir"], hugoDefaultContentDir)
	// staticDir may be a list of directories. The first one is the primary.
	assetDir := configDir(firstOf(raw["staticdir"]), hugoDefaultAssetDir)

	return contentDir, assetDir
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
// keys, because Hugo treats config keys case-insensitively. A file that is
// missing, too large or not valid yields an empty map: the project still
// opens, with default directories.
func readConfig(fsys fs.FS, name string) map[string]any {
	info, err := fs.Stat(fsys, name)
	if err != nil || info.Size() > maxConfigSize {
		return nil
	}
	data, err := fs.ReadFile(fsys, name)
	if err != nil {
		return nil
	}

	var raw map[string]any
	switch strings.ToLower(path.Ext(name)) {
	case ".toml":
		err = toml.Unmarshal(data, &raw)
	case ".yaml", ".yml":
		err = yaml.Unmarshal(data, &raw)
	case ".json":
		err = json.Unmarshal(data, &raw)
	default:
		return nil
	}
	if err != nil {
		return nil
	}

	out := make(map[string]any, len(raw))
	for k, v := range raw {
		out[strings.ToLower(k)] = v
	}
	return out
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
