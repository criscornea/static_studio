package publish

import (
	"errors"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"

	"github.com/criscornea/static_studio/internal/build"
)

func waitDone(t *testing.T, j *Job) build.Event {
	t.Helper()
	history, ch, cancel := j.Events.Subscribe()
	defer cancel()
	var last build.Event
	for _, ev := range history {
		last = ev
	}
	if last.Kind == "done" {
		return last
	}
	timeout := time.After(10 * time.Second)
	for {
		select {
		case ev, ok := <-ch:
			if !ok {
				return last
			}
			last = ev
			if ev.Kind == "done" {
				return ev
			}
		case <-timeout:
			t.Fatal("timed out")
		}
	}
}

func TestJobBuildThenPush(t *testing.T) {
	requireGit(t)
	if runtime.GOOS == "windows" {
		t.Skip("uses sh")
	}
	remote := bareRepo(t)
	dir := t.TempDir()
	root := openRoot(t, dir)
	builder := &build.Runner{Bin: "sh", Args: []string{"-c", "mkdir -p public && echo hi > public/index.html"}}

	j := &Job{}
	if err := j.Start(builder, root, Settings{Remote: remote}); err != nil {
		t.Fatal(err)
	}
	if ev := waitDone(t, j); !ev.OK {
		t.Fatalf("done = %+v", ev)
	}
	if got := gitOut(t, remote, "show", "gh-pages:index.html"); strings.TrimSpace(got) != "hi" {
		t.Errorf("index = %q", got)
	}
	if _, err := os.Stat(filepath.Join(dir, ".git")); err == nil {
		t.Error("project got a .git folder")
	}
}

func TestJobBuildFailureSkipsPush(t *testing.T) {
	requireGit(t)
	if runtime.GOOS == "windows" {
		t.Skip("uses sh")
	}
	remote := bareRepo(t)
	builder := &build.Runner{Bin: "sh", Args: []string{"-c", "echo broken; exit 1"}}

	j := &Job{}
	if err := j.Start(builder, openRoot(t, t.TempDir()), Settings{Remote: remote}); err != nil {
		t.Fatal(err)
	}
	ev := waitDone(t, j)
	if ev.OK || !strings.Contains(ev.Error, "build failed") {
		t.Fatalf("done = %+v", ev)
	}
	if out := gitOut(t, remote, "branch", "--list"); strings.TrimSpace(out) != "" {
		t.Errorf("branches = %q", out)
	}
}

func TestJobInvalidSettingsAndBusy(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("uses sh")
	}
	j := &Job{}
	builder := &build.Runner{Bin: "sh", Args: []string{"-c", "sleep 5"}}
	if err := j.Start(builder, openRoot(t, t.TempDir()), Settings{}); !errors.Is(err, ErrInvalid) {
		t.Fatalf("err = %v", err)
	}
	if err := j.Start(builder, openRoot(t, t.TempDir()), Settings{Remote: "/srv/x.git"}); err != nil {
		t.Fatal(err)
	}
	if err := j.Start(builder, openRoot(t, t.TempDir()), Settings{Remote: "/srv/x.git"}); !errors.Is(err, ErrBusy) {
		t.Fatalf("second start err = %v", err)
	}
	j.Stop()
	if ev := waitDone(t, j); ev.OK || !strings.Contains(ev.Error, "cancelled") {
		t.Fatalf("done = %+v", ev)
	}
}
