package content

import (
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path"
)

// writeFileAtomic writes data to name inside root so that readers see either
// the old file or the complete new one, never a partial write. It writes a
// temp file in the same directory, syncs it, then renames it over name. The
// temp file is removed on any failure. Paths that escape root are refused by
// os.Root.
func writeFileAtomic(root *os.Root, name string, data []byte, perm os.FileMode) error {
	// Keep the mode of an existing target; use perm for a new file.
	if info, err := root.Stat(name); err == nil {
		perm = info.Mode().Perm()
	}
	tmp, err := writeTemp(root, name, data, perm)
	if err != nil {
		return err
	}
	if err := root.Rename(tmp, name); err != nil {
		_ = root.Remove(tmp)
		return fmt.Errorf("replacing %s: %w", name, err)
	}
	syncDir(root, path.Dir(name))
	return nil
}

// syncDir fsyncs a directory so a rename or link survives a crash. Best
// effort: errors (e.g. EINVAL on filesystems that cannot sync directories)
// are ignored.
func syncDir(root *os.Root, dir string) {
	d, err := root.Open(dir)
	if err != nil {
		return
	}
	_ = d.Sync()
	_ = d.Close()
}

// linkFile creates a hard link; a variable so tests can simulate filesystems
// without hard link support.
var linkFile = func(root *os.Root, oldname, newname string) error {
	return root.Link(oldname, newname)
}

// writeFileAtomicNew is like writeFileAtomic but never replaces an existing
// file: it hard-links the finished temp file to name, which fails with
// fs.ErrExist if name is taken. If the filesystem cannot hard-link, it falls
// back to creating name with O_EXCL (not atomic against crashes, but still
// never overwrites).
func writeFileAtomicNew(root *os.Root, name string, data []byte, perm os.FileMode) error {
	tmp, err := writeTemp(root, name, data, perm)
	if err != nil {
		return err
	}
	defer func() { _ = root.Remove(tmp) }()
	err = linkFile(root, tmp, name)
	if err == nil {
		syncDir(root, path.Dir(name))
		return nil
	}
	if errors.Is(err, fs.ErrExist) {
		return err
	}
	return writeExcl(root, name, data, perm)
}

// writeExcl creates name with O_EXCL and writes data, removing name on failure.
func writeExcl(root *os.Root, name string, data []byte, perm os.FileMode) error {
	f, err := root.OpenFile(name, os.O_WRONLY|os.O_CREATE|os.O_EXCL, perm)
	if err != nil {
		return err
	}
	_, err = f.Write(data)
	if err == nil {
		err = f.Sync()
	}
	if closeErr := f.Close(); err == nil {
		err = closeErr
	}
	if err != nil {
		_ = root.Remove(name)
		return fmt.Errorf("writing %s: %w", name, err)
	}
	syncDir(root, path.Dir(name))
	return nil
}

// writeTemp writes data to a fresh, synced temp file next to name and
// returns its path. On failure the temp file is already removed.
func writeTemp(root *os.Root, name string, data []byte, perm os.FileMode) (string, error) {
	var suffix [6]byte
	if _, err := rand.Read(suffix[:]); err != nil {
		return "", fmt.Errorf("temp name: %w", err)
	}
	base := path.Base(name)
	if len(base) > 64 {
		base = base[:64]
	}
	tmp := path.Join(path.Dir(name), "."+base+"."+hex.EncodeToString(suffix[:])+".tmp")

	f, err := root.OpenFile(tmp, os.O_WRONLY|os.O_CREATE|os.O_EXCL, perm)
	if err != nil {
		return "", fmt.Errorf("creating temp file for %s: %w", name, err)
	}
	_, err = f.Write(data)
	if err == nil {
		err = f.Sync()
	}
	if closeErr := f.Close(); err == nil {
		err = closeErr
	}
	if err != nil {
		_ = root.Remove(tmp)
		return "", fmt.Errorf("writing %s: %w", name, err)
	}
	return tmp, nil
}
