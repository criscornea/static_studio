package ssg

import (
	"errors"
	"io/fs"
)

type Kind string

const (
	KindHugo  Kind = "hugo"
	KindAstro Kind = "astro"
)

var (
	ErrNotDetected = errors.New("no static site generator detected")
	ErrUnsupported = errors.New("static site generator recognised but not supported")
)

// SSG describes one static site generator's conventions.
type SSG interface {
	Kind() Kind
	// Supported reports whether this build can actually edit such a project.
	Supported() bool
	// Detect looks for the generator's config file at the root fo fsys.
	// It returns the config file's path relative to the root.
	Detect(fsys fs.FS) (configFile string, ok bool)
	// ContentDir is where editable markdown lives, relative to the root.
	ContentDir() string
	// AssetDir is where images and other static files live, relative to the root.
	AssetDir() string
}

type Project struct {
	Root       string `json:"root"`
	Kind       Kind   `json:"kind"`
	ConfigFile string `json:"configFile"`
	ContentDir string `json:"contentDir"`
	AssetDir   string `json:"assetDir"`
}
