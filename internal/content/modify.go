package content

import (
	"bytes"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path"
	"strings"
	"sync"

	"github.com/criscornea/static_studio/internal/frontmatter"
)

// locks holds one mutex per open project root. It serialises check-and-write
// in SavePage, RenamePage and DeletePage within this process.
var locks sync.Map // *os.Root -> *sync.Mutex

func lockRoot(root *os.Root) func() {
	m, _ := locks.LoadOrStore(root, &sync.Mutex{})
	mu := m.(*sync.Mutex)
	mu.Lock()
	return mu.Unlock
}

// regularFile returns ErrNotFound if name is missing and ErrNotEditable if it
// is not a plain file (directories and symlinks are refused).
func regularFile(root *os.Root, name string) error {
	info, err := root.Lstat(name)
	switch {
	case errors.Is(err, fs.ErrNotExist):
		return ErrNotFound
	case err != nil:
		return fmt.Errorf("stat %s: %w", name, err)
	case !info.Mode().IsRegular():
		return ErrNotEditable
	}
	return nil
}

// checkSegments refuses hidden/node_modules segments in name below contentDir.
func checkSegments(contentDir, name string) error {
	for _, segment := range strings.Split(strings.TrimPrefix(name, contentDir+"/"), "/") {
		if skip(segment) {
			return ErrNotEditable
		}
	}
	return nil
}

// checkParents Lstats every folder between contentDir and name and returns
// ErrNotEditable for a symlink, which os.Root would follow. A missing folder
// ends the walk without error.
func checkParents(root *os.Root, contentDir, name string) error {
	cur := contentDir
	parts := strings.Split(strings.TrimPrefix(path.Dir(name), contentDir), "/")
	for _, part := range parts {
		if part == "" {
			continue
		}
		cur = cur + "/" + part
		info, err := root.Lstat(cur)
		switch {
		case errors.Is(err, fs.ErrNotExist):
			return nil
		case err != nil:
			return fmt.Errorf("stat %s: %w", cur, err)
		case info.Mode()&fs.ModeSymlink != 0 || !info.IsDir():
			return ErrNotEditable
		}
	}
	return nil
}

// RenamePage moves the page from to to, both paths inside a resolved content dir (they may be in different ones) with an
// editable extension. It never overwrites: ErrPageExists is returned if to is
// taken. Missing folders of to are created. It returns the new path.
func RenamePage(root *os.Root, dirs Dirs, from, to string) (string, error) {
	defer lockRoot(root)()
	fromDir, err := checkPagePath(dirs, from)
	if err != nil {
		return "", err
	}
	toDir, err := checkPagePath(dirs, to)
	if err != nil {
		return "", err
	}
	for i, name := range []string{from, to} {
		contentDir := []string{fromDir, toDir}[i]
		if err := checkSegments(contentDir, name); err != nil {
			return "", err
		}
		if err := checkParents(root, contentDir, name); err != nil {
			return "", err
		}
	}
	if err := regularFile(root, from); err != nil {
		return "", err
	}
	if from == to {
		return "", ErrPageExists
	}
	if toInfo, err := root.Lstat(to); err == nil {
		fromInfo, ferr := root.Lstat(from)
		if ferr != nil || !os.SameFile(fromInfo, toInfo) {
			return "", ErrPageExists
		}
		// Case-only rename on a case-insensitive filesystem.
		if err := root.Rename(from, to); err != nil {
			return "", fmt.Errorf("renaming %s: %w", from, err)
		}
		syncDir(root, path.Dir(to))
		return to, nil
	}

	dir := path.Dir(to)
	if err := root.MkdirAll(dir, 0o750); err != nil {
		return "", fmt.Errorf("creating %s: %w", dir, err)
	}
	hook("rename:after-regular")
	checked, err := root.Lstat(from)
	if err != nil {
		return "", ErrNotFound
	}
	if !checked.Mode().IsRegular() {
		return "", ErrNotEditable
	}
	// Grab-then-verify: move from aside so nobody can swap it any more, check
	// the grabbed copy is the file that was checked, then link it to its new
	// name. On any problem it is put back.
	hook("rename:before-link")
	grabbed, err := grabFile(root, from)
	if errors.Is(err, fs.ErrNotExist) {
		return "", ErrNotFound
	} else if err != nil {
		return "", fmt.Errorf("renaming %s: %w", from, err)
	}
	if gi, gerr := root.Lstat(grabbed); gerr != nil || !os.SameFile(checked, gi) {
		return "", undoGrab(root, grabbed, from, ErrConflict)
	}
	hook("rename:after-grab")
	// A hard link fails if to exists, so a race cannot overwrite anything.
	err = linkFile(root, grabbed, to)
	switch {
	case err == nil:
		_ = root.Remove(grabbed)
	case errors.Is(err, fs.ErrExist):
		return "", undoGrab(root, grabbed, from, ErrPageExists)
	default:
		// No hard link support: check again, then rename.
		if _, statErr := root.Lstat(to); statErr == nil {
			return "", undoGrab(root, grabbed, from, ErrPageExists)
		}
		if err := root.Rename(grabbed, to); err != nil {
			rerr := fmt.Errorf("renaming %s: %w", from, err)
			return "", undoGrab(root, grabbed, from, rerr)
		}
	}
	syncDir(root, dir)
	syncDir(root, path.Dir(from))
	return to, nil
}

// undoGrab puts a grabbed file back at name and returns cause, or ErrConflict
// if name was recreated meanwhile (the grabbed copy is then kept aside), or
// the restore error.
func undoGrab(root *os.Root, grabbed, name string, cause error) error {
	if _, err := restoreFile(root, grabbed, name); err != nil {
		return err
	}
	return cause
}

// DeletePage removes the page name: a plain file inside a content dir with an
// editable extension. Folders are never removed. hash must be the hash of the
// file as the caller last read it; otherwise ErrConflict is returned and the
// file is kept. The file is first moved aside and verified, so a change made
// by an external editor after the check is never deleted unseen.
func DeletePage(root *os.Root, dirs Dirs, name, hash string) error {
	defer lockRoot(root)()
	contentDir, err := checkPagePath(dirs, name)
	if err != nil {
		return err
	}
	if err := checkSegments(contentDir, name); err != nil {
		return err
	}
	if err := checkParents(root, contentDir, name); err != nil {
		return err
	}
	if err := regularFile(root, name); err != nil {
		return err
	}
	// Hash as a stream so pages of any size can be deleted.
	got, err := hashFile(root, name)
	if err != nil {
		return err
	}
	if got != hash {
		return ErrConflict
	}

	hook("delete:before-grab")
	grabbed, err := grabFile(root, name)
	if errors.Is(err, fs.ErrNotExist) {
		return ErrNotFound
	} else if err != nil {
		return fmt.Errorf("deleting %s: %w", name, err)
	}
	got, err = hashFile(root, grabbed)
	if err != nil {
		return undoGrab(root, grabbed, name, err)
	}
	if got != hash {
		return undoGrab(root, grabbed, name, ErrConflict)
	}
	if err := root.Remove(grabbed); err != nil {
		_, _ = restoreFile(root, grabbed, name)
		return fmt.Errorf("deleting %s: %w", name, err)
	}
	syncDir(root, path.Dir(name))
	return nil
}

// ErrConflict means the file on disk no longer matches the hash the caller read.
var ErrConflict = errors.New("page changed on disk")

// SavePage replaces the body of the page name, keeping its frontmatter bytes
// verbatim. hash must be the hash of the file as the caller last read it;
// otherwise ErrConflict is returned and nothing is written. It returns the
// hash of the saved file. When fields is not nil the known frontmatter
// fields are rewritten too, keeping unknown keys; a page without
// frontmatter gets a new YAML block.
func SavePage(root *os.Root, dirs Dirs, name, body, hash string, fields *frontmatter.Fields) (string, error) {
	defer lockRoot(root)()
	contentDir, err := checkPagePath(dirs, name)
	if err != nil {
		return "", err
	}
	if err := checkSegments(contentDir, name); err != nil {
		return "", err
	}
	if err := checkParents(root, contentDir, name); err != nil {
		return "", err
	}
	if err := regularFile(root, name); err != nil {
		return "", err
	}
	if len(body) > maxPageSize {
		return "", ErrTooLarge
	}

	info, err := root.Stat(name)
	if err != nil {
		return "", fmt.Errorf("stat %s: %w", name, err)
	}
	if info.Size() > maxPageSize {
		return "", ErrTooLarge
	}
	data, err := fs.ReadFile(root.FS(), name)
	if err != nil {
		return "", fmt.Errorf("reading %s: %w", name, err)
	}
	if hashBytes(data) != hash {
		return "", ErrConflict
	}

	doc, err := frontmatter.Split(data)
	if err != nil {
		return "", err
	}
	head := data[:len(data)-len(doc.Body)]
	if fields != nil {
		if head, err = rewriteHead(doc, head, *fields); err != nil {
			return "", err
		}
	}
	out := append(append([]byte{}, head...), body...)
	if len(out) > maxPageSize {
		return "", ErrTooLarge
	}

	// Grab-then-verify: move the file aside so no writer can change what is
	// checked, verify the hash on the moved copy, then link the new content
	// into place without overwriting anything created at name meanwhile.
	hook("save:before-grab")
	grabbed, err := grabFile(root, name)
	if errors.Is(err, fs.ErrNotExist) {
		return "", ErrNotFound
	} else if err != nil {
		return "", fmt.Errorf("saving %s: %w", name, err)
	}
	if _, err := verifyGrabbed(root, grabbed, name, hash); err != nil {
		return "", err
	}
	hook("save:after-grab")
	if err := writeFileAtomicNew(root, name, out, info.Mode().Perm()); err != nil {
		if errors.Is(err, fs.ErrExist) {
			// Recreated externally: keep that file, drop our grabbed copy
			// (identical to what the caller read).
			_ = root.Remove(grabbed)
			return "", ErrConflict
		}
		if _, rerr := restoreFile(root, grabbed, name); rerr != nil {
			return "", fmt.Errorf("saving %s: %w", name, errors.Join(err, rerr))
		}
		return "", fmt.Errorf("saving %s: %w", name, err)
	}
	_ = root.Remove(grabbed)
	return hashBytes(out), nil
}

// rewriteHead returns the file prefix (BOM and frontmatter block) with the
// known fields replaced.
func rewriteHead(doc *frontmatter.Document, head []byte, f frontmatter.Fields) ([]byte, error) {
	format := doc.Format
	if format == frontmatter.FormatNone {
		format = frontmatter.FormatYAML
		doc = &frontmatter.Document{Format: format}
		head = nil
	}
	meta, err := doc.Update(f)
	if err != nil {
		return nil, err
	}
	fence := "---\n"
	if format == frontmatter.FormatTOML {
		fence = "+++\n"
	}
	var prefix []byte
	if bytes.HasPrefix(head, []byte{0xEF, 0xBB, 0xBF}) {
		prefix = head[:3]
	}
	// Follow the line ending of the opening fence line.
	if first, _, ok := bytes.Cut(head[len(prefix):], []byte("\n")); ok && bytes.HasSuffix(first, []byte("\r")) {
		fence = strings.TrimSuffix(fence, "\n") + "\r\n"
		meta = bytes.ReplaceAll(meta, []byte("\n"), []byte("\r\n"))
	}
	out := append([]byte{}, prefix...)
	out = append(out, fence...)
	out = append(out, meta...)
	out = append(out, fence...)
	return out, nil
}
