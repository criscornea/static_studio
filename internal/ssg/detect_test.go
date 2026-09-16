package ssg

import (
	"errors"
	"os"
	"path/filepath"
	"testing"
	"testing/fstest"
)

func TestDetectFS(t *testing.T) {
	tests := []struct {
		name       string
		files      fstest.MapFS
		wantKind   Kind
		wantConfig string
		wantErr    error
	}{
		{
			name:       "hugo with modern config name",
			files:      fstest.MapFS{"hugo.toml": {}, "content/post.md": {}},
			wantKind:   KindHugo,
			wantConfig: "hugo.toml",
		},
		{
			name:       "hugo with legacy config name",
			files:      fstest.MapFS{"config.toml": {}},
			wantKind:   KindHugo,
			wantConfig: "config.toml",
		},
		{
			name:       "hugo.toml wins over config.toml",
			files:      fstest.MapFS{"config.toml": {}, "hugo.toml": {}},
			wantKind:   KindHugo,
			wantConfig: "hugo.toml",
		},
		{
			name:       "hugo with config directory layout",
			files:      fstest.MapFS{"config/_default/hugo.toml": {}},
			wantKind:   KindHugo,
			wantConfig: "config/_default/hugo.toml",
		},
		{
			name:    "astro is recognised but unsupported",
			files:   fstest.MapFS{"astro.config.mjs": {}, "package.json": {}},
			wantErr: ErrUnsupported,
		},
		{
			name:       "hugo wins when both are present",
			files:      fstest.MapFS{"hugo.toml": {}, "astro.config.mjs": {}},
			wantKind:   KindHugo,
			wantConfig: "hugo.toml",
		},
		{
			name:    "empty directory",
			files:   fstest.MapFS{},
			wantErr: ErrNotDetected,
		},
		{
			name:    "unrelated project",
			files:   fstest.MapFS{"package.json": {}, "src/index.ts": {}},
			wantErr: ErrNotDetected,
		},
		{
			name:    "config name taken by a directory",
			files:   fstest.MapFS{"hugo.toml/notes.md": {}},
			wantErr: ErrNotDetected,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := detectFS(tt.files)

			if tt.wantErr != nil {
				if !errors.Is(err, tt.wantErr) {
					t.Fatalf("err = %v, want %v", err, tt.wantErr)
				}
				if got != nil {
					t.Errorf("project = %+v, want nil on error", got)
				}
				return
			}

			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if got.Kind != tt.wantKind {
				t.Errorf("Kind = %q, want %q", got.Kind, tt.wantKind)
			}
			if got.ConfigFile != tt.wantConfig {
				t.Errorf("ConfigFile = %q, want %q", got.ConfigFile, tt.wantConfig)
			}
			if got.ContentDir == "" || got.AssetDir == "" {
				t.Errorf("ContentDir/AssetDir must be set, got %+v", got)
			}
		})
	}
}

func TestDetectResolvesAbsolutePath(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "hugo.toml"), []byte("title = 'x'\n"), 0o600); err != nil {
		t.Fatal(err)
	}

	got, err := Detect(dir)
	if err != nil {
		t.Fatalf("unexpetected error: %v", err)
	}
	if !filepath.IsAbs(got.Root) {
		t.Errorf("Root = %q, want an absolute path", got.Root)
	}
	if got.Kind != KindHugo {
		t.Errorf("Kind = %q, want %q", got.Kind, KindHugo)
	}
}

func TestDetectRejectsNonDirectories(t *testing.T) {
	dir := t.TempDir()
	file := filepath.Join(dir, "hugo.toml")
	if err := os.WriteFile(file, nil, 0o600); err != nil {
		t.Fatal(err)
	}

	if _, err := Detect(file); err == nil {
		t.Error("Detect on a file: err = nil, want an error")
	}
	if _, err := Detect(filepath.Join(dir, "does-not-exist")); err == nil {
		t.Error("Detect on a missing path: err = nil, want an error")
	}
}
