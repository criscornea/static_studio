package ssg

import (
	"errors"
	"io/fs"
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
		{"wrong type", "contentDir = 3\n", "hugo.toml", "content", "static"},
		{"empty string", "contentDir = ''\n", "hugo.toml", "content", "static"},
		{"dot", "contentDir = '.'\n", "hugo.toml", "content", "static"},
		{"absolute path", "contentDir = '/etc'\n", "hugo.toml", "content", "static"},
		{"parent traversal", "contentDir = '../elsewhere'\nstaticDir = 'a/../../b'\n", "hugo.toml", "content", "static"},
		{"backslashes", `contentDir = 'site\content'` + "\n", "hugo.toml", "site/content", "static"},
		{"config directory layout", "contentDir = 'docs'\n", "config/_default/hugo.toml", "docs", "static"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fsys := fstest.MapFS{tt.file: {Data: []byte(tt.config)}}

			content, asset, err := hugo{}.Dirs(fsys, tt.file)
			if err != nil {
				t.Fatal(err)
			}

			if content != tt.wantContent {
				t.Errorf("contentDir = %q, want %q", content, tt.wantContent)
			}
			if asset != tt.wantAsset {
				t.Errorf("assetDir = %q, want %q", asset, tt.wantAsset)
			}
		})
	}
}

func TestHugoDirsMissingConfigIsNotAnError(t *testing.T) {
	content, asset, err := hugo{}.Dirs(fstest.MapFS{}, "hugo.toml")

	if err != nil {
		t.Fatal(err)
	}
	if content != "content" || asset != "static" {
		t.Errorf("dirs = %q, %q, want the defaults", content, asset)
	}
}

type failingFS struct{}

func (f failingFS) Open(string) (fs.File, error) {
	return nil, &fs.PathError{Op: "open", Path: "hugo.toml", Err: fs.ErrPermission}
}

func TestHugoDirsInvalidConfig(t *testing.T) {
	tests := []struct {
		name       string
		fsys       fs.FS
		file       string
		wantReason string
	}{
		{"toml syntax", fstest.MapFS{"hugo.toml": {Data: []byte("title = 'x'\ncontentDir = [unclosed\n")}}, "hugo.toml", "line 2"},
		{"yaml syntax", fstest.MapFS{"hugo.yaml": {Data: []byte("contentDir: [unclosed\n")}}, "hugo.yaml", ""},
		{"json syntax", fstest.MapFS{"hugo.json": {Data: []byte("{\n\"a\": 1,\n}")}}, "hugo.json", "line 3"},
		{"too large", fstest.MapFS{"hugo.toml": {Data: []byte("contentDir = 'docs'\n" + strings.Repeat("#", maxConfigSize))}}, "hugo.toml", "larger than"},
		{"unreadable", failingFS{}, "hugo.toml", "cannot be read"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, _, err := hugo{}.Dirs(tt.fsys, tt.file)

			if !errors.Is(err, ErrInvalidConfig) {
				t.Fatalf("err = %v, want ErrInvalidConfig", err)
			}
			var ce *ConfigError
			if !errors.As(err, &ce) || ce.File != tt.file {
				t.Fatalf("err = %#v, want ConfigError for %s", err, tt.file)
			}
			if !strings.Contains(ce.Reason, tt.wantReason) {
				t.Errorf("reason = %q, want it to contain %q", ce.Reason, tt.wantReason)
			}
		})
	}
}

func TestDetectFSReturnsInvalidConfig(t *testing.T) {
	_, err := detectFS(fstest.MapFS{"hugo.toml": {Data: []byte("a = [")}})

	if !errors.Is(err, ErrInvalidConfig) {
		t.Fatalf("err = %v, want ErrInvalidConfig", err)
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
