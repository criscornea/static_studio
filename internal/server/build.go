package server

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"time"

	"github.com/criscornea/static_studio/internal/build"
)

func (s *Server) handleBuild(w http.ResponseWriter, r *http.Request) {
	open, ok := s.requireProject(w, r)
	if !ok {
		return
	}

	switch err := s.build.Start(open.Info.Root); {
	case err == nil:
		s.log.Info("build started", "root", open.Info.Root)
		s.writeJSON(w, http.StatusAccepted, map[string]bool{"started": true})

	case errors.Is(err, build.ErrBusy):
		s.writeError(w, http.StatusConflict, "build_running",
			"A build is already running.")

	default:
		s.log.Warn("starting build failed", "err", err)
		s.writeError(w, http.StatusInternalServerError, "internal",
			"Something went wrong. Please try again.")
	}
}

func (s *Server) handleBuildCancel(w http.ResponseWriter, r *http.Request) {
	if _, ok := s.requireProject(w, r); !ok {
		return
	}

	s.build.Stop()
	w.WriteHeader(http.StatusNoContent)
}

// handleBuildStream sends the current (or last) build's output as server-sent
// events: the log so far, then live lines, ending after the "done" event.
func (s *Server) handleBuildStream(w http.ResponseWriter, r *http.Request) {
	if _, ok := s.requireProject(w, r); !ok {
		return
	}

	flusher, ok := w.(http.Flusher)
	if !ok {
		s.writeError(w, http.StatusInternalServerError, "internal",
			"Something went wrong. Please try again.")
		return
	}

	// The server's WriteTimeout would cut a long build stream short.
	if err := http.NewResponseController(w).SetWriteDeadline(time.Time{}); err != nil {
		s.log.Debug("clearing write deadline failed", "err", err)
	}

	history, live, cancel := s.build.Events.Subscribe()
	defer cancel()

	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.WriteHeader(http.StatusOK)

	send := func(ev build.Event) bool {
		data, err := json.Marshal(ev)
		if err != nil {
			return false
		}
		if _, err := fmt.Fprintf(w, "data: %s\n\n", data); err != nil {
			return false
		}
		flusher.Flush()

		return true
	}

	for _, ev := range history {
		if !send(ev) {
			return
		}
	}
	// Nothing is running and nothing was recorded: there is no stream to wait for.
	if !s.build.Running() && len(history) == 0 {
		return
	}

	for {
		select {
		case <-r.Context().Done():
			return
		case ev, open := <-live:
			if !open || !send(ev) {
				return
			}
		}
	}
}
