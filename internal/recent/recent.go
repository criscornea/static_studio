// Package recent keeps a short list of recently opened projects in the
// user's config directory, never inside a project.
package recent

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"log/slog"
	"os"
	"path/filepath"
	"sync"
	"time"
)

// Max is the number of entries kept.
const Max = 10

// Entry is one recently opened project.
type Entry struct {
	Path     string    `json:"path"`
	Name     string    `json:"name"`
	Kind     string    `json:"kind"`
	OpenedAt time.Time `json:"openedAt"`
}

// Store persists entries in <dir>/static-studio/recent.json. A Store with
// an empty file path is disabled: it reads empty and ignores writes.
type Store struct {
	log  *slog.Logger
	mu   sync.Mutex
	file string
}

// New returns a Store rooted at configDir. An empty configDir disables it.
func New(log *slog.Logger, configDir string) *Store {
	if log == nil {
		log = slog.Default()
	}
	s := &Store{log: log}
	if configDir != "" {
		s.file = filepath.Join(configDir, "static-studio", "recent.json")
	}
	return s
}

// NewDefault uses os.UserConfigDir, disabling the store when it is unavailable.
func NewDefault(log *slog.Logger) *Store {
	dir, err := os.UserConfigDir()
	if err != nil {
		if log == nil {
			log = slog.Default()
		}
		log.Warn("config dir unavailable, recent projects disabled", "err", err)
		dir = ""
	}
	return New(log, dir)
}

func (s *Store) load() []Entry {
	if s.file == "" {
		return nil
	}
	data, err := os.ReadFile(s.file)
	if err != nil {
		if !errors.Is(err, fs.ErrNotExist) {
			s.log.Warn("reading recent projects failed", "err", err)
		}
		return nil
	}
	var entries []Entry
	if err := json.Unmarshal(data, &entries); err != nil {
		s.log.Warn("recent projects file is corrupt, ignoring", "err", err)
		return nil
	}
	return entries
}

func (s *Store) save(entries []Entry) error {
	if s.file == "" {
		return nil
	}
	data, err := json.Marshal(entries)
	if err != nil {
		return err
	}
	dir := filepath.Dir(s.file)
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return err
	}
	var suffix [6]byte
	if _, err := rand.Read(suffix[:]); err != nil {
		return err
	}
	tmp := filepath.Join(dir, ".recent."+hex.EncodeToString(suffix[:])+".tmp")
	if err := os.WriteFile(tmp, data, 0o600); err != nil {
		return err
	}
	if err := os.Rename(tmp, s.file); err != nil {
		_ = os.Remove(tmp)
		return err
	}
	return nil
}

// List returns entries, most recent first.
func (s *Store) List() []Entry {
	s.mu.Lock()
	defer s.mu.Unlock()
	entries := s.load()
	if entries == nil {
		entries = []Entry{}
	}
	return entries
}

// Add records a project as most recent, deduped by cleaned absolute path.
func (s *Store) Add(path, kind string) error {
	if s.file == "" {
		return nil
	}
	abs, err := filepath.Abs(path)
	if err != nil {
		return fmt.Errorf("absolute path: %w", err)
	}
	abs = filepath.Clean(abs)

	s.mu.Lock()
	defer s.mu.Unlock()
	entries := []Entry{{Path: abs, Name: filepath.Base(abs), Kind: kind, OpenedAt: time.Now().UTC()}}
	for _, e := range s.load() {
		if e.Path != abs {
			entries = append(entries, e)
		}
	}
	if len(entries) > Max {
		entries = entries[:Max]
	}
	return s.save(entries)
}

// Remove drops the entry with the given path, if present.
func (s *Store) Remove(path string) error {
	if s.file == "" {
		return nil
	}
	clean := filepath.Clean(path)
	s.mu.Lock()
	defer s.mu.Unlock()
	old := s.load()
	entries := make([]Entry, 0, len(old))
	for _, e := range old {
		if e.Path != clean {
			entries = append(entries, e)
		}
	}
	if len(entries) == len(old) {
		return nil
	}
	return s.save(entries)
}
