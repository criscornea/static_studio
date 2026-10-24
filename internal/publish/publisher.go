package publish

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/criscornea/static_studio/internal/build"
)

// Publisher publishes built files: to a branch of a git remote, or with rsync.
type Publisher struct {
	// Git is the git binary. Empty means "git".
	Git string
	// RsyncBin and SSH are the rsync and ssh programs. Empty means the PATH names.
	RsyncBin string
	SSH      string
}

// gitEnv keeps git from waiting on a terminal that does not exist and limits
// the transports it may use. Credentials come from the user's own SSH agent
// or credential helper. When identity is true, commits get a fallback
// identity because the user has none configured.
func gitEnv(identity bool) []string {
	env := []string{"GIT_TERMINAL_PROMPT=0"}
	if identity {
		env = append(env,
			"GIT_AUTHOR_NAME=Static Studio", "GIT_AUTHOR_EMAIL=static-studio@localhost",
			"GIT_COMMITTER_NAME=Static Studio", "GIT_COMMITTER_EMAIL=static-studio@localhost")
	}
	if os.Getenv("GIT_ALLOW_PROTOCOL") == "" {
		env = append(env, "GIT_ALLOW_PROTOCOL=ssh:https:http:git:file")
	}
	if os.Getenv("GIT_SSH_COMMAND") == "" {
		env = append(env, "GIT_SSH_COMMAND=ssh -o BatchMode=yes")
	}
	return env
}

// currentBranch returns the branch checked out in the project repository, or
// "" when the project is not a git repository or HEAD is detached.
func currentBranch(root *os.Root) string {
	f, err := root.Open(".git/HEAD")
	if err != nil {
		return ""
	}
	defer func() { _ = f.Close() }()
	data, err := io.ReadAll(io.LimitReader(f, 1024))
	if err != nil {
		return ""
	}
	ref, ok := strings.CutPrefix(strings.TrimSpace(string(data)), "ref: refs/heads/")
	if !ok {
		return ""
	}
	return ref
}

// Push copies the regular files of root/public into a fresh temporary
// repository, commits them and force-pushes the commit to branch on remote.
// The branch holds only the built site, so its history is replaced on each
// publish. The project itself is never touched. Progress goes to emit.
func (p *Publisher) Push(ctx context.Context, root *os.Root, s Settings, emit func(string)) error {
	s, err := s.Validate()
	if err != nil {
		return err
	}
	if b := currentBranch(root); b != "" && b == s.Branch {
		return fmt.Errorf("%q is the branch your project is on; publishing would replace it. Use another branch such as %s", b, DefaultBranch)
	}
	public, err := root.OpenRoot("public")
	if err != nil {
		return errors.New("there is no built site to publish; build first")
	}
	defer func() { _ = public.Close() }()

	tmp, err := os.MkdirTemp("", "static-studio-publish-")
	if err != nil {
		return fmt.Errorf("creating temporary folder: %w", err)
	}
	defer func() { _ = os.RemoveAll(tmp) }()

	emit("Copying the built site...")
	n, err := copyFiles(public, tmp)
	if err != nil {
		return fmt.Errorf("copying the built site: %w", err)
	}
	if n == 0 {
		return errors.New("the built site is empty; nothing to publish")
	}
	// Keeps GitHub Pages from running its own Jekyll build over the output.
	if err := os.WriteFile(filepath.Join(tmp, ".nojekyll"), nil, 0o600); err != nil {
		return err
	}

	bin := p.Git
	if bin == "" {
		bin = "git"
	}
	// Without a configured identity git refuses to commit.
	idCmd := exec.CommandContext(ctx, bin, "config", "user.email") //nolint:gosec // bin is the configured git binary
	idOut, _ := idCmd.Output()
	env := gitEnv(len(bytes.TrimSpace(idOut)) == 0)

	run := func(args ...string) error {
		r := &build.Runner{Bin: bin, Args: args, Env: env}
		if err := r.Run(ctx, tmp, emit); err != nil {
			if ctx.Err() != nil {
				return errors.New("publishing was cancelled")
			}
			return fmt.Errorf("git %s failed: %w", subcommand(args), err)
		}
		return nil
	}

	emit(fmt.Sprintf("Publishing %d files to %s (%s)...", n, s.Remote, s.Branch))
	steps := [][]string{
		{"init", "--quiet"},
		{"checkout", "--quiet", "-b", "publish"},
		{"add", "--all"},
		{"-c", "commit.gpgsign=false", "commit", "--quiet", "-m", "Publish site"},
		{"push", "--force", "--", s.Remote, "HEAD:refs/heads/" + s.Branch},
	}
	for _, args := range steps {
		if err := run(args...); err != nil {
			return err
		}
	}
	emit("Published.")
	return nil
}

// copyFiles copies regular files from src to dst and returns how many. Links
// and other irregular files are skipped so nothing outside src is read.
func copyFiles(src *os.Root, dst string) (int, error) {
	n := 0
	err := fs.WalkDir(src.FS(), ".", func(p string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if p == "." {
			return nil
		}
		target := filepath.Join(dst, filepath.FromSlash(p))
		if d.IsDir() {
			if d.Name() == ".git" {
				return fs.SkipDir
			}
			return os.MkdirAll(target, 0o700)
		}
		if !d.Type().IsRegular() {
			return nil
		}
		if err := copyFile(src, p, target); err != nil {
			return err
		}
		n++
		return nil
	})
	return n, err
}

func copyFile(src *os.Root, name, dst string) error {
	in, err := src.Open(name)
	if err != nil {
		return err
	}
	defer func() { _ = in.Close() }()
	out, err := os.OpenFile(dst, os.O_CREATE|os.O_WRONLY|os.O_EXCL, 0o600) //nolint:gosec // inside our temp dir
	if err != nil {
		return err
	}
	if _, err := io.Copy(out, in); err != nil {
		_ = out.Close()
		return err
	}
	return out.Close()
}

// subcommand returns the git subcommand, skipping leading "-c key=value" pairs.
func subcommand(args []string) string {
	for len(args) > 2 && args[0] == "-c" {
		args = args[2:]
	}
	return args[0]
}
