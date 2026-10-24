// Package build runs the site generator and streams its output.
package build

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"io"
	"os/exec"
	"slices"
	"sync"
	"time"
)

// ErrBusy is returned when a build is already running.
var ErrBusy = errors.New("a build is already running")

// maxLine bounds one output line so a runaway program cannot exhaust memory.
const maxLine = 1 << 20

// maxProblems bounds how many parsed errors one build reports.
const maxProblems = 100

// waitDelay bounds how long Wait lingers for output after the process ends or
// is killed, for example when a grandchild still holds the pipe.
const waitDelay = 5 * time.Second

// Runner runs one external build command at a time.
type Runner struct {
	// Bin is the command to run, for example "hugo".
	Bin  string
	Args []string

	// Events receives the output of builds begun with Start.
	Events Broadcaster

	mu      sync.Mutex
	running bool
	cancel  context.CancelFunc
}

// Running reports whether a build is in progress.
func (r *Runner) Running() bool {
	r.mu.Lock()
	defer r.mu.Unlock()

	return r.running
}

func (r *Runner) acquire() bool {
	r.mu.Lock()
	defer r.mu.Unlock()

	if r.running {
		return false
	}
	r.running = true

	return true
}

func (r *Runner) release() {
	r.mu.Lock()
	r.running = false
	r.mu.Unlock()
}

// Run executes the command in dir and calls emit for each line of combined
// stdout and stderr. It blocks until the command exits. Cancelling ctx kills
// the process. A second concurrent Run returns ErrBusy without starting.
func (r *Runner) Run(ctx context.Context, dir string, emit func(line string)) error {
	if !r.acquire() {
		return ErrBusy
	}
	defer r.release()

	return r.run(ctx, dir, emit)
}

// Start begins a build in the background and returns at once, or ErrBusy if
// one is running. Output goes to Events, which ends with a "done" event. Stop
// cancels the build.
func (r *Runner) Start(dir string) error {
	if !r.acquire() {
		return ErrBusy
	}

	ctx, cancel := context.WithCancel(context.Background())
	r.mu.Lock()
	r.cancel = cancel
	r.mu.Unlock()
	r.Events.Reset()

	go func() {
		defer r.release()
		defer cancel()

		var problems []Problem
		err := r.run(ctx, dir, func(l string) {
			r.Events.Publish(Event{Kind: "line", Line: l})
			if p, ok := ParseLine(dir, l); ok && len(problems) < maxProblems && !slices.Contains(problems, p) {
				problems = append(problems, p)
			}
		})
		done := Event{Kind: "done", OK: err == nil, Problems: problems}
		if err != nil {
			done.Error = err.Error()
		}
		r.Events.Publish(done)
	}()

	return nil
}

// Stop cancels the running build, if any.
func (r *Runner) Stop() {
	r.mu.Lock()
	cancel := r.cancel
	r.mu.Unlock()

	if cancel != nil {
		cancel()
	}
}

func (r *Runner) run(ctx context.Context, dir string, emit func(line string)) error {
	cmd := exec.CommandContext(ctx, r.Bin, r.Args...) //nolint:gosec // Bin is set by the program, never by a request
	cmd.Dir = dir
	cmd.WaitDelay = waitDelay
	setProcessGroup(cmd)

	pr, pw := io.Pipe()
	cmd.Stdout = pw
	cmd.Stderr = pw

	if err := cmd.Start(); err != nil {
		return fmt.Errorf("starting %s: %w", r.Bin, err)
	}

	scanned := make(chan struct{})
	go func() {
		defer close(scanned)

		for {
			sc := bufio.NewScanner(pr)
			sc.Buffer(make([]byte, 0, 64<<10), maxLine)
			for sc.Scan() {
				emit(sc.Text())
			}
			if !errors.Is(sc.Err(), bufio.ErrTooLong) {
				break
			}
			// The oversized line is skipped; resume with a fresh scanner.
			emit("output truncated: line too long")
		}
		// Keep draining so the process never blocks on a full pipe.
		_, _ = io.Copy(io.Discard, pr)
	}()

	err := cmd.Wait()
	_ = pw.Close()
	<-scanned

	if ctx.Err() != nil {
		return ctx.Err()
	}

	return err
}
