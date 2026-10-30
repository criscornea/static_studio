package publish

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func fakeBin(t *testing.T, dir, name, script string) string {
	t.Helper()
	p := filepath.Join(dir, name)
	if err := os.WriteFile(p, []byte("#!/bin/sh\n"+script), 0o700); err != nil { //nolint:gosec // test script
		t.Fatal(err)
	}
	return p
}

func TestGHAPIArgvAndStdin(t *testing.T) {
	dir := t.TempDir()
	log := filepath.Join(dir, "log")
	bin := fakeBin(t, dir, "gh", `{ for a in "$@"; do echo "ARG:$a"; done; cat; } > `+log+`; echo '{"ok":true}'`)
	out, err := GH{Bin: bin}.API(context.Background(), "POST", "repos/o/r/pages", map[string]string{"a": "b"})
	if err != nil || !strings.Contains(string(out), "ok") {
		t.Fatalf("out=%s err=%v", out, err)
	}
	got, _ := os.ReadFile(log)
	want := "ARG:api\nARG:--hostname\nARG:github.com\nARG:--method\nARG:POST\nARG:repos/o/r/pages\nARG:--input\nARG:-\n{\"a\":\"b\"}"
	if string(got) != want {
		t.Fatalf("got %q want %q", got, want)
	}
}

func TestGHRefusesInjection(t *testing.T) {
	dir := t.TempDir()
	marker := filepath.Join(dir, "ran")
	bin := fakeBin(t, dir, "gh", "touch "+marker)
	for _, c := range [][2]string{{"o; rm -rf /", "r"}, {"o", "r$(id)"}, {"-x", "r"}, {"o", ".."}, {"o/x", "r"}, {"o", "r r"}, {"", "r"}, {"o-", "r"}} {
		if _, err := RepoPath(c[0], c[1]); err == nil {
			t.Errorf("accepted %q %q", c[0], c[1])
		}
	}
	g := GH{Bin: bin}
	for _, p := range []string{"--hostname=evil", "repos/../x", "repos/o r", "repos/o/r?x=1", "", "/repos//x"} {
		if _, err := g.API(context.Background(), "GET", p, nil); err == nil {
			t.Errorf("accepted path %q", p)
		}
	}
	if _, err := g.API(context.Background(), "GET; x", "repos/o/r", nil); err == nil {
		t.Error("accepted method")
	}
	if _, err := os.Stat(marker); err == nil {
		t.Fatal("gh ran for an invalid input")
	}
	if p, err := RepoPath("octo-cat", "my.repo_1", "pages"); err != nil || p != "repos/octo-cat/my.repo_1/pages" {
		t.Fatalf("%q %v", p, err)
	}
}

func TestGHErrorHasNoToken(t *testing.T) {
	dir := t.TempDir()
	const tok = "ghp_SECRET123"
	bin := fakeBin(t, dir, "gh", `echo "`+tok+`"; echo "HTTP 404" >&2; exit 1`)
	_, err := GH{Bin: bin}.API(context.Background(), "GET", "repos/o/r", nil)
	if err == nil || !strings.Contains(err.Error(), "404") || strings.Contains(err.Error(), tok) {
		t.Fatalf("err = %v", err)
	}
	// Preflight with a token-printing gh and ssh must not leak it either.
	sshBin := fakeBin(t, dir, "ssh", `echo "`+tok+`"; echo "`+tok+`" >&2; exit 255`)
	root, _ := os.OpenRoot(t.TempDir())
	defer func() { _ = root.Close() }()
	r := Preflight{Git: bin, GH: fakeBin(t, dir, "gh3", `echo "`+tok+`"; echo "`+tok+`" >&2; exit 1`), SSH: sshBin}.Run(context.Background(), root)
	if b, _ := json.Marshal(r); strings.Contains(string(b), tok) {
		t.Fatalf("report leaks token: %s", b)
	}
}

func TestOriginURL(t *testing.T) {
	dir := t.TempDir()
	if err := os.Mkdir(filepath.Join(dir, ".git"), 0o700); err != nil {
		t.Fatal(err)
	}
	cfg := "[core]\n\tbare = false\n[remote \"upstream\"]\n\turl = git@github.com:x/y.git\n[remote \"origin\"]\n\turl = git@github.com:me/site.git\n\tfetch = +refs/heads/*:refs/remotes/origin/*\n"
	if err := os.WriteFile(filepath.Join(dir, ".git", "config"), []byte(cfg), 0o600); err != nil {
		t.Fatal(err)
	}
	root, err := os.OpenRoot(dir)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = root.Close() }()
	if got := OriginURL(root); got != "git@github.com:me/site.git" {
		t.Fatalf("got %q", got)
	}
	empty, _ := os.OpenRoot(t.TempDir())
	defer func() { _ = empty.Close() }()
	if got := OriginURL(empty); got != "" {
		t.Fatalf("got %q", got)
	}
}

func byID(r Report, id string) Check {
	for _, c := range r.Checks {
		if c.ID == id {
			return c
		}
	}
	return Check{}
}

func TestPreflight(t *testing.T) {
	dir := t.TempDir()
	root, _ := os.OpenRoot(t.TempDir())
	defer func() { _ = root.Close() }()
	git := fakeBin(t, dir, "git", "true")
	sshLog := filepath.Join(dir, "sshlog")

	good := Preflight{Git: git,
		GH:  fakeBin(t, dir, "gh", "[ \"$1 $2 $3 $4\" = \"auth status --hostname github.com\" ]"),
		SSH: fakeBin(t, dir, "ssh", `echo "$*" > `+sshLog+`; echo "Hi me! You've successfully authenticated, but GitHub does not provide shell access."; exit 1`)}
	if r := good.Run(context.Background(), root); !r.OK || byID(r, "origin").OK {
		t.Fatalf("good = %+v", r)
	}

	if got, _ := os.ReadFile(sshLog); string(got) != "-T -o BatchMode=yes git@github.com\n" {
		t.Fatalf("ssh args = %q", got)
	}

	bad := Preflight{Git: filepath.Join(dir, "nogit"),
		GH:  fakeBin(t, dir, "gh2", "exit 1"),
		SSH: fakeBin(t, dir, "ssh2", `echo "git@github.com: Permission denied (publickey)."; exit 255`)}
	r := bad.Run(context.Background(), root)
	if r.OK || byID(r, "git").OK || byID(r, "gh-login").OK || byID(r, "ssh").OK {
		t.Fatalf("bad = %+v", r)
	}
	if !strings.Contains(byID(r, "gh-login").Fix, "gh auth login") || !strings.Contains(byID(r, "ssh").Fix, "SSH key") {
		t.Fatalf("fixes = %+v", r)
	}

	hk := Preflight{Git: git, GH: good.GH, SSH: fakeBin(t, dir, "ssh3", `echo "Host key verification failed."; exit 255`)}
	if f := byID(hk.Run(context.Background(), root), "ssh").Fix; !strings.Contains(f, "ssh -T git@github.com") {
		t.Fatalf("fix = %q", f)
	}
}

func TestParseOriginURL(t *testing.T) {
	for name, c := range map[string]struct{ cfg, want string }{
		"upper subsection ignored": {"[remote \"ORIGIN\"]\n\turl = a\n", ""},
		"upper section name":       {"[REMOTE \"origin\"]\n\turl = a\n", "a"},
		"hash comment":             {"[remote \"origin\"]\n\turl = git@github.com:a/b.git # mine\n", "git@github.com:a/b.git"},
		"semicolon comment":        {"[remote \"origin\"]\n\turl = https://github.com/a/b ; x\n", "https://github.com/a/b"},
		"quoted hash kept":         {"[remote \"origin\"]\n\turl = \"a#b\" # c\n", "a#b"},
	} {
		if got := parseOriginURL(strings.NewReader(c.cfg)); got != c.want {
			t.Errorf("%s: got %q want %q", name, got, c.want)
		}
	}
}

func TestRedactURL(t *testing.T) {
	for in, want := range map[string]string{
		"https://user:tok@github.com/a/b.git": "https://github.com/a/b.git",
		"https://tok@github.com/a/b.git":      "https://github.com/a/b.git",
		"user:tok@github.com:a/b.git":         "github.com:a/b.git",
		"git@github.com:a/b.git":              "git@github.com:a/b.git",
	} {
		if got := redactURL(in); got != want || strings.Contains(got, "tok") {
			t.Errorf("%q -> %q want %q", in, got, want)
		}
	}
	dir := t.TempDir()
	_ = os.Mkdir(filepath.Join(dir, ".git"), 0o700)
	_ = os.WriteFile(filepath.Join(dir, ".git", "config"), []byte("[remote \"origin\"]\n\turl = https://u:tok@github.com/a/b\n"), 0o600)
	root, _ := os.OpenRoot(dir)
	defer func() { _ = root.Close() }()
	r := Preflight{Git: "true", GH: "true", SSH: "true"}.Run(context.Background(), root)
	if d := byID(r, "origin").Detail; d != "https://github.com/a/b" {
		t.Fatalf("detail = %q", d)
	}
}

func TestOriginGitFile(t *testing.T) {
	open := func(t *testing.T, gitFile string, setup func(dir string)) (string, error) {
		dir := t.TempDir()
		if setup != nil {
			setup(dir)
		}
		if err := os.WriteFile(filepath.Join(dir, ".git"), []byte(gitFile), 0o600); err != nil {
			t.Fatal(err)
		}
		root, err := os.OpenRoot(dir)
		if err != nil {
			t.Fatal(err)
		}
		defer func() { _ = root.Close() }()
		return originLookup(root)
	}
	cfg := func(dir string) {
		_ = os.MkdirAll(filepath.Join(dir, "gd"), 0o700)
		_ = os.WriteFile(filepath.Join(dir, "gd", "config"), []byte("[remote \"origin\"]\n\turl = x\n"), 0o600)
	}
	// .git is written after setup, so gd is a sibling directory.
	if u, err := open(t, "gitdir: gd\n", cfg); err != nil || u != "x" {
		t.Fatalf("inside: %q %v", u, err)
	}
	for _, g := range []string{"gitdir: ../elsewhere\n", "gitdir: /etc\n", "garbage\n"} {
		if _, err := open(t, g, nil); err == nil {
			t.Errorf("%q: want error", g)
		}
	}
	// Report says "could not read", not "no origin", for an unsafe pointer.
	dir := t.TempDir()
	_ = os.WriteFile(filepath.Join(dir, ".git"), []byte("gitdir: ../x\n"), 0o600)
	root, _ := os.OpenRoot(dir)
	defer func() { _ = root.Close() }()
	r := Preflight{Git: "true", GH: "true", SSH: "true"}.Run(context.Background(), root)
	if d := byID(r, "origin").Detail; !strings.Contains(d, "Could not read") {
		t.Fatalf("detail = %q", d)
	}
}
