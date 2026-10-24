package build

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"runtime"
	"slices"
	"testing"
	"time"
)

// fakeBin writes an executable shell script and returns its path.
func fakeBin(t *testing.T, script string) string {
	t.Helper()
	if runtime.GOOS == "windows" {
		t.Skip("shell script fake binary")
	}
	p := filepath.Join(t.TempDir(), "fake")
	if err := os.WriteFile(p, []byte("#!/bin/sh\n"+script), 0o700); err != nil { //nolint:gosec // test script must be executable
		t.Fatal(err)
	}
	return p
}

func TestRunOutputAndDir(t *testing.T) {
	dir := t.TempDir()
	r := &Runner{Bin: fakeBin(t, "echo one; echo two >&2; pwd")}

	var lines []string
	if err := r.Run(t.Context(), dir, func(l string) { lines = append(lines, l) }); err != nil {
		t.Fatal(err)
	}
	if len(lines) != 3 || lines[0] != "one" || lines[1] != "two" {
		t.Fatalf("lines = %q", lines)
	}
	if got, _ := filepath.EvalSymlinks(lines[2]); got != func() string { d, _ := filepath.EvalSymlinks(dir); return d }() {
		t.Errorf("cwd = %q, want %q", lines[2], dir)
	}
}

func TestRunFailure(t *testing.T) {
	r := &Runner{Bin: fakeBin(t, "echo boom; exit 3")}
	var lines []string
	err := r.Run(t.Context(), t.TempDir(), func(l string) { lines = append(lines, l) })
	if err == nil {
		t.Fatal("want error")
	}
	if !slices.Equal(lines, []string{"boom"}) {
		t.Errorf("lines = %q", lines)
	}
}

func TestRunMissingBinary(t *testing.T) {
	r := &Runner{Bin: filepath.Join(t.TempDir(), "nope")}
	if err := r.Run(t.Context(), t.TempDir(), func(string) {}); err == nil {
		t.Fatal("want error")
	}
	if r.Running() {
		t.Error("lock not released after start failure")
	}
}

func TestRunSingleBuildAndCancel(t *testing.T) {
	r := &Runner{Bin: fakeBin(t, "echo started; exec sleep 30")}
	ctx, cancel := context.WithCancel(t.Context())

	started := make(chan struct{})
	done := make(chan error, 1)
	go func() {
		done <- r.Run(ctx, t.TempDir(), func(string) {
			select {
			case started <- struct{}{}:
			default:
			}
		})
	}()
	<-started

	if err := r.Run(t.Context(), t.TempDir(), func(string) {}); !errors.Is(err, ErrBusy) {
		t.Fatalf("second run = %v, want ErrBusy", err)
	}

	cancel()
	select {
	case err := <-done:
		if !errors.Is(err, context.Canceled) {
			t.Fatalf("err = %v", err)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("cancel did not stop the build")
	}
	if r.Running() {
		t.Error("still running after cancel")
	}
}

func TestStartStreamsToEvents(t *testing.T) {
	r := &Runner{Bin: fakeBin(t, "echo hi")}
	if err := r.Start(t.TempDir()); err != nil {
		t.Fatal(err)
	}
	_, ch, cancel := r.Events.Subscribe()
	defer cancel()
	for range ch {
	}
	hist, _, _ := r.Events.Subscribe()
	if len(hist) != 2 || hist[0].Line != "hi" || !hist[1].OK {
		t.Fatalf("hist = %+v", hist)
	}
}

func TestStopCancelsStartedBuild(t *testing.T) {
	r := &Runner{Bin: fakeBin(t, "exec sleep 30")}
	if err := r.Start(t.TempDir()); err != nil {
		t.Fatal(err)
	}
	if err := r.Start(t.TempDir()); !errors.Is(err, ErrBusy) {
		t.Fatalf("err = %v", err)
	}
	_, ch, cancel := r.Events.Subscribe()
	defer cancel()
	r.Stop()
	timeout := time.After(5 * time.Second)
	for {
		select {
		case _, open := <-ch:
			if !open {
				hist, _, _ := r.Events.Subscribe()
				if last := hist[len(hist)-1]; last.Kind != "done" || last.OK {
					t.Fatalf("last = %+v", last)
				}
				return
			}
		case <-timeout:
			t.Fatal("stop did not end the build")
		}
	}
}

func TestRunLongLineKeepsReading(t *testing.T) {
	r := &Runner{Bin: fakeBin(t, "head -c 1200000 /dev/zero | tr '\\0' x; echo; echo after")}
	var lines []string
	if err := r.Run(t.Context(), t.TempDir(), func(l string) { lines = append(lines, l) }); err != nil {
		t.Fatal(err)
	}
	if !slices.Contains(lines, "output truncated: line too long") || lines[len(lines)-1] != "after" {
		t.Fatalf("lines = %d, last = %q", len(lines), lines[len(lines)-1])
	}
}
