package publish

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os/exec"
	"regexp"
	"strings"
)

// GH talks to the GitHub API through the user's own gh CLI. The app never
// reads, logs, stores or returns the token: gh holds it.
type GH struct {
	// Bin is the gh program. Empty means "gh".
	Bin string
}

var (
	ownerRe = regexp.MustCompile(`^[A-Za-z0-9](?:[A-Za-z0-9-]{0,38})$`)
	repoRe  = regexp.MustCompile(`^[A-Za-z0-9_.-]{1,100}$`)
	apiRe   = regexp.MustCompile(`^[A-Za-z0-9_./-]+$`)
)

// ErrName is returned for an owner, repository or API path that is not a
// plain GitHub name.
var ErrName = errors.New("not a valid GitHub name")

// RepoPath builds "repos/<owner>/<repo>[/<suffix>]" after validating each
// segment, so user input never reaches the gh argv unchecked.
func RepoPath(owner, repo string, suffix ...string) (string, error) {
	if !ownerRe.MatchString(owner) || strings.HasSuffix(owner, "-") {
		return "", fmt.Errorf("%w: owner %q", ErrName, owner)
	}
	if !repoRe.MatchString(repo) || repo == "." || repo == ".." {
		return "", fmt.Errorf("%w: repository %q", ErrName, repo)
	}
	p := "repos/" + owner + "/" + repo
	for _, s := range suffix {
		p += "/" + s
	}
	if err := checkAPIPath(p); err != nil {
		return "", err
	}
	return p, nil
}

func checkAPIPath(p string) error {
	if p == "" || p[0] == '-' || !apiRe.MatchString(p) {
		return fmt.Errorf("%w: path %q", ErrName, p)
	}
	for _, seg := range strings.Split(p, "/") {
		if seg == ".." || seg == "" {
			return fmt.Errorf("%w: path %q", ErrName, p)
		}
	}
	return nil
}

func (g GH) bin() string {
	if g.Bin == "" {
		return "gh"
	}
	return g.Bin
}

// API runs `gh api --method <method> <path>` and returns the response body.
// A non-nil body is sent as JSON on stdin (--input -). method must be an
// upper-case HTTP verb.
func (g GH) API(ctx context.Context, method, path string, body any) ([]byte, error) {
	switch method {
	case "GET", "POST", "PUT", "PATCH", "DELETE":
	default:
		return nil, fmt.Errorf("%w: method %q", ErrName, method)
	}
	if err := checkAPIPath(path); err != nil {
		return nil, err
	}
	args := []string{"api", "--hostname", "github.com", "--method", method, path}
	cmd := exec.CommandContext(ctx, g.bin()) //nolint:gosec // argv only, path validated
	var in []byte
	if body != nil {
		var err error
		if in, err = json.Marshal(body); err != nil {
			return nil, err
		}
		args = append(args, "--input", "-")
		cmd.Stdin = bytes.NewReader(in)
	}
	cmd.Args = append(cmd.Args[:1], args...)
	var out, errb bytes.Buffer
	cmd.Stdout, cmd.Stderr = &out, &errb
	cmd.WaitDelay = waitDelay
	if err := cmd.Run(); err != nil {
		// gh prints the API error body on stdout and a message on stderr.
		msg := strings.TrimSpace(errb.String())
		if msg == "" {
			msg = err.Error()
		}
		if len(msg) > 500 {
			msg = msg[:500]
		}
		return out.Bytes(), fmt.Errorf("gh api %s %s: %s", method, path, msg)
	}
	return out.Bytes(), nil
}
