package publish

import (
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/criscornea/static_studio/internal/build"
)

func TestRsyncValidate(t *testing.T) {
	id := filepath.Join(t.TempDir(), "id_ed25519")
	if err := os.WriteFile(id, []byte("k"), 0o600); err != nil {
		t.Fatal(err)
	}
	good := []Settings{
		{Target: TargetRsync, RsyncDest: "deploy@example.com:/var/www/site"},
		{Target: TargetRsync, RsyncDest: "deploy@192.168.1.5:site/", RsyncPort: 2222, RsyncIdentity: id},
		{Target: TargetRsync, RsyncDest: "u@[2001:db8::1]:/srv/x", RsyncDelete: true},
		{Target: TargetRsync, RsyncDest: "/srv/out", RsyncDryRun: true},
		{Target: TargetRsync, RsyncDest: "example.com:site"},
	}
	for _, s := range good {
		if _, err := s.Validate(); err != nil {
			t.Errorf("%+v: %v", s, err)
		}
	}
	bad := []string{
		"", "-e", "-oProxyCommand=x@h:/p", "u@h:-rf", "u@h:/p; rm -rf /", "u@h:/p q", "u@h:/p\nx",
		"u@h:/$(id)", "u@h:/`id`", "u@h:/a|b", "u@h:/a&b", "u@h:/a>b", "u@h:/a*", "rel/path",
		"-u@h:/p", "u v@h:/p", "u;x@h:/p", "u@-h:/p", "u@h_x:/p", "u@h..x:/p", "u@[::1:/p", "u@[zz]:/p",
		"u@h", "u@h:", "u@:/p", "@h:/p", "u@h:/p:q", "/srv/a b", "/srv/$x", "u@h:~/x",
	}
	for _, d := range bad {
		if _, err := (Settings{Target: TargetRsync, RsyncDest: d}).Validate(); !errors.Is(err, ErrInvalid) {
			t.Errorf("dest %q accepted (err %v)", d, err)
		}
	}
	for _, port := range []int{-1, 65536, 100000} {
		if _, err := (Settings{Target: TargetRsync, RsyncDest: "u@h:/p", RsyncPort: port}).Validate(); !errors.Is(err, ErrInvalid) {
			t.Errorf("port %d accepted", port)
		}
	}
	for _, i := range []string{"relative/key", "-i", filepath.Dir(id), filepath.Join(t.TempDir(), "missing"), id + " x", id + ";x"} {
		if _, err := (Settings{Target: TargetRsync, RsyncDest: "u@h:/p", RsyncIdentity: i}).Validate(); !errors.Is(err, ErrInvalid) {
			t.Errorf("identity %q accepted", i)
		}
	}
	if _, err := (Settings{Target: "ftp"}).Validate(); !errors.Is(err, ErrInvalid) {
		t.Errorf("unknown target accepted")
	}
}

func TestMissingTargetMeansGitHubPages(t *testing.T) {
	got, err := Settings{Remote: "/srv/x.git"}.Validate()
	if err != nil || got.Target != TargetGitHubPages {
		t.Fatalf("got %+v, %v", got, err)
	}
	st := NewStore(nil, t.TempDir())
	if g := st.Get(t.TempDir()); g.Target != TargetGitHubPages {
		t.Fatalf("default target = %q", g.Target)
	}
}

func TestRsyncArgs(t *testing.T) {
	s := Settings{Target: TargetRsync, RsyncDest: "u@h:/srv/x", RsyncPort: 2200, RsyncIdentity: "/k/id", RsyncDelete: true, RsyncDryRun: true}
	got := strings.Join(rsyncArgs(s, "/p/public", "ssh"), "\x00")
	want := strings.Join([]string{
		"-rtzv", "--delete", "-n",
		"-e", "ssh -o BatchMode=yes -p 2200 -o IdentitiesOnly=yes -i /k/id",
		"--", "/p/public/", "u@h:/srv/x",
	}, "\x00")
	if got != want {
		t.Fatalf("argv\n got %q\nwant %q", got, want)
	}
	plain := rsyncArgs(Settings{RsyncDest: "/srv/out"}, "/p/public", "ssh")
	if strings.Join(plain, " ") != "-rtzv -- /p/public/ /srv/out" {
		t.Fatalf("local argv = %q", plain)
	}
	if strings.Contains(strings.Join(rsyncArgs(s, "/p", "ssh"), " "), "StrictHostKeyChecking") {
		t.Fatal("host key checking touched")
	}
}

// fakeRsync writes a script that records its argv, one per line, and then
// prints out and exits with code.
func fakeRsync(t *testing.T, out string, code int) (bin, argvFile string) {
	t.Helper()
	if runtime.GOOS == "windows" {
		t.Skip("uses sh")
	}
	dir := t.TempDir()
	argvFile = filepath.Join(dir, "argv")
	bin = filepath.Join(dir, "rsync")
	script := "#!/bin/sh\nfor a in \"$@\"; do printf '%s\\n' \"$a\" >> '" + argvFile + "'; done\n" +
		"cat <<'EOF'\n" + out + "\nEOF\nexit " + itoa(code) + "\n"
	if err := os.WriteFile(bin, []byte(script), 0o700); err != nil { //nolint:gosec // test script
		t.Fatal(err)
	}
	return bin, argvFile
}

func itoa(n int) string { return strconv.Itoa(n) }

func TestRsyncRunsFakeBinaryWithoutShell(t *testing.T) {
	bin, argvFile := fakeRsync(t, "sending incremental file list", 0)
	root := site(t, "x")
	p := &Publisher{RsyncBin: bin, SSH: bin}
	var lines []string
	s := Settings{Target: TargetRsync, RsyncDest: "u@example.com:/srv/x", RsyncPort: 22, RsyncDryRun: true}
	if err := p.Publish(t.Context(), root, s, func(l string) { lines = append(lines, l) }); err != nil {
		t.Fatal(err)
	}
	data, _ := os.ReadFile(argvFile)
	argv := strings.Split(strings.TrimSpace(string(data)), "\n")
	want := []string{"-rtzv", "-n", "-e", "" + bin + " -o BatchMode=yes -p 22", "--", filepath.Join(root.Name(), "public") + "/", "u@example.com:/srv/x"}
	if strings.Join(argv, "|") != strings.Join(want, "|") {
		t.Fatalf("argv = %q\nwant %q", argv, want)
	}
	joined := strings.Join(lines, "\n")
	if !strings.Contains(joined, "Dry run") || !strings.Contains(joined, "sending incremental file list") {
		t.Errorf("lines = %q", lines)
	}
}

func TestRsyncRefusesInjectionBeforeRunning(t *testing.T) {
	bin, argvFile := fakeRsync(t, "", 0)
	p := &Publisher{RsyncBin: bin, SSH: bin}
	for _, d := range []string{"-e sh@h:/p", "u@h:/p;touch x", "u@h:/$(id)"} {
		err := p.Publish(t.Context(), site(t, "x"), Settings{Target: TargetRsync, RsyncDest: d}, func(string) {})
		if !errors.Is(err, ErrInvalid) {
			t.Errorf("%q: err = %v", d, err)
		}
	}
	if _, err := os.Stat(argvFile); err == nil {
		t.Fatal("rsync ran")
	}
}

func TestRsyncFailuresAreReadable(t *testing.T) {
	cases := []struct{ out, want string }{
		{"Host key verification failed.", "ssh in a terminal"},
		{"@@@ WARNING: REMOTE HOST IDENTIFICATION HAS CHANGED! @@@", "ssh in a terminal"},
		{"u@h: Permission denied (publickey).", "refused the login"},
		{"ssh: Could not resolve hostname nope: Name or service not known", "host name could not be found"},
		{"ssh: connect to host h port 22: Connection refused", "could not be reached"},
		{"ssh: connect to host h port 22: Connection timed out", "could not be reached"},
		{"bash: rsync: command not found", "not installed on the server"},
		{"something else", "rsync failed"},
	}
	for _, c := range cases {
		bin, _ := fakeRsync(t, c.out, 255)
		p := &Publisher{RsyncBin: bin, SSH: bin}
		err := p.Publish(t.Context(), site(t, "x"), Settings{Target: TargetRsync, RsyncDest: "u@h:/p"}, func(string) {})
		if err == nil || !strings.Contains(err.Error(), c.want) {
			t.Errorf("%q: err = %v, want %q", c.out, err, c.want)
		}
	}
}

func TestRsyncMissingBinaries(t *testing.T) {
	root := site(t, "x")
	p := &Publisher{RsyncBin: filepath.Join(t.TempDir(), "nope")}
	err := p.Publish(t.Context(), root, Settings{Target: TargetRsync, RsyncDest: "/srv/out"}, func(string) {})
	if err == nil || !strings.Contains(err.Error(), "rsync is not installed") {
		t.Fatalf("err = %v", err)
	}
	bin, _ := fakeRsync(t, "", 0)
	p = &Publisher{RsyncBin: bin, SSH: filepath.Join(t.TempDir(), "nossh")}
	err = p.Publish(t.Context(), root, Settings{Target: TargetRsync, RsyncDest: "u@h:/p"}, func(string) {})
	if err == nil || !strings.Contains(err.Error(), "ssh is not installed") {
		t.Fatalf("err = %v", err)
	}
}

func TestRsyncRefusesEmptyAndMissingSite(t *testing.T) {
	bin, argvFile := fakeRsync(t, "", 0)
	p := &Publisher{RsyncBin: bin}
	s := Settings{Target: TargetRsync, RsyncDest: "/srv/out", RsyncDelete: true}
	if err := p.Publish(t.Context(), openRoot(t, t.TempDir()), s, func(string) {}); err == nil || !strings.Contains(err.Error(), "build first") {
		t.Fatalf("err = %v", err)
	}
	dir := t.TempDir()
	if err := os.Mkdir(filepath.Join(dir, "public"), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := p.Publish(t.Context(), openRoot(t, dir), s, func(string) {}); err == nil || !strings.Contains(err.Error(), "empty") {
		t.Fatalf("err = %v", err)
	}
	if _, err := os.Stat(argvFile); err == nil {
		t.Fatal("rsync ran")
	}
}

func TestRsyncCancelStopsProcess(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("uses sh")
	}
	dir := t.TempDir()
	pidFile := filepath.Join(dir, "pid")
	bin := filepath.Join(dir, "rsync")
	script := "#!/bin/sh\nsleep 30 &\necho $! > '" + pidFile + "'\nwait\n"
	if err := os.WriteFile(bin, []byte(script), 0o700); err != nil { //nolint:gosec // test script
		t.Fatal(err)
	}
	j := &Job{Publisher: Publisher{RsyncBin: bin}}
	s := Settings{Target: TargetRsync, RsyncDest: "/srv/out"}
	proj := t.TempDir()
	builder := &build.Runner{Bin: "sh", Args: []string{"-c", "mkdir -p public && echo hi > public/index.html"}}
	if err := j.Start(builder, openRoot(t, proj), s); err != nil {
		t.Fatal(err)
	}
	var pid string
	for range 200 {
		if b, err := os.ReadFile(pidFile); err == nil && strings.TrimSpace(string(b)) != "" {
			pid = strings.TrimSpace(string(b))
			break
		}
		time.Sleep(25 * time.Millisecond)
	}
	if pid == "" {
		t.Fatal("rsync never started")
	}
	j.Stop()
	if ev := waitDone(t, j); ev.OK || !strings.Contains(ev.Error, "cancelled") {
		t.Fatalf("done = %+v", ev)
	}
	for range 100 {
		if exec.Command("kill", "-0", pid).Run() != nil {
			return
		}
		time.Sleep(25 * time.Millisecond)
	}
	t.Fatalf("child %s still alive", pid)
}

func TestJobRsyncLocalPathEndToEnd(t *testing.T) {
	rs, err := exec.LookPath("rsync")
	if err != nil {
		t.Skip("rsync not installed")
	}
	dest := filepath.Join(t.TempDir(), "out")
	proj := t.TempDir()
	builder := &build.Runner{Bin: "sh", Args: []string{"-c", "mkdir -p public/posts && echo hi > public/index.html && echo a > public/posts/a.html && ln -sf /etc/passwd public/link"}}
	j := &Job{Publisher: Publisher{RsyncBin: rs}}
	if err := j.Start(builder, openRoot(t, proj), Settings{Target: TargetRsync, RsyncDest: dest, RsyncDryRun: true}); err != nil {
		t.Fatal(err)
	}
	if ev := waitDone(t, j); !ev.OK {
		t.Fatalf("done = %+v", ev)
	}
	if _, err := os.Stat(filepath.Join(dest, "index.html")); err == nil {
		t.Fatal("dry run wrote files")
	}
	if err := j.Start(builder, openRoot(t, proj), Settings{Target: TargetRsync, RsyncDest: dest}); err != nil {
		t.Fatal(err)
	}
	if ev := waitDone(t, j); !ev.OK {
		t.Fatalf("done = %+v", ev)
	}
	if b, _ := os.ReadFile(filepath.Join(dest, "posts", "a.html")); strings.TrimSpace(string(b)) != "a" {
		t.Errorf("a.html = %q", b)
	}
	if _, err := os.Lstat(filepath.Join(dest, "link")); err == nil {
		t.Error("symlink was copied")
	}
	// Delete is opt-in: an extra file stays without it and goes with it.
	extra := filepath.Join(dest, "stale.html")
	if err := os.WriteFile(extra, []byte("x"), 0o600); err != nil {
		t.Fatal(err)
	}
	for _, del := range []bool{false, true} {
		if err := j.Start(builder, openRoot(t, proj), Settings{Target: TargetRsync, RsyncDest: dest, RsyncDelete: del}); err != nil {
			t.Fatal(err)
		}
		if ev := waitDone(t, j); !ev.OK {
			t.Fatalf("done = %+v", ev)
		}
		_, err := os.Stat(extra)
		if del == (err == nil) {
			t.Errorf("delete=%v but stale exists=%v", del, err == nil)
		}
	}
}
