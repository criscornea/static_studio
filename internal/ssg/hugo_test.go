package ssg

import (
	"strings"
	"testing"
	"testing/fstest"
)

func TestHugoDirs(t *testing.T) {
	tests := []struct {
		name        string
		config      string
		file        string
		wantContent string
		wantAsset   string
	}{
		{"toml", "contentDir = 'docs'\nstaticDir = 'assets'\n", "hugo.toml", "docs", "assets"},
		{"yaml", "contentDir: docs\nstaticDir: assets\n", "hugo.yaml", "docs", "assets"},
		{"yml extension", "contentDir: docs\n", "config.yml", "docs", "static"},
		{"json", `{"contentDir": "docs", "staticDir": "assets"}`, "hugo.json", "docs", "assets"},
		{"keys are case-insensitive", "CONTENTDIR = 'docs'\n", "hugo.toml", "docs", "static"},
		{"nested directory", "contentDir = 'site/content'\n", "hugo.toml", "site/content", "static"},
		{"path is cleaned", "contentDir = 'site//content/'\n", "hugo.toml", "site/content", "static"},
		{"static dir list uses the first", "staticDir = ['assets', 'more']\n", "hugo.toml", "content", "assets"},
		{"empty static dir list", "staticDir = []\n", "hugo.toml", "content", "static"},
		{"neither key set", "title = 'x'\n", "hugo.toml", "content", "static"},
		{"empty config", "", "hugo.toml", "content", "static"},
		{"invalid toml", "contentDir = [unclosed\n", "hugo.toml", "content", "static"},
		{"invalid yaml", "contentDir: [unclosed\n", "hugo.yaml", "content", "static"},
		{"wrong type", "contentDir = 3\n", "hugo.toml", "content", "static"},
		{"empty string", "contentDir = ''\n", "hugo.toml", "content", "static"},
		{"dot", "contentDir = '.'\n", "hugo.toml", "content", "static"},
		{"absolute path", "contentDir = '/etc'\n", "hugo.toml", "content", "static"},
		{"parent traversal", "contentDir = '../elsewhere'\nstaticDir = 'a/../../b'\n", "hugo.toml", "content", "static"},
		{"backslashes", `contentDir = 'site\content'` + "\n", "hugo.toml", "site/content", "static"},
		{"config directory layout", "contentDir = 'docs'\n", "config/_default/hugo.toml", "docs", "static"},
		{"too large", "contentDir = 'docs'\n" + strings.Repeat("#", maxConfigSize), "hugo.toml", "content", "static"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fsys := fstest.MapFS{tt.file: {Data: []byte(tt.config)}}

			content, asset := hugo{}.Dirs(fsys, tt.file)

			if content != tt.wantContent {
				t.Errorf("contentDir = %q, want %q", content, tt.wantContent)
			}
			if asset != tt.wantAsset {
				t.Errorf("assetDir = %q, want %q", asset, tt.wantAsset)
			}
		})
	}
}

func TestHugoDirsUnreadableConfig(t *testing.T) {
	content, asset := hugo{}.Dirs(fstest.MapFS{}, "hugo.toml")

	if content != "content" || asset != "static" {
		t.Errorf("dirs = %q, %q, want the defaults", content, asset)
	}
}

func TestDetectFSUsesConfiguredDirs(t *testing.T) {
	got, err := detectFS(fstest.MapFS{"hugo.toml": {Data: []byte("contentDir = 'docs'\n")}})
	if err != nil {
		t.Fatal(err)
	}
	if got.ContentDir != "docs" {
		t.Errorf("ContentDir = %q, want docs", got.ContentDir)
	}
}
