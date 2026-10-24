package ssg

import (
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
)

// detectors are tried in order; the first match wins.
var detectors = []SSG{hugo{env: os.Getenv}, astro{}}

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

		var l *Layout
		if lo, ok := d.(layouter); ok {
			var err error
			if l, err = lo.Layout(fsys, cfg); err != nil {
				return nil, err
			}
		} else {
			contentDir, assetDir, err := d.Dirs(fsys, cfg)
			if err != nil {
				return nil, err
			}
			l = &Layout{ContentDir: contentDir, AssetDir: assetDir, ContentDirs: []ContentDir{{Path: contentDir}}}
		}

		return &Project{
			Kind:        d.Kind(),
			ConfigFile:  cfg,
			ContentDir:  l.ContentDir,
			ContentDirs: l.ContentDirs,
			AssetDir:    l.AssetDir,
			Warnings:    l.Warnings,
		}, nil
	}
	return nil, ErrNotDetected
}
