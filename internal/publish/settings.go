// Package publish pushes the built site to a git remote branch, using the
// system git binary and the user's own credentials.
package publish

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"log/slog"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"slices"
	"strings"
	"sync"
)

// DefaultBranch is used when no branch is configured.
const DefaultBranch = "gh-pages"

// ErrInvalid wraps validation failures that the user can fix.
var ErrInvalid = errors.New("invalid publish settings")

// Settings say where a project is published.
type Settings struct {
	// Target is TargetGitHubPages or TargetRsync. Empty means GitHub Pages so
	// settings saved before targets existed load unchanged.
	Target string `json:"target"`
	Remote string `json:"remote"`
	Branch string `json:"branch"`

	// SiteKind is SiteProject, SiteUser or SiteCustom; empty means none chosen,
	// so settings saved before site addresses existed load unchanged.
	SiteKind string `json:"siteKind"`
	// CustomDomain is the single source of truth for the custom domain.
	CustomDomain string `json:"customDomain"`
	// CNAMEChoice is CNAMESetting or CNAMEStatic, the answer to a conflict
	// between CustomDomain and static/CNAME, made against the static/CNAME
	// value kept in CNAMEChoiceFor so a later edit of that file asks again.
	CNAMEChoice    string `json:"cnameChoice"`
	CNAMEChoiceFor string `json:"cnameChoiceFor"`

	// Rsync fields apply when Target is TargetRsync. No credentials are kept:
	// ssh uses the user's own agent, keys, config and known_hosts.
	RsyncDest     string `json:"rsyncDest"`
	RsyncPort     int    `json:"rsyncPort"`
	RsyncIdentity string `json:"rsyncIdentity"`
	RsyncDelete   bool   `json:"rsyncDelete"`
	RsyncDryRun   bool   `json:"rsyncDryRun"`
}

// Publish targets.
const (
	TargetGitHubPages = "github-pages"
	TargetRsync       = "rsync"
)

var branchRe = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9._/-]*$`)

// protectedBranches are never force-pushed to: that is where the source of a
// site normally lives.
var protectedBranches = []string{"main", "master"}

// validRemote reports whether the remote is a URL without a password, an
// scp-style address, or an absolute local path.
func validRemote(r string) bool {
	if strings.Contains(r, "://") {
		u, err := url.Parse(r)
		if err != nil || u.Host == "" && u.Scheme != "file" {
			return false
		}
		_, hasPassword := u.User.Password()
		return !hasPassword
	}
	if i := strings.IndexAny(r, ":/"); i > 0 && r[i] == ':' && !filepath.IsAbs(r) {
		return true // scp-style user@host:path
	}
	return filepath.IsAbs(r)
}

// Validate checks the settings and fills in the default branch. The remote
// may not start with "-" so it can never be read as a git option, may not
// carry a password, and a local path must be absolute. The branch may not be
// main or master.
func (s Settings) Validate() (Settings, error) {
	s.Remote = strings.TrimSpace(s.Remote)
	s.Branch = strings.TrimSpace(s.Branch)
	if err := s.validateSite(); err != nil {
		return s, err
	}
	if s.Branch == "" {
		s.Branch = DefaultBranch
	}
	switch s.Target {
	case "":
		s.Target = TargetGitHubPages
	case TargetGitHubPages:
	case TargetRsync:
		return s.validateRsync()
	default:
		return s, fmt.Errorf("%w: the publish target is not supported", ErrInvalid)
	}
	switch {
	case s.Remote == "":
		return s, fmt.Errorf("%w: the remote is empty", ErrInvalid)
	case strings.HasPrefix(s.Remote, "-") || strings.ContainsAny(s.Remote, "\n\r\x00"):
		return s, fmt.Errorf("%w: the remote is not a valid URL or path", ErrInvalid)
	case !validRemote(s.Remote):
		return s, fmt.Errorf("%w: the remote must be a URL without a password, an scp-style address, or a full local path", ErrInvalid)
	case slices.Contains(protectedBranches, strings.ToLower(s.Branch)):
		return s, fmt.Errorf("%w: publishing replaces the whole branch, so %s is not allowed; use gh-pages or another branch", ErrInvalid, s.Branch)
	case !branchRe.MatchString(s.Branch) || strings.Contains(s.Branch, "..") ||
		strings.HasSuffix(s.Branch, "/") || strings.HasSuffix(s.Branch, ".lock") || strings.Contains(s.Branch, "//"):
		return s, fmt.Errorf("%w: the branch name is not valid", ErrInvalid)
	}
	return s, nil
}

// Store keeps settings per project path in <configDir>/static-studio/publish.json,
// never inside the project. An empty file path disables it. Only the remote
// URL and branch are stored, no credentials.
type Store struct {
	log  *slog.Logger
	mu   sync.Mutex
	file string
}

// NewStore returns a Store rooted at configDir. An empty configDir disables it.
func NewStore(log *slog.Logger, configDir string) *Store {
	if log == nil {
		log = slog.Default()
	}
	s := &Store{log: log}
	if configDir != "" {
		s.file = filepath.Join(configDir, "static-studio", "publish.json")
	}
	return s
}

// NewDefaultStore uses os.UserConfigDir, disabling the store when unavailable.
func NewDefaultStore(log *slog.Logger) *Store {
	dir, err := os.UserConfigDir()
	if err != nil {
		if log == nil {
			log = slog.Default()
		}
		log.Warn("config dir unavailable, publish settings disabled", "err", err)
		dir = ""
	}
	return NewStore(log, dir)
}

func (s *Store) load() map[string]Settings {
	all := map[string]Settings{}
	if s.file == "" {
		return all
	}
	data, err := os.ReadFile(s.file)
	if err != nil {
		if !errors.Is(err, fs.ErrNotExist) {
			s.log.Warn("reading publish settings failed", "err", err)
		}
		return all
	}
	if err := json.Unmarshal(data, &all); err != nil || all == nil {
		s.log.Warn("publish settings file is corrupt, ignoring", "err", err)
		return map[string]Settings{}
	}
	return all
}

func key(project string) string {
	if abs, err := filepath.Abs(project); err == nil {
		project = abs
	}
	return filepath.Clean(project)
}

// Get returns the settings for a project. Unset settings have the default branch.
func (s *Store) Get(project string) Settings {
	s.mu.Lock()
	defer s.mu.Unlock()
	got := s.load()[key(project)]
	if got.Branch == "" {
		got.Branch = DefaultBranch
	}
	if got.Target == "" {
		got.Target = TargetGitHubPages
	}
	return got
}

// Set validates and saves the settings for a project, returning what was saved.
func (s *Store) Set(project string, in Settings) (Settings, error) {
	clean, err := in.Validate()
	if err != nil {
		return in, err
	}
	if s.file == "" {
		return clean, nil
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	all := s.load()
	all[key(project)] = clean
	data, err := json.Marshal(all)
	if err != nil {
		return clean, err
	}
	dir := filepath.Dir(s.file)
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return clean, err
	}
	var suffix [6]byte
	if _, err := rand.Read(suffix[:]); err != nil {
		return clean, err
	}
	tmp := filepath.Join(dir, ".publish."+hex.EncodeToString(suffix[:])+".tmp")
	if err := os.WriteFile(tmp, data, 0o600); err != nil {
		return clean, err
	}
	if err := os.Rename(tmp, s.file); err != nil {
		_ = os.Remove(tmp)
		return clean, err
	}
	return clean, nil
}
