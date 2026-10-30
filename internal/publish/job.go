package publish

import (
	"context"
	"errors"
	"os"
	"sync"
	"time"

	"github.com/criscornea/static_studio/internal/build"
)

// ErrBusy is returned when a publish is already running.
var ErrBusy = errors.New("a publish is already running")

// Job runs one publish at a time: a site build, then a push. Output goes to
// Events and ends with a "done" event.
type Job struct {
	Publisher Publisher
	Events    build.Broadcaster

	mu      sync.Mutex
	running bool
	cancel  context.CancelFunc
	wg      sync.WaitGroup
}

// stopWait bounds how long Stop waits for the publish to clean up.
const stopWait = 5 * time.Second

// Running reports whether a publish is in progress.
func (j *Job) Running() bool {
	j.mu.Lock()
	defer j.mu.Unlock()

	return j.running
}

// Start builds the site in root with builder, then pushes root/public using
// s. All reads of the build output go through root. It returns at once, or ErrBusy if a publish is running. Settings are
// validated first so a mistake fails before any build starts.
func (j *Job) Start(builder *build.Runner, root *os.Root, s Settings) error {
	s, err := s.Validate()
	if err != nil {
		return err
	}
	if s.Target == TargetGitHubPages {
		if _, err := CheckSite(root, s); err != nil {
			return err
		}
	}

	j.mu.Lock()
	if j.running {
		j.mu.Unlock()
		return ErrBusy
	}
	j.running = true
	ctx, cancel := context.WithCancel(context.Background())
	j.cancel = cancel
	j.wg.Add(1)
	j.mu.Unlock()
	j.Events.Reset()

	go func() {
		defer j.wg.Done()
		defer func() {
			cancel()
			j.mu.Lock()
			j.running = false
			j.mu.Unlock()
		}()

		line := func(l string) { j.Events.Publish(build.Event{Kind: "line", Line: l}) }
		err := j.run(ctx, builder, root, s, line)
		done := build.Event{Kind: "done", OK: err == nil}
		if err != nil {
			done.Error = err.Error()
		}
		j.Events.Publish(done)
	}()

	return nil
}

func (j *Job) run(ctx context.Context, builder *build.Runner, root *os.Root, s Settings, line func(string)) error {
	line("Building the site...")
	if err := builder.Run(ctx, root.Name(), line); err != nil {
		switch {
		case errors.Is(err, build.ErrBusy):
			return errors.New("a build is already running; wait for it to finish")
		case ctx.Err() != nil:
			return errors.New("publishing was cancelled")
		}
		return errors.New("the site build failed, so nothing was published")
	}

	return j.Publisher.Publish(ctx, root, s, line)
}

// Stop cancels the running publish, if any, and waits a short while for it to
// finish so its temporary folder is removed.
func (j *Job) Stop() {
	j.mu.Lock()
	cancel := j.cancel
	j.mu.Unlock()

	if cancel != nil {
		cancel()
	}

	done := make(chan struct{})
	go func() {
		j.wg.Wait()
		close(done)
	}()
	select {
	case <-done:
	case <-time.After(stopWait):
	}
}
