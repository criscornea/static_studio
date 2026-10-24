package ssg

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path"
	"sort"
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
// HUGO_CONTENTDIR and HUGO_STATICDIR. The content dir returned is the primary
// one; see Layout for every content dir.
func (h hugo) Dirs(fsys fs.FS, configFile string) (string, string, error) {
	l, err := h.Layout(fsys, configFile)
	if err != nil {
		return "", "", err
	}
	return l.ContentDir, l.AssetDir, nil
}

// Layout resolves the primary content and asset dirs plus every content dir:
// the implicit contentDir (or, when module.mounts has content mounts, those
// mounts instead), then languages.<lang>.contentDir. The first static mount
// sets the asset dir. Sources that are not plain project-relative
// directories are skipped and logged, never followed.
func (h hugo) Layout(fsys fs.FS, configFile string) (*Layout, error) {
	raw, err := readConfig(fsys, configFile)
	if err != nil {
		return nil, err
	}
	raw = mergeConfig(nil, raw)

	for _, dir := range []string{"config/_default", "config/" + h.environment()} {
		if err := mergeConfigDir(fsys, dir, raw); err != nil {
			return nil, err
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

	contentMounts, staticMount, warnings := mountDirs(raw)
	if staticMount != "" {
		assetDir = staticMount
	}

	langs, _ := lowerMap(raw["languages"])
	names := make([]string, 0, len(langs))
	for name := range langs {
		names = append(names, name)
	}
	sort.Strings(names)
	var langDirs []ContentDir
	missingOwnDir := false
	for _, name := range names {
		lm, ok := lowerMap(langs[name])
		if !ok {
			missingOwnDir = true
			continue
		}
		lm = mergeConfig(nil, lm)
		v, set := lm["contentdir"]
		if !set {
			missingOwnDir = true
			continue
		}
		if dir, ok := mountSource(v); ok {
			langDirs = append(langDirs, ContentDir{Path: dir, Lang: strings.ToLower(name)})
		} else {
			missingOwnDir = true
			warnings = append(warnings, fmt.Sprintf("skipping language %s contentDir outside the project: %v", name, v))
		}
	}

	// The default language's dir is primary when it has one.
	defLang := strings.ToLower(strings.TrimSpace(stringOf(raw["defaultcontentlanguage"])))
	if defLang == "" {
		defLang = "en"
	}
	sort.SliceStable(langDirs, func(i, j int) bool {
		return langDirs[i].Lang == defLang && langDirs[j].Lang != defLang
	})

	var dirs []ContentDir
	switch {
	case len(contentMounts) > 0:
		// Per Hugo, any content mount replaces the implicit contentDir mount.
		dirs = append(dirs, contentMounts...)
	case len(langs) == 0 || missingOwnDir:
		// The implicit dir serves every language without a dir of its own.
		dirs = append(dirs, ContentDir{Path: contentDir})
	}
	if len(langDirs) > 0 && langDirs[0].Lang == defLang {
		dirs = append(langDirs[:1:1], dirs...)
		dirs = append(dirs, langDirs[1:]...)
	} else {
		dirs = append(dirs, langDirs...)
	}
	if len(dirs) == 0 {
		dirs = []ContentDir{{Path: contentDir}}
	}
	dirs = dropNested(dedupeDirs(dirs))

	return &Layout{
		ContentDir:  dirs[0].Path,
		AssetDir:    assetDir,
		ContentDirs: dirs,
		Warnings:    warnings,
	}, nil
}

func stringOf(v any) string {
	s, _ := v.(string)
	return s
}

// dropNested removes dirs lying inside another dir of the list, so no page is
// listed twice.
func dropNested(dirs []ContentDir) []ContentDir {
	out := make([]ContentDir, 0, len(dirs))
	for i, d := range dirs {
		nested := false
		for j, o := range dirs {
			if i != j && strings.HasPrefix(d.Path, o.Path+"/") {
				nested = true
				break
			}
		}
		if !nested {
			out = append(out, d)
		}
	}
	return out
}

// mountDirs reads module.mounts. It returns the content mounts in order, the
// source of the first static mount whose target is exactly "static" ("" when
// none) and warnings for skipped entries. Mounts whose source is not a plain
// project-relative directory are skipped. Mounts of imported modules
// (module.imports) are never read.
func mountDirs(raw map[string]any) (content []ContentDir, static string, warnings []string) {
	mod, _ := lowerMap(raw["module"])
	if imports, ok := mod["imports"].([]any); ok && len(imports) > 0 {
		warnings = append(warnings, fmt.Sprintf("ignoring mounts of %d imported modules", len(imports)))
	}
	mounts, _ := mod["mounts"].([]any)
	for _, m := range mounts {
		mm, ok := lowerMap(m)
		if !ok {
			continue
		}
		mm = mergeConfig(nil, mm)
		target, _ := mm["target"].(string)
		target = path.Clean(strings.ReplaceAll(strings.TrimSpace(target), `\`, "/"))
		isContent := target == "content" || strings.HasPrefix(target, "content/")
		isStatic := target == "static" || strings.HasPrefix(target, "static/")
		if !isContent && !isStatic {
			continue
		}
		dir, ok := mountSource(mm["source"])
		if !ok {
			warnings = append(warnings, fmt.Sprintf("skipping mount source outside the project: %v (target %s)", mm["source"], target))
			continue
		}
		if isStatic {
			if static == "" && target == "static" {
				static = dir
			}
			continue
		}
		lang, _ := mm["lang"].(string)
		content = append(content, ContentDir{Path: dir, Lang: strings.ToLower(lang)})
	}
	return content, static, warnings
}

// mountSource validates a configured source: a string naming a directory
// strictly inside the project. Unlike configDir it has no fallback.
func mountSource(v any) (string, bool) {
	s, ok := v.(string)
	if !ok {
		return "", false
	}
	clean := path.Clean(strings.ReplaceAll(strings.TrimSpace(s), `\`, "/"))
	if clean == "." || !fs.ValidPath(clean) {
		return "", false
	}
	return clean, true
}

// dedupeDirs drops later entries with a path already seen. A dropped entry's
// Lang is copied onto the kept one when that has none.
func dedupeDirs(dirs []ContentDir) []ContentDir {
	seen := make(map[string]int, len(dirs))
	out := make([]ContentDir, 0, len(dirs))
	for _, d := range dirs {
		if i, ok := seen[d.Path]; ok {
			if out[i].Lang == "" {
				out[i].Lang = d.Lang
			}
			continue
		}
		seen[d.Path] = len(out)
		out = append(out, d)
	}
	return out
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
