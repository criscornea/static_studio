// Command studio runs the static_studio server: a local HTTP API and UI
// for editing SSG projeccts.
package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"io"
	"log/slog"
	"net"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/criscornea/static_studio/internal/publish"
	"github.com/criscornea/static_studio/internal/recent"
	"github.com/criscornea/static_studio/internal/server"
	"github.com/criscornea/static_studio/web"
)

func main() {
	if err := run(); err != nil {
		slog.Error("fatal", "err", err)
		os.Exit(1)
	}
}

func run() error {
	addr := flag.String("addr", "127.0.0.1:7777", "address to listen on")
	logLevel := flag.String("log-level", "info", "log level: debug, info, warn or error")
	logFormat := flag.String("log-format", "text", "log format: text or json")
	flag.Parse()

	logger, err := newLogger(os.Stderr, *logLevel, *logFormat)
	if err != nil {
		return err
	}
	slog.SetDefault(logger)

	app := server.New(logger, web.Assets())
	app.UseRecent(recent.NewDefault(logger))
	app.UsePublishSettings(publish.NewDefaultStore(logger))
	// Idempotent; also covers the listen-error path, which skips the signal branch.
	defer app.Close()
	baseCtx, cancelBase := context.WithCancel(context.Background())
	defer cancelBase()

	srv := &http.Server{
		Addr:         *addr,
		Handler:      app.Routes(),
		BaseContext:  func(net.Listener) context.Context { return baseCtx },
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 10 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	errCh := make(chan error, 1)
	go func() {
		slog.Info("listening", "addr", srv.Addr)
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			errCh <- err
		}
	}()

	select {
	case err := <-errCh:
		return err
	case <-ctx.Done():
		slog.Info("shutting down")
		// End the build, dev server and open streams first so Shutdown does not wait on them.
		app.Close()
		cancelBase()
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		return srv.Shutdown(shutdownCtx)
	}
}

// newLogger builds a slog logger for the given level and format.
func newLogger(w io.Writer, level, format string) (*slog.Logger, error) {
	var lvl slog.Level
	if err := lvl.UnmarshalText([]byte(level)); err != nil {
		return nil, fmt.Errorf("invalid -log-level %q: want debug, info, warn or error", level)
	}
	opts := &slog.HandlerOptions{Level: lvl}
	switch format {
	case "text":
		return slog.New(slog.NewTextHandler(w, opts)), nil
	case "json":
		return slog.New(slog.NewJSONHandler(w, opts)), nil
	default:
		return nil, fmt.Errorf("invalid -log-format %q: want text or json", format)
	}
}
