package ssg

import "io/fs"

// astro is recognised but not editable. It exists so that pointing the tool
// at an Astro project produces a clear message instead of "not a site".
type astro struct{}

func (astro) Kind() Kind         { return KindAstro }
func (astro) Supported() bool    { return false }
func (astro) ContentDir() string { return "src/content" }
func (astro) AssetDir() string   { return "public" }

var astroConfigs = []string{
	"astro.config.mjs", "astro.config.ts", "astro.config.mts",
	"astro.config.js", "astro.config.cjs",
}

func (astro) Detect(fsys fs.FS) (string, bool) {
	for _, name := range astroConfigs {
		if info, err := fs.Stat(fsys, name); err == nil && !info.IsDir() {
			return name, true
		}
	}
	return "", false
}
