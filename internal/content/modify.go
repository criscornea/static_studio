package content

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path"
	"strings"
)

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

// RenamePage moves the page from to to, both paths inside contentDir with an
// editable extension. It never overwrites: ErrPageExists is returned if to is
// taken. Missing folders of to are created. It returns the new path.
func RenamePage(root *os.Root, contentDir, from, to string) (string, error) {
	if err := checkPagePath(contentDir, from); err != nil {
		return "", err
	}
	if err := checkPagePath(contentDir, to); err != nil {
		return "", err
	}
	for _, name := range []string{from, to} {
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
	// A hard link fails if to exists, so a race cannot overwrite anything.
	err := linkFile(root, from, to)
	switch {
	case err == nil:
		if err := root.Remove(from); err != nil {
			_ = root.Remove(to)
			return "", fmt.Errorf("removing %s: %w", from, err)
		}
	case errors.Is(err, fs.ErrExist):
		return "", ErrPageExists
	default:
		// No hard link support: check again, then rename.
		if _, statErr := root.Lstat(to); statErr == nil {
			return "", ErrPageExists
		}
		if err := root.Rename(from, to); err != nil {
			return "", fmt.Errorf("renaming %s: %w", from, err)
		}
	}
	syncDir(root, dir)
	syncDir(root, path.Dir(from))
	return to, nil
}

// DeletePage removes the page name: a plain file inside contentDir with an
// editable extension. Folders are never removed.
func DeletePage(root *os.Root, contentDir, name string) error {
	if err := checkPagePath(contentDir, name); err != nil {
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
	if err := root.Remove(name); err != nil {
		return fmt.Errorf("deleting %s: %w", name, err)
	}
	syncDir(root, path.Dir(name))
	return nil
}
