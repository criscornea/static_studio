package publish

import (
	"bufio"
	"bytes"
	"context"
	"errors"
	"io"
	"io/fs"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

// Check is the result of one preflight item. Fix is empty when OK.
type Check struct {
	ID     string `json:"id"`
	Label  string `json:"label"`
	OK     bool   `json:"ok"`
	Detail string `json:"detail,omitempty"`
	Fix    string `json:"fix,omitempty"`
}

// Report lists the checks GitHub Pages publishing needs.
type Report struct {
	Checks []Check `json:"checks"`
	// OK is true when every check passed.
	OK bool `json:"ok"`
}

// Preflight checks the tools and access publishing needs. Empty binaries mean
// the PATH names. Nothing is stored and no token is read.
type Preflight struct {
	Git, GH, SSH string
}

func orDefault(v, d string) string {
	if v == "" {
		return d
	}
	return v
}

// Run executes every check. root is the confined project root.
func (p Preflight) Run(ctx context.Context, root *os.Root) Report {
	ctx, cancel := context.WithTimeout(ctx, 20*time.Second)
	defer cancel()

	gitBin, ghBin, sshBin := orDefault(p.Git, "git"), orDefault(p.GH, "gh"), orDefault(p.SSH, "ssh")
	var checks []Check

	_, err := exec.LookPath(gitBin)
	checks = append(checks, Check{ID: "git", Label: "git is installed", OK: err == nil,
		Fix: fixIf(err, "Install git from https://git-scm.com/downloads.")})

	_, ghErr := exec.LookPath(ghBin)
	checks = append(checks, Check{ID: "gh", Label: "GitHub CLI (gh) is installed", OK: ghErr == nil,
		Fix: fixIf(ghErr, "Install the GitHub CLI from https://cli.github.com.")})

	login := Check{ID: "gh-login", Label: "gh is logged in to github.com"}
	if ghErr != nil {
		login.Fix = "Install gh first, then run `gh auth login` in a terminal."
	} else if err := runQuiet(ctx, ghBin, "auth", "status", "--hostname", "github.com"); err != nil {
		login.Fix = "Run `gh auth login` in a terminal."
	} else {
		login.OK = true
	}
	checks = append(checks, login)
	checks = append(checks, p.sshCheck(ctx, sshBin))

	origin := Check{ID: "origin", Label: "The project has an origin remote"}
	if u, err := originLookup(root); err != nil {
		origin.Detail = "Could not read the project's git configuration."
		origin.Fix = "Not needed to publish. Check that .git is a normal repository inside the project."
	} else if u != "" {
		origin.OK, origin.Detail = true, redactURL(u)
	} else {
		origin.Fix = "Not needed to publish. Creating a repo on GitHub will set it up for you."
	}
	checks = append(checks, origin)

	r := Report{Checks: checks, OK: true}
	for _, c := range checks {
		// The origin remote is informational: publishing can create it.
		if !c.OK && c.ID != "origin" {
			r.OK = false
		}
	}
	return r
}

const waitDelay = 2 * time.Second

// runQuiet runs a command with a bounded wait for its pipes after the context
// ends, so a child holding them open cannot block past the timeout.
func runQuiet(ctx context.Context, bin string, args ...string) error {
	cmd := exec.CommandContext(ctx, bin, args...) //nolint:gosec // configured binary, fixed argv
	cmd.WaitDelay = waitDelay
	return cmd.Run()
}

// redactURL drops credentials from a remote URL before it is shown.
func redactURL(s string) string {
	if strings.Contains(s, "://") {
		u, err := url.Parse(s)
		if err != nil {
			return "(unreadable url)"
		}
		u.User = nil
		return u.String()
	}
	// scp-style user[:secret]@host:path. A plain user such as git is kept.
	if at := strings.Index(s, "@"); at >= 0 && strings.Contains(s[:at], ":") {
		return s[at+1:]
	}
	return s
}

func fixIf(err error, fix string) string {
	if err != nil {
		return fix
	}
	return ""
}

func (p Preflight) sshCheck(ctx context.Context, sshBin string) Check {
	c := Check{ID: "ssh", Label: "SSH access to git@github.com"}
	if _, err := exec.LookPath(sshBin); err != nil {
		c.Fix = "Install OpenSSH so the app can reach GitHub over SSH."
		return c
	}
	var out bytes.Buffer
	cmd := exec.CommandContext(ctx, sshBin, "-T", "-o", "BatchMode=yes", "git@github.com") //nolint:gosec // configured binary, fixed argv
	cmd.Stdout, cmd.Stderr = &out, &out
	cmd.WaitDelay = waitDelay
	_ = cmd.Run() // GitHub exits 1 even on success; judge by the greeting.
	text := out.String()
	switch {
	case strings.Contains(text, "successfully authenticated"):
		c.OK = true
	case strings.Contains(text, "Host key verification failed"):
		c.Fix = "Connect once with `ssh -T git@github.com` in a terminal and accept the host key."
	case strings.Contains(text, "Permission denied"):
		c.Fix = "Add your SSH key to GitHub at https://github.com/settings/keys."
	default:
		c.Fix = "Connect once with `ssh -T git@github.com` and check your network and SSH key."
	}
	return c
}

// OriginURL returns the url of remote "origin" from the git config read
// through the confined root, or "" when there is none or it cannot be read.
// It never runs git.
func OriginURL(root *os.Root) string {
	u, _ := originLookup(root)
	return u
}

// originLookup returns the origin url, "" when there is no origin, and an
// error when the git config exists but cannot be read safely.
func originLookup(root *os.Root) (string, error) {
	cfg := ".git/config"
	if fi, err := root.Stat(".git"); err == nil && fi.Mode().IsRegular() {
		// Worktree or submodule: .git is a file "gitdir: <path>".
		b, err := fs.ReadFile(root.FS(), ".git")
		if err != nil {
			return "", err
		}
		dir, ok := strings.CutPrefix(strings.TrimSpace(string(b)), "gitdir:")
		if !ok {
			return "", errors.New("unrecognised .git file")
		}
		dir = strings.TrimSpace(dir)
		if filepath.IsAbs(dir) {
			rel, err := filepath.Rel(root.Name(), dir)
			if err != nil {
				return "", err
			}
			dir = rel
		}
		dir = filepath.Clean(dir)
		if dir == ".." || strings.HasPrefix(dir, ".."+string(filepath.Separator)) || filepath.IsAbs(dir) {
			return "", errors.New("gitdir points outside the project")
		}
		cfg = filepath.Join(dir, "config")
	}
	f, err := root.Open(cfg)
	if errors.Is(err, fs.ErrNotExist) {
		return "", nil
	}
	if err != nil {
		return "", err
	}
	defer func() { _ = f.Close() }()
	return parseOriginURL(io.LimitReader(f, 1<<20)), nil
}

// isOriginHeader reports whether a section header is [remote "origin"]: the
// section name is case-insensitive, the quoted subsection is exact.
func isOriginHeader(line string) bool {
	end := strings.Index(line, "]")
	if !strings.HasPrefix(line, "[") || end < 0 {
		return false
	}
	name, sub, ok := strings.Cut(strings.TrimSpace(line[1:end]), " ")
	return ok && strings.EqualFold(name, "remote") && strings.TrimSpace(sub) == `"origin"`
}

// stripComment removes a trailing # or ; comment that is not inside quotes.
func stripComment(v string) string {
	q := false
	for i := 0; i < len(v); i++ {
		switch v[i] {
		case '\\':
			i++
		case '"':
			q = !q
		case '#', ';':
			if !q {
				return v[:i]
			}
		}
	}
	return v
}

func parseOriginURL(r io.Reader) string {
	in := false
	sc := bufio.NewScanner(r)
	for sc.Scan() {
		line := strings.TrimSpace(sc.Text())
		if strings.HasPrefix(line, "[") {
			in = isOriginHeader(line)
			continue
		}
		if !in {
			continue
		}
		k, v, ok := strings.Cut(line, "=")
		if ok && strings.EqualFold(strings.TrimSpace(k), "url") {
			return strings.Trim(strings.TrimSpace(stripComment(v)), `"`)
		}
	}
	return ""
}
