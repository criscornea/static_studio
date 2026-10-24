package ssg

import (
	"errors"
	"testing"
	"testing/fstest"
)

func mapFS(files map[string]string) fstest.MapFS {
	fsys := fstest.MapFS{}
	for name, data := range files {
		fsys[name] = &fstest.MapFile{Data: []byte(data)}
	}
	return fsys
}

func envOf(m map[string]string) func(string) string {
	return func(k string) string { return m[k] }
}

func TestHugoDirsMergedConfig(t *testing.T) {
	tests := []struct {
		name        string
		files       map[string]string
		env         map[string]string
		wantContent string
		wantAsset   string
	}{
		{"default dir overrides root file",
			map[string]string{
				"hugo.toml":                 "contentDir = 'root'\nstaticDir = 'rootstatic'\n",
				"config/_default/hugo.toml": "contentDir = 'default'\n",
			}, nil, "default", "rootstatic"},
		{"root file used when dir has no key",
			map[string]string{
				"hugo.toml":                   "contentDir = 'root'\n",
				"config/_default/params.toml": "x = 1\n",
			}, nil, "root", "static"},
		{"mixed formats and legacy name",
			map[string]string{
				"hugo.toml":                   "title = 'x'\n",
				"config/_default/config.yaml": "contentDir: fromyaml\n",
				"config/_default/extra.json":  `{"a": 1}`,
			}, nil, "fromyaml", "static"},
		{"file named after a key nests under it",
			map[string]string{
				"hugo.toml":                       "contentDir = 'root'\n",
				"config/_default/contentdir.toml": "x = 1\n",
			}, nil, "content", "static"},
		{"keys merge case-insensitively",
			map[string]string{
				"hugo.toml":                 "CONTENTDIR = 'root'\n",
				"config/_default/hugo.toml": "ContentDir = 'default'\n",
			}, nil, "default", "static"},
		{"no root file, only config dir",
			map[string]string{
				"config/_default/hugo.toml": "contentDir = 'only'\n",
			}, nil, "only", "static"},
		{"environment dir overrides default",
			map[string]string{
				"config/_default/hugo.toml":    "contentDir = 'default'\nstaticDir = 'ds'\n",
				"config/development/hugo.toml": "contentDir = 'dev'\n",
				"config/production/hugo.toml":  "contentDir = 'prod'\n",
			}, nil, "dev", "ds"},
		{"HUGO_ENVIRONMENT selects the overlay",
			map[string]string{
				"config/_default/hugo.toml":    "contentDir = 'default'\n",
				"config/development/hugo.toml": "contentDir = 'dev'\n",
				"config/production/hugo.toml":  "contentDir = 'prod'\n",
			}, map[string]string{"HUGO_ENVIRONMENT": "production"}, "prod", "static"},
		{"unknown environment keeps defaults",
			map[string]string{
				"config/_default/hugo.toml": "contentDir = 'default'\n",
			}, map[string]string{"HUGO_ENVIRONMENT": "staging"}, "default", "static"},
		{"environment name cannot escape config",
			map[string]string{
				"hugo.toml":   "contentDir = 'root'\n",
				"x/hugo.toml": "contentDir = 'evil'\n",
			}, map[string]string{"HUGO_ENVIRONMENT": "../x"}, "root", "static"},
		{"HUGO_CONTENTDIR beats every file",
			map[string]string{
				"hugo.toml":                    "contentDir = 'root'\n",
				"config/_default/hugo.toml":    "contentDir = 'default'\n",
				"config/development/hugo.toml": "contentDir = 'dev'\n",
			}, map[string]string{"HUGO_CONTENTDIR": "envdocs"}, "envdocs", "static"},
		{"HUGO_STATICDIR beats files and takes the first of a list",
			map[string]string{
				"hugo.toml": "staticDir = ['a', 'b']\n",
			}, map[string]string{"HUGO_STATICDIR": "x y"}, "content", "x"},
		{"env override cannot escape the project",
			map[string]string{
				"hugo.toml": "contentDir = 'root'\n",
			}, map[string]string{"HUGO_CONTENTDIR": "../outside", "HUGO_STATICDIR": "/etc"}, "content", "static"},
		{"blank env override is ignored",
			map[string]string{
				"hugo.toml": "contentDir = 'root'\n",
			}, map[string]string{"HUGO_CONTENTDIR": "  "}, "root", "static"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			file := "hugo.toml"
			if _, ok := tt.files[file]; !ok {
				file = "config/_default/hugo.toml"
			}
			h := hugo{env: envOf(tt.env)}
			content, asset, err := h.Dirs(mapFS(tt.files), file)
			if err != nil {
				t.Fatal(err)
			}
			if content != tt.wantContent || asset != tt.wantAsset {
				t.Errorf("got (%q, %q), want (%q, %q)", content, asset, tt.wantContent, tt.wantAsset)
			}
		})
	}
}

func TestHugoDirsInvalidConfigDirFile(t *testing.T) {
	for _, dir := range []string{"config/_default", "config/development"} {
		fsys := mapFS(map[string]string{
			"hugo.toml":          "contentDir = 'x'\n",
			dir + "/params.toml": "a = = 1\n",
		})
		_, _, err := hugo{env: envOf(nil)}.Dirs(fsys, "hugo.toml")
		var ce *ConfigError
		if !errors.Is(err, ErrInvalidConfig) || !errors.As(err, &ce) || ce.File != dir+"/params.toml" {
			t.Errorf("%s: got %v, want ConfigError for params.toml", dir, err)
		}
	}
}
