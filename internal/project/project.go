// Package project holds the currently open project and confines all
// filesystem access to its root directory.
package project

import (
	"crypto/rand"
	"encoding/base64"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"sync"

	"github.com/criscornea/static_studio/internal/ssg"
)

var (
	ErrNoProject = errors.New("no project is open")
	ErrStaleID   = errors.New("project id does not match the open project")
)

// Open is the currently open project and its confined filesystem handle.
type Open struct {
	ID   string
	Info ssg.Project

	root *os.Root
}

// Root returns the confined filesystem handle. Every path-based operation
// must go through it: it refuses to escape the project directory, including
// via symlinks and "..".
func (o *Open) Root() *os.Root { return o.root }

// Manager owns the single open project. It is safe for concurrent use.
type Manager struct {
	log     *slog.Logger
	mu      sync.RWMutex
	current *Open
}

// NewManager returns a Manager. A nil logger falls back to slog.Default.
func NewManager(log *slog.Logger) *Manager {
	if log == nil {
		log = slog.Default()
	}
	return &Manager{log: log}
}

func (m *Manager) Open(dir string) (*Open, error) {
	info, err := ssg.Detect(dir)
	if err != nil {
		return nil, err
	}

	root, err := os.OpenRoot(info.Root)
	if err != nil {
		return nil, fmt.Errorf("openening project root: %w", err)
	}

	id, err := newID()
	if err != nil {
		_ = root.Close()

		return nil, err
	}

	next := &Open{Info: *info, ID: id, root: root}

	m.mu.Lock()
	defer m.mu.Unlock()

	if err := m.closeLocked(); err != nil {
		// The new project is open and usable; a failure to close the old
		// root is a possible descriptor leak, not a reason to fail here.
		m.log.Warn("closing the previous project root failed", "err", err)
	}
	m.current = next

	return next, nil
}

// Current returns the open project, or ErrNoProject.
func (m *Manager) Current() (*Open, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	if m.current == nil {
		return nil, ErrNoProject
	}

	return m.current, nil
}

// Require returns the open project only if id matches it. An empty id is
// rejected: callers must state which project they believe is open.
func (m *Manager) Require(id string) (*Open, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	if m.current == nil {
		return nil, ErrNoProject
	}
	if id != m.current.ID {
		return nil, ErrStaleID
	}

	return m.current, nil
}

// Close closes the open project, if any. It is safe to call repeatedly.
func (m *Manager) Close() error {
	m.mu.Lock()
	defer m.mu.Unlock()

	return m.closeLocked()
}

// closeLocked requires m.mu to be held for writing.
func (m *Manager) closeLocked() error {
	if m.current == nil {
		return nil
	}

	err := m.current.root.Close()
	m.current = nil

	return err
}

func newID() (string, error) {
	b := make([]byte, 16)
	if _, err := rand.Read(b); err != nil {
		return "", fmt.Errorf("generating project id: %w", err)
	}

	return base64.RawURLEncoding.EncodeToString(b), nil
}
