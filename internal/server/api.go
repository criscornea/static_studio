package server

import (
	"encoding/json"
	"errors"
	"net/http"

	"github.com/criscornea/static_studio/internal/content"
	"github.com/criscornea/static_studio/internal/frontmatter"
	"github.com/criscornea/static_studio/internal/project"
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
			"This looks like an Astro project. static_studio currently works with Hugo  only.")

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

// contentResponse is the content treee of the open project.
type contentResponse struct {
	Root *content.Node `json:"root"`
}

// projectID reads the projectc id from the request. The header is preferred;
// the query parameter exists so that links and manual testing work.
func projectID(r *http.Request) string {
	if id := r.Header.Get("X-Project-ID"); id != "" {
		return id
	}

	return r.URL.Query().Get("projectId")
}

// requireProject resolves the open project for this request, writing an
// error response and returning false if it cannot.
func (s *Server) requireProject(w http.ResponseWriter, r *http.Request) (*project.Open, bool) {
	open, err := s.project.Require(projectID(r))
	switch {
	case err == nil:
		return open, true

	case errors.Is(err, project.ErrNoProject):
		s.writeError(w, http.StatusConflict, "no_project",
			"No project is open. Open a project folder first.")

	case errors.Is(err, project.ErrStaleID):
		s.writeError(w, http.StatusConflict, "stale_project",
			"A different project is open now. Reload to continue.")

	default:
		s.log.Warn("resolving the open project failed", "err", err)
		s.writeError(w, http.StatusInternalServerError, "internal",
			"Something went wrong. Please try again.")
	}

	return nil, false
}

func (s *Server) handleContent(w http.ResponseWriter, r *http.Request) {
	open, ok := s.requireProject(w, r)
	if !ok {
		return
	}

	tree, err := content.Tree(open.Root().FS(), open.Info.ContentDir)
	if err != nil {
		s.log.Warn("reading the content tree failed", "root", open.Info.Root, "dir", open.Info.ContentDir, "err", err)
		s.writeError(w, http.StatusInternalServerError, "cannot_read_content",
			"The content folder could not be read. Check that you have permission to read it.")
	}

	s.writeJSON(w, http.StatusOK, contentResponse{Root: tree})
}

func (s *Server) handlePage(w http.ResponseWriter, r *http.Request) {
	open, ok := s.requireProject(w, r)
	if !ok {
		return
	}

	name := r.URL.Query().Get("path")
	if name == "" {
		s.writeError(w, http.StatusBadRequest, "missing_path", "No page was given.")
		return
	}

	page, err := content.ReadPage(open.Root().FS(), open.Info.ContentDir, name)
	switch {
	case err == nil:
		s.writeJSON(w, http.StatusOK, page)

	case errors.Is(err, content.ErrNotEditable):
		s.writeError(w, http.StatusBadRequest, "not_editable",
			"This file can't be edited here. Only pages inside the content folder can be opened.")

	case errors.Is(err, content.ErrNotFound):
		s.writeError(w, http.StatusNotFound, "page_not_found",
			"This page no longer exists. It may have been moved or deleted outside the editor.")

	case errors.Is(err, content.ErrTooLarge):
		s.writeError(w, http.StatusUnprocessableEntity, "page_too_large",
			"This page is too large to open in the editor.")

	case errors.Is(err, frontmatter.ErrInvalid):
		s.log.Info("invalid frontmatter", "path", name, "err", err)
		s.writeError(w, http.StatusUnprocessableEntity, "invalid_frontmatter",
			"The settings block at the top of this page could not be read. It proably contains a typo.")

	default:
		s.log.Warn("reading page failed", "path", name, "err", err)
		s.writeError(w, http.StatusInternalServerError, "cannot_read_page",
			"This page could not be read. Check that you have permission to read it.")
	}
}
