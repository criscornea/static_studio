package ssg

import (
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
)

// detectors are tried in order; the first match wins.
var detectors = []SSG{hugo{}, astro{}}

// Detect identifies the generator used by the project rooted at dir.
// It returns ErrNotDetected or ErrUnsupported when the directory cannot be used.
func Detect(dir string) (*Project, error) {
	abs, err := filepath.Abs(dir)
	if err != nil {
		return nil, fmt.Errorf("resolving %q: %w", dir, err)
	}

	info, err := os.Stat(abs)
	if err != nil {
		return nil, fmt.Errorf("opening %q: %w", abs, err)
	}
	if !info.IsDir() {
		return nil, fmt.Errorf("%q is not a directory", abs)
	}

	p, err := detectFS(os.DirFS(abs))
	if err != nil {
		return nil, err
	}

	p.Root = abs

	return p, nil
}

// detectFS is the testable core: it neeeds no real filesystem
func detectFS(fsys fs.FS) (*Project, error) {
	for _, d := range detectors {
		cfg, ok := d.Detect(fsys)
		if !ok {
			continue
		}
		if !d.Supported() {
			return nil, fmt.Errorf("%w: %s", ErrUnsupported, d.Kind())
		}

		return &Project{
			Kind:       d.Kind(),
			ConfigFile: cfg,
			ContentDir: d.ContentDir(),
			AssetDir:   d.AssetDir(),
		}, nil
	}
	return nil, ErrNotDetected
}
