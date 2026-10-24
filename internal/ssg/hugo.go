package ssg

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"os"
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

// hugo reads its environment through env, so tests need not touch the process
// environment. A nil env means os.Getenv.
type hugo struct {
	env func(string) string
}

func (hugo) Kind() Kind      { return KindHugo }
func (hugo) Supported() bool { return true }

// Dirs reads contentDir and staticDir from the merged Hugo config: the root
// config file, then config/_default/*, then config/<environment>/*, then
// HUGO_CONTENTDIR and HUGO_STATICDIR. Per-language overrides and module
// mounts are not followed.
func (h hugo) Dirs(fsys fs.FS, configFile string) (string, string, error) {
	raw, err := readConfig(fsys, configFile)
	if err != nil {
		return "", "", err
	}
	raw = mergeConfig(nil, raw)

	for _, dir := range []string{"config/_default", "config/" + h.environment()} {
		if err := mergeConfigDir(fsys, dir, raw); err != nil {
			return "", "", err
		}
	}

	if v := strings.TrimSpace(h.getenv("HUGO_CONTENTDIR")); v != "" {
		raw["contentdir"] = v
	}
	if v := strings.TrimSpace(h.getenv("HUGO_STATICDIR")); v != "" {
		// Hugo splits list values from the environment on whitespace.
		raw["staticdir"] = strings.Fields(v)[0]
	}

	contentDir := configDir(raw["contentdir"], hugoDefaultContentDir)
	// staticDir may be a list of directories. The first one is the primary.
	assetDir := configDir(firstOf(raw["staticdir"]), hugoDefaultAssetDir)

	return contentDir, assetDir, nil
}

// getenv reads the injected environment, or the process environment when none
// was injected.
func (h hugo) getenv(key string) string {
	if h.env != nil {
		return h.env(key)
	}
	return os.Getenv(key)
}

// environment is the Hugo environment whose config directory is overlaid:
// HUGO_ENVIRONMENT, else "development", the dev server's default.
func (h hugo) environment() string {
	if e := strings.TrimSpace(h.getenv("HUGO_ENVIRONMENT")); e != "" {
		return e
	}
	return "development"
}

// mergeConfigDir merges every config file in dir into raw, in file name
// order. A file named hugo or config holds top-level keys; any other file
// nests under the key named after it (languages.toml under "languages"). A
// missing directory is fine; an invalid file is a *ConfigError.
func mergeConfigDir(fsys fs.FS, dir string, raw map[string]any) error {
	if !fs.ValidPath(dir) {
		return nil
	}
	entries, err := fs.ReadDir(fsys, dir)
	if errors.Is(err, fs.ErrNotExist) {
		return nil
	}
	if err != nil {
		return &ConfigError{File: dir, Reason: "the directory cannot be read (" + reasonOf(err) + ")"}
	}
	for _, e := range entries {
		ext := strings.ToLower(path.Ext(e.Name()))
		if e.IsDir() || (ext != ".toml" && ext != ".yaml" && ext != ".yml" && ext != ".json") {
			continue
		}
		part, err := readConfig(fsys, dir+"/"+e.Name())
		if err != nil {
			return err
		}
		key := strings.ToLower(strings.TrimSuffix(e.Name(), path.Ext(e.Name())))
		if key == "hugo" || key == "config" {
			mergeConfig(raw, part)
		} else {
			mergeConfig(raw, map[string]any{key: part})
		}
	}
	return nil
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
