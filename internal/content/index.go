package content

import (
	"io/fs"
	"os"
	"strings"
	"sync"

	"github.com/criscornea/static_studio/internal/frontmatter"
)

// Index caches what the tree and search read from each page, so unchanged
// pages are not read again. Every lookup compares the file's size,
// modification time and identity with the cached values; a mismatch reloads
// the page, so an external edit shows up without a file watcher.
//
// An Index is safe for concurrent use. The zero value is not usable: call
// NewIndex. A nil *Index is valid and caches nothing.
type Index struct {
	mu      sync.RWMutex
	entries map[string]*indexed
}

// indexed is the cached view of one page.
type indexed struct {
	info fs.FileInfo // what the entry was validated against

	// treeFields is nil when the page has no readable frontmatter.
	treeFields *frontmatter.Fields

	// Search data. title and draft are set only when the fields parsed.
	title string
	draft bool
	tags  string // tags joined by newlines
	body  string // the text after the frontmatter, or the whole file

	// lower is body in lower case, kept only for pure-ASCII bodies, where
	// it has the same byte offsets and a plain substring search gives the
	// same answer as a case-insensitive regexp.
	lower string
}

// NewIndex returns an empty Index.
func NewIndex() *Index {
	return &Index{entries: map[string]*indexed{}}
}

// Len returns how many pages are cached.
func (ix *Index) Len() int {
	if ix == nil {
		return 0
	}
	ix.mu.RLock()
	defer ix.mu.RUnlock()
	return len(ix.entries)
}

// Drop forgets name. The next lookup reads the page again.
func (ix *Index) Drop(name string) {
	if ix == nil {
		return
	}
	ix.mu.Lock()
	delete(ix.entries, name)
	ix.mu.Unlock()
}

// Reset forgets every page.
func (ix *Index) Reset() {
	if ix == nil {
		return
	}
	ix.mu.Lock()
	clear(ix.entries)
	ix.mu.Unlock()
}

// prune forgets cached pages under prefix that are not in seen, such as
// pages deleted or renamed outside the app.
func (ix *Index) prune(prefix string, seen map[string]bool) {
	if ix == nil {
		return
	}
	ix.mu.Lock()
	for name := range ix.entries {
		if strings.HasPrefix(name, prefix) && !seen[name] {
			delete(ix.entries, name)
		}
	}
	ix.mu.Unlock()
}

// fresh reports whether cached still describes info.
func fresh(cached, info fs.FileInfo) bool {
	return cached.Size() == info.Size() &&
		cached.ModTime().Equal(info.ModTime()) &&
		os.SameFile(cached, info)
}

// lookup returns the entry for the page name, whose current stat is info.
// It returns nil when the page is too large or cannot be read.
func (ix *Index) lookup(fsys fs.FS, name string, info fs.FileInfo) *indexed {
	// A symlink's own stat never changes when its target is edited, so
	// validate against the target.
	if info.Mode()&fs.ModeSymlink != 0 {
		target, err := fs.Stat(fsys, name)
		if err != nil {
			ix.Drop(name)
			return nil
		}
		info = target
	}
	if info.Size() > maxPageSize {
		ix.Drop(name)
		return nil
	}
	if ix != nil {
		ix.mu.RLock()
		e := ix.entries[name]
		ix.mu.RUnlock()
		if e != nil && fresh(e.info, info) {
			return e
		}
	}

	e := load(fsys, name, info)
	if ix == nil {
		return e
	}
	ix.mu.Lock()
	if e == nil {
		delete(ix.entries, name)
	} else {
		ix.entries[name] = e
	}
	ix.mu.Unlock()
	return e
}

// load reads name and extracts everything the index keeps.
func load(fsys fs.FS, name string, info fs.FileInfo) *indexed {
	data, err := fs.ReadFile(fsys, name)
	if err != nil {
		return nil
	}
	// A write between the stat and the read leaves info stale, so the next
	// lookup reloads. Only a racing same-size, same-mtime write is missed.
	e := &indexed{info: info}
	defer func() {
		if isASCII(e.body) {
			e.lower = strings.ToLower(e.body)
		}
	}()
	e.body = string(data)

	// A page with broken frontmatter is searched as plain text, so a typo in
	// its settings does not hide it.
	doc, err := frontmatter.Split(data)
	if err != nil {
		return e
	}
	e.body = string(doc.Body)
	fields, err := doc.Fields()
	if err != nil {
		return e
	}
	e.title = fields.Title
	e.draft = fields.Draft
	e.tags = strings.Join(fields.Tags, "\n")
	if doc.Format != frontmatter.FormatNone {
		e.treeFields = fields
	}
	return e
}

func isASCII(s string) bool {
	for i := 0; i < len(s); i++ {
		if s[i] >= 0x80 {
			return false
		}
	}
	return true
}
