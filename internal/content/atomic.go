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

// raceHook is called at named points between a check and the write that
// follows it. It is nil outside tests, which use it to simulate an external
// editor changing the file in that window.
var raceHook func(stage string)

// SetRaceHook installs fn as the race hook and returns a function that
// removes it. Tests in other packages use it to provoke races.
func SetRaceHook(fn func(stage string)) (restore func()) {
	raceHook = fn
	return func() { raceHook = nil }
}

func hook(stage string) {
	if raceHook != nil {
		raceHook(stage)
	}
}

// grabFile moves name atomically to a hidden sibling and returns that path.
// From then on nobody addressing name sees the file, so what is read from
// the grabbed copy is exactly what will be replaced or deleted. A missing
// name yields fs.ErrNotExist.
func grabFile(root *os.Root, name string) (string, error) {
	var suffix [6]byte
	if _, err := rand.Read(suffix[:]); err != nil {
		return "", fmt.Errorf("temp name: %w", err)
	}
	base := path.Base(name)
	if len(base) > 64 {
		base = base[:64]
	}
	grabbed := path.Join(path.Dir(name), "."+base+"."+hex.EncodeToString(suffix[:])+".grab")
	if err := root.Rename(name, grabbed); err != nil {
		return "", err
	}
	return grabbed, nil
}

// restoreFile puts a grabbed file back at name without overwriting. If name
// was recreated meanwhile, the grabbed copy is kept under a visible
// ".conflict" sibling so no data is lost, and ErrConflict is returned along
// with that path. Without hard link support it checks name, then renames.
func restoreFile(root *os.Root, grabbed, name string) (string, error) {
	err := linkFile(root, grabbed, name)
	switch {
	case err == nil:
		_ = root.Remove(grabbed)
		syncDir(root, path.Dir(name))
		return "", nil
	case errors.Is(err, fs.ErrExist):
		return keepConflict(root, grabbed, name)
	}
	if _, statErr := root.Lstat(name); statErr == nil {
		return keepConflict(root, grabbed, name)
	}
	if err := root.Rename(grabbed, name); err != nil {
		return "", fmt.Errorf("restoring %s: %w", name, err)
	}
	syncDir(root, path.Dir(name))
	return "", nil
}

func keepConflict(root *os.Root, grabbed, name string) (string, error) {
	var suffix [3]byte
	if _, err := rand.Read(suffix[:]); err != nil {
		return "", fmt.Errorf("temp name: %w", err)
	}
	kept := name + ".conflict-" + hex.EncodeToString(suffix[:])
	if err := root.Rename(grabbed, kept); err != nil {
		return "", fmt.Errorf("keeping %s: %w", grabbed, err)
	}
	return kept, ErrConflict
}

// verifyGrabbed reads the grabbed copy and compares its hash. On a mismatch
// (or read failure) it restores the file to name and returns ErrConflict.
func verifyGrabbed(root *os.Root, grabbed, name, hash string) ([]byte, error) {
	data, err := fs.ReadFile(root.FS(), grabbed)
	if err == nil && hashBytes(data) == hash {
		return data, nil
	}
	if _, rerr := restoreFile(root, grabbed, name); rerr != nil && !errors.Is(rerr, ErrConflict) {
		return nil, rerr
	}
	if err != nil {
		return nil, fmt.Errorf("reading %s: %w", name, err)
	}
	return nil, ErrConflict
}
