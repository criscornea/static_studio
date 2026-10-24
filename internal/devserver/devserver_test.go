package devserver

import (
	"errors"
	"net"
	"os"
	"os/exec"
	"runtime"
	"slices"
	"strconv"
	"syscall"
	"testing"
	"time"
)

// TestMain doubles as the fake dev server: when STUDIO_FAKE_SERVER=1 is set
// it listens on 127.0.0.1:<port from the last arg> until killed.
func TestMain(m *testing.M) {
	if os.Getenv("STUDIO_FAKE_SERVER") == "1" {
		port := os.Args[len(os.Args)-1]
		if os.Getenv("STUDIO_FAKE_FORK") == "1" {
			// Hand the port to a grandchild and idle, like a wrapper process.
			child := exec.Command(os.Args[0], port)
			child.Env = append(os.Environ(), "STUDIO_FAKE_FORK=0")
			child.Stdout, child.Stderr = os.Stdout, os.Stderr
			if err := child.Start(); err != nil {
				os.Exit(4)
			}
			select {}
		}
		l, err := net.Listen("tcp", net.JoinHostPort(Host, port))
		if err != nil {
			os.Exit(2)
		}
		for {
			c, err := l.Accept()
			if err != nil {
				os.Exit(3)
			}
			_ = c.Close()
		}
	}
	os.Exit(m.Run())
}

func newFake(t *testing.T) *Server {
	t.Helper()
	if runtime.GOOS == "windows" {
		t.Skip("process group checks are unix only")
	}
	exe, err := os.Executable()
	if err != nil {
		t.Fatal(err)
	}
	s := &Server{
		Bin:  exe,
		Args: func(port int) []string { return []string{strconv.Itoa(port)} },
		Env:  []string{"STUDIO_FAKE_SERVER=1"},
	}
	t.Cleanup(s.Stop)

	return s
}

func waitReady(t *testing.T, s *Server) Status {
	t.Helper()
	deadline := time.Now().Add(10 * time.Second)
	for time.Now().Before(deadline) {
		if st := s.Status(); st.Ready {
			return st
		}
		time.Sleep(20 * time.Millisecond)
	}
	t.Fatal("server never became ready")

	return Status{}
}

func TestHugoArgsBindLoopback(t *testing.T) {
	args := HugoArgs(1234)
	i := slices.Index(args, "--bind")
	if i < 0 || args[i+1] != "127.0.0.1" {
		t.Fatalf("args = %q, want --bind 127.0.0.1", args)
	}
	if args[0] != "server" {
		t.Errorf("args = %q", args)
	}
	for _, want := range []string{"--buildDrafts", "--buildFuture"} {
		if !slices.Contains(args, want) {
			t.Errorf("args = %q, missing %s", args, want)
		}
	}
	i = slices.Index(args, "--baseURL")
	if i < 0 || args[i+1] != "http://127.0.0.1/" {
		t.Errorf("args = %q, want --baseURL http://127.0.0.1/", args)
	}
}

func TestStopKillsGrandchild(t *testing.T) {
	s := newFake(t)
	s.Env = append(s.Env, "STUDIO_FAKE_FORK=1")
	if err := s.Start(t.TempDir()); err != nil {
		t.Fatal(err)
	}
	waitReady(t, s)

	s.mu.Lock()
	port := s.port
	s.mu.Unlock()

	s.Stop()

	addr := net.JoinHostPort(Host, strconv.Itoa(port))
	deadline := time.Now().Add(3 * time.Second)
	for {
		c, err := net.DialTimeout("tcp", addr, 200*time.Millisecond)
		if err != nil {
			return
		}
		_ = c.Close()
		if time.Now().After(deadline) {
			t.Fatal("grandchild still holds the port after Stop")
		}
		time.Sleep(20 * time.Millisecond)
	}
}

func TestCrashKeepsOutput(t *testing.T) {
	s := &Server{
		Bin:  "sh",
		Args: func(int) []string { return []string{"-c", "echo boom; echo bad config >&2; exit 1"} },
	}
	if err := s.Start(t.TempDir()); err != nil {
		t.Fatal(err)
	}
	deadline := time.Now().Add(5 * time.Second)
	for s.Status().Running {
		if time.Now().After(deadline) {
			t.Fatal("still running")
		}
		time.Sleep(10 * time.Millisecond)
	}
	if out := s.Status().Output; len(out) != 2 {
		t.Fatalf("output = %q, want 2 lines", out)
	}
	s.Stop()
	if out := s.Status().Output; len(out) != 0 {
		t.Errorf("output after stop = %q", out)
	}
}

func TestLifecycle(t *testing.T) {
	s := newFake(t)

	if st := s.Status(); st.Running {
		t.Fatal("running before start")
	}
	if err := s.Start(t.TempDir()); err != nil {
		t.Fatal(err)
	}
	if err := s.Start(t.TempDir()); !errors.Is(err, ErrRunning) {
		t.Fatalf("second start = %v, want ErrRunning", err)
	}

	st := waitReady(t, s)
	host, _, err := net.SplitHostPort(st.URL[len("http://"):])
	if err != nil || host != "127.0.0.1" {
		t.Fatalf("url = %q", st.URL)
	}

	s.mu.Lock()
	pid, port := s.cmd.Process.Pid, s.port
	s.mu.Unlock()

	s.Stop()

	if st := s.Status(); st.Running {
		t.Error("running after stop")
	}
	if err := syscall.Kill(pid, 0); err == nil {
		t.Errorf("process %d still exists", pid)
	}
	if c, err := net.DialTimeout("tcp", net.JoinHostPort(Host, strconv.Itoa(port)), time.Second); err == nil {
		_ = c.Close()
		t.Error("port still accepting connections")
	}

	// A stopped server can start again.
	if err := s.Start(t.TempDir()); err != nil {
		t.Fatal(err)
	}
	waitReady(t, s)
}

func TestProcessExitClearsState(t *testing.T) {
	s := &Server{Bin: "true", Args: func(int) []string { return nil }}
	if err := s.Start(t.TempDir()); err != nil {
		t.Fatal(err)
	}
	deadline := time.Now().Add(5 * time.Second)
	for s.Status().Running {
		if time.Now().After(deadline) {
			t.Fatal("state not cleared after exit")
		}
		time.Sleep(10 * time.Millisecond)
	}
	s.Stop() // must not hang
}

func TestStartMissingBinary(t *testing.T) {
	s := &Server{Bin: "/nonexistent/hugo"}
	if err := s.Start(t.TempDir()); err == nil {
		t.Fatal("want error")
	}
	if s.Status().Running {
		t.Error("running after failed start")
	}
}
