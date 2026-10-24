package ssg

import (
	"errors"
	"fmt"
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
	// ErrInvalidConfig marks a config file that exists but cannot be used.
	// The concrete error is a *ConfigError.
	ErrInvalidConfig = errors.New("invalid site configuration")
)

// ConfigError says which config file is unusable and why.
type ConfigError struct {
	File   string // slash-separated, relative to the project root
	Reason string
}

func (e *ConfigError) Error() string {
	return fmt.Sprintf("%s: %s: %s", ErrInvalidConfig, e.File, e.Reason)
}

func (e *ConfigError) Is(target error) bool { return target == ErrInvalidConfig }

// SSG describes one static site generator's conventions.
type SSG interface {
	Kind() Kind
	// Supported reports whether this build can actually edit such a project.
	Supported() bool
	// Detect looks for the generator's config file at the root fo fsys.
	// It returns the config file's path relative to the root.
	Detect(fsys fs.FS) (configFile string, ok bool)
	// Dirs returns where editable markdown (contentDir) and images and other
	// static files (assetDir) live, relative to the root. It honours the
	// project's config file and falls back to the generator's defaults for
	// anything missing or pointing outside the project. A config file that
	// exists but cannot be read or parsed yields a *ConfigError.
	Dirs(fsys fs.FS, configFile string) (contentDir, assetDir string, err error)
}

type Project struct {
	Root       string `json:"root"`
	Kind       Kind   `json:"kind"`
	ConfigFile string `json:"configFile"`
	ContentDir string `json:"contentDir"`
	AssetDir   string `json:"assetDir"`
}
