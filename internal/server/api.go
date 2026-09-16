package server

import (
	"encoding/json"
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

// projectResponse is what the client receives for an open project. The id
// must be sent back with every request that touches project files.
type projectResponse struct {
	ssg.Project
	ID string `json:"id"`
}

type openProjectRequest struct {
	Path string `json:"path"`
}

func (s *Server) handleOpenProject(w http.ResponseWriter, r *http.Request) {
	var req openProjectRequest
	dec := json.NewDecoder(http.MaxBytesReader(w, r.Body, 8<<10))
	dec.DisallowUnknownFields()

	if err := dec.Decode(&req); err != nil {
		s.writeError(w, http.StatusBadRequest, "bad_request",
			"The request could not be read.")
		return
	}
	if req.Path == "" {
		s.writeError(w, http.StatusBadRequest, "missing_path",
			"No folder was given.")
		return
	}

	opened, err := s.project.Open(req.Path)
	switch {
	case err == nil:
		s.log.Info("project opened", "root", opened.Info.Root, "kind", opened.Info.Kind, "id", opened.ID)
		s.writeJSON(w, http.StatusOK, projectResponse{Project: opened.Info, ID: opened.ID})

	case errors.Is(err, ssg.ErrNotDetected):
		s.log.Info("no project detected", "path", req.Path)
		s.writeError(w, http.StatusUnprocessableEntity, "not_a_project",
			"This folder doesn't look like a website project. Pick the folder that contains your site's configuration file, for example hugo.toml.")

	case errors.Is(err, ssg.ErrUnsupported):
		s.log.Info("unsupported generator", "path", req.Path, "err", err)
		s.writeError(w, http.StatusNotImplemented, "unsupported_generator",
			"This looks like an Astro project. static_studio currently works with Hugo projects only.")

	default:
		s.log.Warn("opening project failed", "path", req.Path, "err", err)
		s.writeError(w, http.StatusBadRequest, "cannot_open",
			"That folder could not be opened. Check that the path is correct and that you have permission to read it.")
	}
}

func (s *Server) handleCurrentProject(w http.ResponseWriter, _ *http.Request) {
	current, err := s.project.Current()
	if err != nil {
		s.writeError(w, http.StatusConflict, "no_project", "No project is open.")
		return
	}
	s.writeJSON(w, http.StatusOK, projectResponse{Project: current.Info, ID: current.ID})
}
