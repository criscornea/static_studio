// Package devserver runs one `hugo server` bound to 127.0.0.1 and maps
// content paths to the URLs it serves them at.
package devserver

import (
	"context"
	"errors"
	"fmt"
	"net"
	"os/exec"
	"slices"
	"strconv"
	"strings"
	"sync"
	"time"
)

// ErrRunning is returned when the dev server is already running.
var ErrRunning = errors.New("the dev server is already running")

// Host is the only address the dev server binds to.
const Host = "127.0.0.1"

// waitDelay bounds how long Stop waits for a killed process to finish.
const waitDelay = 5 * time.Second

// Status describes the dev server at one moment.
type Status struct {
	Running bool `json:"running"`
	// Ready is true once the port accepts connections.
	Ready bool `json:"ready"`
	// URL is the origin the server answers on, without a trailing slash.
	URL string `json:"url,omitempty"`
	// Output holds the last lines the process printed. It is only set when
	// the process exited on its own, until the next start or stop.
	Output []string `json:"output,omitempty"`
}

// tailLines is how many output lines are kept for a crashed server.
const tailLines = 20

// tail keeps the last tailLines complete lines written to it.
type tail struct {
	mu      sync.Mutex
	lines   []string
	partial string
}

func (t *tail) Write(p []byte) (int, error) {
	t.mu.Lock()
	defer t.mu.Unlock()

	t.partial += string(p)
	for {
		i := strings.IndexByte(t.partial, '\n')
		if i < 0 {
			break
		}
		t.lines = append(t.lines, strings.TrimRight(t.partial[:i], "\r"))
		t.partial = t.partial[i+1:]
	}
	if len(t.lines) > tailLines {
		t.lines = t.lines[len(t.lines)-tailLines:]
	}

	return len(p), nil
}

func (t *tail) snapshot() []string {
	t.mu.Lock()
	defer t.mu.Unlock()

	out := slices.Clone(t.lines)
	if t.partial != "" {
		out = append(out, t.partial)
	}
	if len(out) > tailLines {
		out = out[len(out)-tailLines:]
	}

	return out
}

// Server manages at most one dev server process.
type Server struct {
	// Bin is the command to run, for example "hugo".
	Bin string
	// Args returns the arguments for a server on the given port. Nil means
	// the Hugo defaults, see HugoArgs.
	Args func(port int) []string
	// Env, if set, is added to the process environment. Used by tests.
	Env []string

	mu     sync.Mutex
	cmd    *exec.Cmd
	port   int
	cancel context.CancelFunc
	done   chan struct{} // closed when the current process has been reaped
	// crashed holds the output of a process that exited on its own.
	crashed []string
}

// HugoArgs returns the arguments that start Hugo on 127.0.0.1:port. Drafts
// and future-dated pages are built so new pages can be previewed. The base
// URL is pinned to the loopback host (Hugo appends the port), so a path in
// the site's configured baseURL does not move pages.
func HugoArgs(port int) []string {
	return []string{
		"server", "--bind", Host, "--port", strconv.Itoa(port),
		"--baseURL", "http://" + Host + "/",
		"--buildDrafts", "--buildFuture",
	}
}

// Start launches the server in dir on a free port. It returns ErrRunning if
// one is already running.
func (s *Server) Start(dir string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.running() {
		return ErrRunning
	}

	port, err := freePort()
	if err != nil {
		return err
	}

	args := HugoArgs
	if s.Args != nil {
		args = s.Args
	}

	ctx, cancel := context.WithCancel(context.Background())
	cmd := exec.CommandContext(ctx, s.Bin, args(port)...) //nolint:gosec // Bin is set by the program, never by a request
	cmd.Dir = dir
	cmd.WaitDelay = waitDelay
	cmd.Env = append(cmd.Environ(), s.Env...)
	out := &tail{}
	cmd.Stdout, cmd.Stderr = out, out
	setProcessGroup(cmd)

	if err := cmd.Start(); err != nil {
		cancel()

		return fmt.Errorf("starting %s: %w", s.Bin, err)
	}

	done := make(chan struct{})
	s.cmd, s.port, s.cancel, s.done, s.crashed = cmd, port, cancel, done, nil

	go func() {
		_ = cmd.Wait()
		onItsOwn := ctx.Err() == nil
		cancel()

		s.mu.Lock()
		if s.cmd == cmd {
			s.cmd, s.port, s.cancel, s.done = nil, 0, nil, nil
			if onItsOwn {
				s.crashed = out.snapshot()
			}
		}
		s.mu.Unlock()
		close(done)
	}()

	return nil
}

// Stop kills the process group and waits until it is gone. It is safe to
// call when nothing runs.
func (s *Server) Stop() {
	s.mu.Lock()
	cancel, done := s.cancel, s.done
	s.crashed = nil
	s.mu.Unlock()

	if cancel == nil {
		return
	}
	cancel()
	<-done
}

// Status reports the current state.
func (s *Server) Status() Status {
	s.mu.Lock()
	port := s.port
	running := s.running()
	crashed := s.crashed
	s.mu.Unlock()

	if !running {
		return Status{Output: crashed}
	}

	addr := net.JoinHostPort(Host, strconv.Itoa(port))
	st := Status{Running: true, URL: "http://" + addr}
	if c, err := net.DialTimeout("tcp", addr, 200*time.Millisecond); err == nil {
		_ = c.Close()
		st.Ready = true
	}

	return st
}

// running requires s.mu.
func (s *Server) running() bool { return s.cmd != nil }

// freePort asks the kernel for an unused port on the loopback interface.
func freePort() (int, error) {
	l, err := net.Listen("tcp", net.JoinHostPort(Host, "0"))
	if err != nil {
		return 0, fmt.Errorf("finding a free port: %w", err)
	}
	defer func() { _ = l.Close() }()

	return l.Addr().(*net.TCPAddr).Port, nil //nolint:forcetypeassert // a tcp listener always has a TCPAddr
}
