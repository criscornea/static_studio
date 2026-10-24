package content

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"io/fs"
	"path"
	"strings"

	"github.com/criscornea/static_studio/internal/frontmatter"
)

// maxPageSize caps how much of a single content file is read into memory
const maxPageSize = 2 << 20 // 2 MiB

var (
	ErrNotFound    = errors.New("page not found")
	ErrNotEditable = errors.New("not an editable content file")
	ErrTooLarge    = errors.New("page is too large to edit")
)

// Page is a content file split into its known fields and its body.
type Page struct {
	Path   string              `json:"path"`
	Format frontmatter.Format  `json:"format"`
	Fields *frontmatter.Fields `json:"fields"`
	Body   string              `json:"body"`
	// BodyLine is the 1-based line of the file where Body starts.
	BodyLine int `json:"bodyLine"`
	// Hash is the sha256 of the file bytes, hex encoded. Saves send it back.
	Hash string `json:"hash"`
}

func ReadPage(fsys fs.FS, dirs Dirs, name string) (*Page, error) {
	if _, err := checkPagePath(dirs, name); err != nil {
		return nil, err
	}

	info, err := fs.Stat(fsys, name)
	switch {
	case errors.Is(err, fs.ErrNotExist):
		return nil, ErrNotFound
	case err != nil:
		return nil, fmt.Errorf("stat %s: %w", name, err)
	case info.IsDir():
		return nil, ErrNotEditable
	case info.Size() > maxPageSize:
		return nil, ErrTooLarge
	}

	data, err := fs.ReadFile(fsys, name)
	if err != nil {
		return nil, fmt.Errorf("reading %s: %w", name, err)
	}

	doc, err := frontmatter.Split(data)
	if err != nil {
		return nil, err
	}
	fields, err := doc.Fields()
	if err != nil {
		return nil, err
	}

	return &Page{
		Path:     name,
		Format:   doc.Format,
		Fields:   fields,
		Body:     string(doc.Body),
		BodyLine: 1 + bytes.Count(data[:len(data)-len(doc.Body)], []byte("\n")),
		Hash:     hashBytes(data),
	}, nil
}

// checkPagePath rejects anything that is not a well-formed path to an
// editable file inside one of dirs. It returns that content dir.
func checkPagePath(dirs Dirs, name string) (string, error) {
	if !fs.ValidPath(name) {
		return "", ErrNotEditable
	}
	dir, ok := dirs.Containing(name)
	if !ok {
		return "", ErrNotEditable
	}
	if !editableExts[strings.ToLower(path.Ext(name))] {
		return "", ErrNotEditable
	}
	return dir, nil
}

func hashBytes(data []byte) string {
	sum := sha256.Sum256(data)
	return hex.EncodeToString(sum[:])
}
