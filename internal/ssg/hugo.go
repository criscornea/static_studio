package ssg

import "io/fs"

type hugo struct{}

func (hugo) Kind() Kind         { return KindHugo }
func (hugo) Supported() bool    { return true }
func (hugo) ContentDir() string { return "content" }
func (hugo) AssetDir() string   { return "static" }

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
