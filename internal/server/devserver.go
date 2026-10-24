package server

import (
	"errors"
	"net/http"

	"github.com/criscornea/static_studio/internal/devserver"
)

// devServerResponse is the dev server state. PageURL is set when the request
// names a content file and the server is running.
type devServerResponse struct {
	devserver.Status

	PageURL string `json:"pageUrl,omitempty"`
}

func (s *Server) devStatus(contentDir, page string) devServerResponse {
	st := s.dev.Status()
	resp := devServerResponse{Status: st}
	if st.Running && page != "" {
		if p, ok := devserver.PageURLPath(contentDir, page, false); ok {
			resp.PageURL = st.URL + p
		}
	}

	return resp
}

func (s *Server) handleDevServerStatus(w http.ResponseWriter, r *http.Request) {
	open, ok := s.requireProject(w, r)
	if !ok {
		return
	}

	s.writeJSON(w, http.StatusOK, s.devStatus(open.Info.ContentDir, r.URL.Query().Get("path")))
}

func (s *Server) handleDevServerStart(w http.ResponseWriter, r *http.Request) {
	open, ok := s.requireProject(w, r)
	if !ok {
		return
	}

	err := s.dev.Start(open.Info.Root)
	if err == nil {
		// A project switch may have run between the check above and Start.
		// The switch stops the server before this one began, so stop it here.
		if _, rerr := s.project.Require(open.ID); rerr != nil {
			s.dev.Stop()
			s.requireProject(w, r)

			return
		}
	}

	switch {
	case err == nil:
		s.log.Info("dev server started", "root", open.Info.Root)
		s.writeJSON(w, http.StatusAccepted, s.devStatus(open.Info.ContentDir, ""))

	case errors.Is(err, devserver.ErrRunning):
		s.writeError(w, http.StatusConflict, "devserver_running",
			"The preview server is already running.")

	default:
		s.log.Warn("starting dev server failed", "err", err)
		s.writeError(w, http.StatusInternalServerError, "devserver_failed",
			"The preview server could not be started. Check that Hugo is installed.")
	}
}

func (s *Server) handleDevServerStop(w http.ResponseWriter, r *http.Request) {
	if _, ok := s.requireProject(w, r); !ok {
		return
	}

	s.dev.Stop()
	w.WriteHeader(http.StatusNoContent)
}
