package publish

import (
	"context"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"sync"

	"github.com/criscornea/static_studio/internal/build"
)

// Publish dispatches by the target of s. Progress goes to emit.
func (p *Publisher) Publish(ctx context.Context, root *os.Root, s Settings, emit func(string)) error {
	s, err := s.Validate()
	if err != nil {
		return err
	}
	if s.Target == TargetRsync {
		return p.Rsync(ctx, root, s, emit)
	}
	return p.Push(ctx, root, s, emit)
}

// rsyncArgs builds the argv (without the program name) for validated s with
// src as the source directory. The flags work with rsync 2.6.9 and
// openrsync. Without -l, symlinks are skipped rather than followed or copied.
func rsyncArgs(s Settings, src, ssh string) []string {
	args := []string{"-rtzv"}
	if s.RsyncDelete {
		args = append(args, "--delete")
	}
	if s.RsyncDryRun {
		args = append(args, "-n")
	}
	if IsRemote(s.RsyncDest) {
		e := ssh + " -o BatchMode=yes"
		if s.RsyncPort > 0 {
			e += " -p " + strconv.Itoa(s.RsyncPort)
		}
		if s.RsyncIdentity != "" {
			e += " -o IdentitiesOnly=yes -i " + s.RsyncIdentity
		}
		args = append(args, "-e", e)
	}
	return append(args, "--", src+"/", s.RsyncDest)
}

// Rsync copies root/public to the destination in s using the system rsync,
// which runs ssh for remote destinations. The argv is built without a shell.
func (p *Publisher) Rsync(ctx context.Context, root *os.Root, s Settings, emit func(string)) error {
	s, err := s.Validate()
	if err != nil {
		return err
	}
	if !IsRemote(s.RsyncDest) {
		if err := checkLocalDest(s.RsyncDest, root.Name()); err != nil {
			return err
		}
	}
	rsyncBin, sshBin := p.RsyncBin, p.SSH
	if rsyncBin == "" {
		rsyncBin = "rsync"
	}
	if sshBin == "" {
		sshBin = "ssh"
	}
	rsyncPath, err := exec.LookPath(rsyncBin)
	if err != nil {
		return errors.New("rsync is not installed or not on the PATH; install it and try again")
	}
	if IsRemote(s.RsyncDest) {
		if _, err := exec.LookPath(sshBin); err != nil {
			return errors.New("ssh is not installed or not on the PATH; install it and try again")
		}
	}

	public, err := root.OpenRoot("public")
	if err != nil {
		return errors.New("there is no built site to publish; build first")
	}
	defer func() { _ = public.Close() }()
	entries, err := fs.ReadDir(public.FS(), ".")
	if err != nil || len(entries) == 0 {
		return errors.New("the built site is empty; nothing to publish")
	}

	if s.RsyncDryRun {
		emit("Dry run: nothing will be changed on the destination.")
	}
	emit("Publishing with rsync to " + s.RsyncDest + "...")

	var mu sync.Mutex
	var tail []string
	capture := func(l string) {
		mu.Lock()
		tail = append(tail, l)
		if len(tail) > 50 {
			tail = tail[1:]
		}
		mu.Unlock()
		emit(l)
	}
	r := &build.Runner{Bin: rsyncPath, Args: rsyncArgs(s, public.Name(), sshBin)}
	if err := r.Run(ctx, root.Name(), capture); err != nil {
		if ctx.Err() != nil {
			return errors.New("publishing was cancelled")
		}
		mu.Lock()
		defer mu.Unlock()
		return rsyncFailure(tail, err)
	}
	if s.RsyncDryRun {
		emit("Dry run finished. Nothing was changed.")
	} else {
		emit("Published.")
	}
	return nil
}

// rsyncFailure turns the tail of rsync/ssh output into a readable error.
func rsyncFailure(lines []string, cause error) error {
	text := strings.ToLower(strings.Join(lines, "\n"))
	has := func(subs ...string) bool {
		for _, s := range subs {
			if strings.Contains(text, s) {
				return true
			}
		}
		return false
	}
	switch {
	case has("host key verification failed", "remote host identification has changed", "no matching host key"):
		return errors.New("the server's host key is unknown or has changed. Connect once with ssh in a terminal to check and accept it, then publish again")
	case has("permission denied (publickey", "permission denied, please try again", "authentication failed", "too many authentication failures"):
		return errors.New("the server refused the login. Check the user name and that your SSH key or agent is set up for it")
	case has("could not resolve hostname", "name or service not known", "nodename nor servname"):
		return errors.New("the host name could not be found. Check the destination")
	case has("connection refused", "connection timed out", "no route to host", "network is unreachable", "operation timed out", "connection closed by"):
		return errors.New("the server could not be reached. Check the host, port and your network")
	case has("rsync: command not found", "rsync: not found"):
		return errors.New("rsync is not installed on the server")
	}
	last := ""
	for i := len(lines) - 1; i >= 0; i-- {
		if t := strings.TrimSpace(lines[i]); t != "" {
			last = t
			break
		}
	}
	if last != "" {
		return fmt.Errorf("rsync failed: %s (%w)", last, cause)
	}
	return fmt.Errorf("rsync failed: %w", cause)
}

// checkLocalDest refuses a local destination that is, contains or sits inside
// the project root: with --delete rsync would remove project files or
// unrelated data.
func checkLocalDest(dest, projectRoot string) error {
	resolve := func(p string) string {
		p = filepath.Clean(p)
		// Resolve the longest existing prefix so symlinks cannot hide an overlap.
		rest := ""
		for q := p; ; q = filepath.Dir(q) {
			if r, err := filepath.EvalSymlinks(q); err == nil {
				return filepath.Join(r, rest)
			}
			if filepath.Dir(q) == q {
				return p
			}
			rest = filepath.Join(filepath.Base(q), rest)
		}
	}
	d, r := resolve(dest), resolve(projectRoot)
	if d == filepath.Dir(d) {
		return fmt.Errorf("%w: the destination may not be the file system root", ErrInvalid)
	}
	if within(d, r) || within(r, d) {
		return fmt.Errorf("%w: a local destination may not be the project folder, a folder containing it, or inside it", ErrInvalid)
	}
	return nil
}

// within reports whether path is equal to or below base.
func within(path, base string) bool {
	rel, err := filepath.Rel(base, path)
	return err == nil && rel != ".." && !strings.HasPrefix(rel, ".."+string(filepath.Separator))
}
