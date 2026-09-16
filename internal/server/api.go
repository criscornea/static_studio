package server

import (
	"errors"
	"net/http"

	"github.com/criscornea/static_studio/internal/ssg"
)

// apiError is the JSON body returned for every failed API request.
type apiError struct {
	// Code is a stable machine-readable identifier for the frontend.
	Code string `json:"code"`
	// Message is human-readable and written for non-developers.
	Message string `json:"message"`
}

func (s *Server) writeError(w http.ResponseWriter, status int, code, message string) {
	s.writeJSON(w, status, apiError{Code: code, Message: message})
}

func (s *Server) handleOpenProject(w http.ResponseWriter, r *http.Request) {
	dir := r.URL.Query().Get("path")
	if dir == "" {
		s.writeError(w, http.StatusBadRequest, "missing_path", "No folder was given.")
		return
	}

	project, err := ssg.Detect(dir)
	switch {
	case err == nil:
		s.log.Info("project opened", "root", project.Root, "kind", project.Kind)
		s.writeJSON(w, http.StatusOK, project)

	case errors.Is(err, ssg.ErrNotDetected):
		s.log.Info("no project detected", "path", dir)
		s.writeError(w, http.StatusUnprocessableEntity, "not_a_project",
			"This folder doesn't look like a website project. Pick the folder that contains your site's config file, for example hugo.toml")

	case errors.Is(err, ssg.ErrUnsupported):
		s.log.Info("unsupported generator", "path", dir, "err", err)
		s.writeError(w, http.StatusNotImplemented, "unsupported_generator",
			"This looks like an Astro project. Static Studio currently works with Hugo only.")

	default:
		s.log.Warn("opnening project failed", "path", dir, "err", err)
		s.writeError(w, http.StatusBadRequest, "cannot_open",
			"That folder could not be openend. Check that the path is correct and that you have permission to read it.")
	}
}
