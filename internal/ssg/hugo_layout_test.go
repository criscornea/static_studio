package ssg

import (
	"reflect"
	"testing"
)

func TestHugoLayout(t *testing.T) {
	tests := []struct {
		name      string
		files     map[string]string
		wantDirs  []ContentDir
		wantAsset string
	}{
		{
			name:      "default",
			files:     map[string]string{"hugo.toml": ""},
			wantDirs:  []ContentDir{{Path: "content"}},
			wantAsset: "static",
		},
		{
			name: "language dirs sorted, deduplicated",
			files: map[string]string{"hugo.toml": `
contentDir = 'content/en'
[languages.nl]
contentDir = 'content/nl'
[languages.de]
contentDir = 'content/de'
[languages.en]
contentDir = 'content/en'
`},
			wantDirs:  []ContentDir{{Path: "content/en", Lang: "en"}, {Path: "content/de", Lang: "de"}, {Path: "content/nl", Lang: "nl"}},
			wantAsset: "static",
		},
		{
			name: "all languages own a dir: implicit content dropped",
			files: map[string]string{"hugo.toml": `
defaultContentLanguage = 'fr'
[languages.en]
contentDir = 'content/english'
[languages.fr]
contentDir = 'content/french'
`},
			wantDirs:  []ContentDir{{Path: "content/french", Lang: "fr"}, {Path: "content/english", Lang: "en"}},
			wantAsset: "static",
		},
		{
			name: "language without own dir keeps implicit, nested dropped",
			files: map[string]string{"hugo.toml": `
[languages.en]
[languages.fr]
contentDir = 'content/french'
`},
			wantDirs:  []ContentDir{{Path: "content"}},
			wantAsset: "static",
		},
		{
			name: "static mount to subpath does not set asset dir",
			files: map[string]string{"hugo.toml": `
[[module.mounts]]
source = 'x'
target = 'static/more'
`},
			wantDirs:  []ContentDir{{Path: "content"}},
			wantAsset: "static",
		},
		{
			name: "language dir outside project skipped",
			files: map[string]string{"hugo.toml": `
[languages.de]
contentDir = '../x'
[languages.fr]
contentDir = '/etc'
`},
			wantDirs:  []ContentDir{{Path: "content"}},
			wantAsset: "static",
		},
		{
			name: "content mount replaces implicit dir",
			files: map[string]string{"hugo.toml": `
contentDir = 'ignored'
[[module.mounts]]
source = 'docs'
target = 'content'
[[module.mounts]]
source = 'blog'
target = 'content/blog'
lang = 'EN'
[[module.mounts]]
source = 'docs'
target = 'content/again'
[[module.mounts]]
source = 'layouts'
target = 'layouts'
`},
			wantDirs:  []ContentDir{{Path: "docs"}, {Path: "blog", Lang: "en"}},
			wantAsset: "static",
		},
		{
			name: "refused mount sources",
			files: map[string]string{"hugo.toml": `
[[module.mounts]]
source = '/abs'
target = 'content'
[[module.mounts]]
source = '../up'
target = 'content'
[[module.mounts]]
source = '.'
target = 'content'
[[module.mounts]]
source = 'x/../../y'
target = 'static'
[[module.imports]]
path = 'github.com/a/b'
`},
			wantDirs:  []ContentDir{{Path: "content"}},
			wantAsset: "static",
		},
		{
			name: "first static mount sets asset dir",
			files: map[string]string{"hugo.toml": `
[[module.mounts]]
source = 'assets1'
target = 'static'
[[module.mounts]]
source = 'assets2'
target = 'static/more'
`},
			wantDirs:  []ContentDir{{Path: "content"}},
			wantAsset: "assets1",
		},
		{
			name: "mounts from config dir",
			files: map[string]string{
				"hugo.toml":                      "",
				"config/_default/module.toml":    "[[mounts]]\nsource='m'\ntarget='content'\n",
				"config/_default/languages.toml": "[de]\ncontentDir='content/de'\n",
			},
			wantDirs:  []ContentDir{{Path: "m"}, {Path: "content/de", Lang: "de"}},
			wantAsset: "static",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			l, err := hugo{env: envOf(nil)}.Layout(mapFS(tt.files), "hugo.toml")
			if err != nil {
				t.Fatal(err)
			}
			if !reflect.DeepEqual(l.ContentDirs, tt.wantDirs) {
				t.Errorf("dirs: got %v, want %v", l.ContentDirs, tt.wantDirs)
			}
			if l.ContentDir != tt.wantDirs[0].Path {
				t.Errorf("primary: got %q", l.ContentDir)
			}
			if l.AssetDir != tt.wantAsset {
				t.Errorf("asset: got %q, want %q", l.AssetDir, tt.wantAsset)
			}
		})
	}
}
