package publish

import (
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func requireGit(t *testing.T) {
	t.Helper()
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git not installed")
	}
}

func bareRepo(t *testing.T) string {
	t.Helper()
	requireGit(t)
	dir := filepath.Join(t.TempDir(), "remote.git")
	if out, err := exec.Command("git", "init", "--bare", "--quiet", dir).CombinedOutput(); err != nil {
		t.Fatalf("%v: %s", err, out)
	}
	return dir
}

func gitOut(t *testing.T, dir string, args ...string) string {
	t.Helper()
	cmd := exec.Command("git", append([]string{"--git-dir", dir}, args...)...)
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("git %v: %v: %s", args, err, out)
	}
	return string(out)
}

// site writes a built site and returns the project root that holds it.
func site(t *testing.T, content string) *os.Root {
	t.Helper()
	proj := t.TempDir()
	pub := filepath.Join(proj, "public")
	if err := os.MkdirAll(filepath.Join(pub, "posts"), 0o700); err != nil {
		t.Fatal(err)
	}
	for name, data := range map[string]string{"index.html": content, "posts/a.html": "a"} {
		if err := os.WriteFile(filepath.Join(pub, name), []byte(data), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	return openRoot(t, proj)
}

func openRoot(t *testing.T, dir string) *os.Root {
	t.Helper()
	r, err := os.OpenRoot(dir)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = r.Close() })
	return r
}

func TestPushToBareRepo(t *testing.T) {
	remote := bareRepo(t)
	p := &Publisher{}
	var lines []string
	emit := func(l string) { lines = append(lines, l) }
	s := Settings{Remote: remote}

	if err := p.Push(t.Context(), site(t, "v1"), s, emit); err != nil {
		t.Fatal(err)
	}
	files := gitOut(t, remote, "ls-tree", "-r", "--name-only", "gh-pages")
	for _, want := range []string{"index.html", "posts/a.html", ".nojekyll"} {
		if !strings.Contains(files, want) {
			t.Errorf("missing %s in %q", want, files)
		}
	}

	// A second publish replaces the branch instead of failing as non-fast-forward.
	if err := p.Push(t.Context(), site(t, "v2"), s, emit); err != nil {
		t.Fatal(err)
	}
	if got := gitOut(t, remote, "show", "gh-pages:index.html"); got != "v2" {
		t.Errorf("index = %q", got)
	}
	if n := strings.TrimSpace(gitOut(t, remote, "rev-list", "--count", "gh-pages")); n != "1" {
		t.Errorf("commits = %s", n)
	}
	if len(lines) == 0 {
		t.Error("no progress emitted")
	}
}

func TestPushCustomBranch(t *testing.T) {
	remote := bareRepo(t)
	err := (&Publisher{}).Push(t.Context(), site(t, "x"), Settings{Remote: remote, Branch: "pages"}, func(string) {})
	if err != nil {
		t.Fatal(err)
	}
	gitOut(t, remote, "rev-parse", "--verify", "pages")
}

func TestPushErrors(t *testing.T) {
	requireGit(t)
	p := &Publisher{}
	emit := func(string) {}
	remote := filepath.Join(t.TempDir(), "missing.git")

	if err := p.Push(t.Context(), site(t, "x"), Settings{Remote: remote}, emit); err == nil || !strings.Contains(err.Error(), "git push failed") {
		t.Errorf("unreachable remote: %v", err)
	}
	if err := p.Push(t.Context(), openRoot(t, t.TempDir()), Settings{Remote: remote}, emit); err == nil {
		t.Error("missing public dir: want error")
	}
	emptyDir := t.TempDir()
	if err := os.Mkdir(filepath.Join(emptyDir, "public"), 0o700); err != nil {
		t.Fatal(err)
	}
	empty := openRoot(t, emptyDir)
	if err := p.Push(t.Context(), empty, Settings{Remote: remote}, emit); err == nil {
		t.Error("empty public dir: want error")
	}
	if err := p.Push(t.Context(), site(t, "x"), Settings{}, emit); err == nil {
		t.Error("no remote: want error")
	}
}

func TestPushSkipsSymlinks(t *testing.T) {
	remote := bareRepo(t)
	pub := site(t, "x")
	secret := filepath.Join(t.TempDir(), "secret")
	pubPath := filepath.Join(pub.Name(), "public")
	if err := os.WriteFile(secret, []byte("s3cret"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(secret, filepath.Join(pubPath, "leak.txt")); err != nil {
		t.Skip("symlinks unavailable")
	}
	if err := (&Publisher{}).Push(t.Context(), pub, Settings{Remote: remote}, func(string) {}); err != nil {
		t.Fatal(err)
	}
	if files := gitOut(t, remote, "ls-tree", "-r", "--name-only", "gh-pages"); strings.Contains(files, "leak.txt") {
		t.Errorf("symlink published: %q", files)
	}
}

func TestPushRefusesProjectBranch(t *testing.T) {
	remote := bareRepo(t)
	proj := site(t, "x")
	if err := os.MkdirAll(filepath.Join(proj.Name(), ".git"), 0o700); err != nil {
		t.Fatal(err)
	}
	head := filepath.Join(proj.Name(), ".git", "HEAD")
	if err := os.WriteFile(head, []byte("ref: refs/heads/site\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	err := (&Publisher{}).Push(t.Context(), proj, Settings{Remote: remote, Branch: "site"}, func(string) {})
	if err == nil || !strings.Contains(err.Error(), "branch your project is on") {
		t.Fatalf("err = %v", err)
	}
}

func TestPushRefusesMainBranch(t *testing.T) {
	remote := bareRepo(t)
	for _, b := range []string{"main", "master", "Main"} {
		err := (&Publisher{}).Push(t.Context(), site(t, "x"), Settings{Remote: remote, Branch: b}, func(string) {})
		if err == nil || !strings.Contains(err.Error(), "not allowed") {
			t.Errorf("%s: err = %v", b, err)
		}
	}
}

func addStatic(t *testing.T, root *os.Root, cname string) string {
	t.Helper()
	p := filepath.Join(root.Name(), "static", "CNAME")
	if err := os.MkdirAll(filepath.Dir(p), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(p, []byte(cname), 0o600); err != nil {
		t.Fatal(err)
	}
	return p
}

func TestPushShipsCNAMEForCustomDomain(t *testing.T) {
	remote := bareRepo(t)
	root := site(t, "x")
	static := addStatic(t, root, "www.example.com\n")
	s := Settings{Remote: remote, SiteKind: SiteCustom, CustomDomain: "www.example.com"}
	if err := (&Publisher{}).Push(t.Context(), root, s, func(string) {}); err != nil {
		t.Fatal(err)
	}
	if got := gitOut(t, remote, "show", "gh-pages:CNAME"); got != "www.example.com" {
		t.Errorf("CNAME = %q", got)
	}
	if files := gitOut(t, remote, "ls-tree", "-r", "--name-only", "gh-pages"); !strings.Contains(files, ".nojekyll") {
		t.Errorf("files = %q", files)
	}
	if got, _ := os.ReadFile(static); string(got) != "www.example.com\n" {
		t.Errorf("static/CNAME changed: %q", got)
	}
	if _, err := os.Stat(filepath.Join(root.Name(), "CNAME")); err == nil {
		t.Error("CNAME written into the project")
	}
}

func TestPushWithoutCustomDomainHasNoCNAME(t *testing.T) {
	remote := bareRepo(t)
	s := Settings{Remote: remote, SiteKind: SiteProject}
	if err := (&Publisher{}).Push(t.Context(), site(t, "x"), s, func(string) {}); err != nil {
		t.Fatal(err)
	}
	files := gitOut(t, remote, "ls-tree", "-r", "--name-only", "gh-pages")
	if strings.Contains(files, "CNAME") || !strings.Contains(files, ".nojekyll") {
		t.Errorf("files = %q", files)
	}
}

func TestPushCNAMEConflict(t *testing.T) {
	remote := bareRepo(t)
	root := site(t, "x")
	addStatic(t, root, "other.example.com")
	s := Settings{Remote: remote, SiteKind: SiteCustom, CustomDomain: "www.example.com"}
	err := (&Publisher{}).Push(t.Context(), root, s, func(string) {})
	if !errors.Is(err, ErrCNAMEConflict) {
		t.Fatalf("err = %v", err)
	}
	// Answered against the same file: the chosen static value is published.
	s.CNAMEChoice, s.CNAMEChoiceFor = CNAMEStatic, "other.example.com"
	if err := (&Publisher{}).Push(t.Context(), root, s, func(string) {}); err != nil {
		t.Fatal(err)
	}
	if got := gitOut(t, remote, "show", "gh-pages:CNAME"); got != "other.example.com" {
		t.Errorf("CNAME = %q", got)
	}
}
