package server

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"time"

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
		s.build.Stop()
		s.publish.Stop()
		s.publish.Events.Reset()
		s.dev.Stop()
		s.log.Info("project opened", "root", opened.Info.Root, "kind", opened.Info.Kind, "id", opened.ID)
		if err := s.recent.Add(opened.Info.Root, string(opened.Info.Kind)); err != nil {
			s.log.Warn("recording recent project failed", "err", err)
		}
		s.writeJSON(w, http.StatusOK, projectResponse{Project: opened.Info, ID: opened.ID})

	case errors.Is(err, ssg.ErrNotDetected):
		s.log.Info("no project detected", "path", req.Path)
		s.writeError(w, http.StatusUnprocessableEntity, "not_a_project",
			"This folder doesn't look like a website project. Pick the folder that contains your site's configuration file, for example hugo.toml.")

	case errors.Is(err, ssg.ErrUnsupported):
		s.log.Info("unsupported generator", "path", req.Path, "err", err)
		s.writeError(w, http.StatusNotImplemented, "unsupported_generator",
			"This looks like an Astro project. static_studio currently works with Hugo  only.")

	case errors.Is(err, ssg.ErrInvalidConfig):
		s.log.Info("invalid site configuration", "path", req.Path, "err", err)
		msg := "The site's configuration file could not be used."
		var ce *ssg.ConfigError
		if errors.As(err, &ce) {
			msg = fmt.Sprintf("The site's configuration file %s could not be used: %s. Fix the file and open the folder again.", ce.File, ce.Reason)
		}
		s.writeError(w, http.StatusUnprocessableEntity, "invalid_config", msg)

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

// contentRoot is the tree of one content dir.
type contentRoot struct {
	Dir  string        `json:"dir"`
	Lang string        `json:"lang,omitempty"`
	Root *content.Node `json:"root"`
}

// contentResponse is the content of the open project: one root per content
// dir, primary first. Root repeats the primary tree for older clients.
type contentResponse struct {
	Root  *content.Node `json:"root"`
	Roots []contentRoot `json:"roots"`
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

	resp := contentResponse{Roots: make([]contentRoot, 0, len(open.Info.ContentDirs))}
	for _, d := range open.Info.ContentDirs {
		tree, err := content.Tree(open.Root().FS(), d.Path)
		if err != nil {
			s.log.Warn("reading the content tree failed", "root", open.Info.Root, "dir", d.Path, "err", err)
			s.writeError(w, http.StatusInternalServerError, "cannot_read_content",
				"The content folder could not be read. Check that you have permission to read it.")
			return
		}
		resp.Roots = append(resp.Roots, contentRoot{Dir: d.Path, Lang: d.Lang, Root: tree})
	}
	if len(resp.Roots) > 0 {
		resp.Root = resp.Roots[0].Root
	}

	s.writeJSON(w, http.StatusOK, resp)
}

// archetypesResponse lists the templates a new page can start from.
type archetypesResponse struct {
	Archetypes []content.Archetype `json:"archetypes"`
}

func (s *Server) handleArchetypes(w http.ResponseWriter, r *http.Request) {
	open, ok := s.requireProject(w, r)
	if !ok {
		return
	}

	s.writeJSON(w, http.StatusOK, archetypesResponse{Archetypes: content.Archetypes(open.Root().FS())})
}

type createPageRequest struct {
	Archetype string `json:"archetype"`
	Folder    string `json:"folder"`
	Name      string `json:"name"`
}

func (s *Server) handleCreatePage(w http.ResponseWriter, r *http.Request) {
	open, ok := s.requireProject(w, r)
	if !ok {
		return
	}

	var req createPageRequest
	dec := json.NewDecoder(http.MaxBytesReader(w, r.Body, 8<<10))
	dec.DisallowUnknownFields()
	if err := dec.Decode(&req); err != nil {
		s.writeError(w, http.StatusBadRequest, "bad_request", "The request could not be read.")
		return
	}

	page, err := content.CreatePage(open.Root(), open.Info.ContentDir, content.NewPage{
		Archetype: req.Archetype,
		Folder:    req.Folder,
		Name:      req.Name,
	}, time.Now())
	switch {
	case err == nil:
		s.log.Info("page created", "path", page.Path)
		s.writeJSON(w, http.StatusCreated, page)

	case errors.Is(err, content.ErrInvalidName):
		s.writeError(w, http.StatusBadRequest, "invalid_name",
			"Give the page a name with at least one letter or number.")

	case errors.Is(err, content.ErrInvalidFolder):
		s.writeError(w, http.StatusBadRequest, "invalid_folder",
			"That folder name can't be used. It can't contain \"..\" or start with a dot.")

	case errors.Is(err, content.ErrUnknownArchetype):
		s.writeError(w, http.StatusBadRequest, "unknown_archetype",
			"That template no longer exists. Pick another one.")

	case errors.Is(err, content.ErrUnsupportedArchetype):
		s.log.Info("unsupported archetype", "archetype", req.Archetype, "err", err)
		s.writeError(w, http.StatusUnprocessableEntity, "unsupported_archetype",
			"This template uses features Static Studio can't fill in yet. Pick another template.")

	case errors.Is(err, content.ErrInvalidArchetype):
		s.log.Info("invalid archetype", "archetype", req.Archetype, "err", err)
		s.writeError(w, http.StatusUnprocessableEntity, "invalid_archetype",
			"This template would create a page with a broken settings block. Fix the template or pick another one.")

	case errors.Is(err, content.ErrPageExists):
		s.writeError(w, http.StatusConflict, "page_exists",
			"A page with this name already exists in that folder.")

	default:
		s.log.Warn("creating a page failed", "root", open.Info.Root, "err", err)
		s.writeError(w, http.StatusInternalServerError, "cannot_create_page",
			"The page could not be created. Check that you have permission to write to the content folder.")
	}
}

// maxQueryLength caps a search query. Longer ones cannot be typed by hand.
const maxQueryLength = 200

func (s *Server) handleSearch(w http.ResponseWriter, r *http.Request) {
	open, ok := s.requireProject(w, r)
	if !ok {
		return
	}

	query := r.URL.Query().Get("q")
	if len(query) > maxQueryLength {
		s.writeError(w, http.StatusBadRequest, "bad_request", "The search text is too long.")
		return
	}

	dirs := make([]string, 0, len(open.Info.ContentDirs))
	for _, d := range open.Info.ContentDirs {
		dirs = append(dirs, d.Path)
	}
	results, err := content.Search(open.Root().FS(), dirs, query)
	switch {
	case err == nil:
		s.writeJSON(w, http.StatusOK, results)

	case errors.Is(err, content.ErrEmptyQuery):
		s.writeError(w, http.StatusBadRequest, "missing_query", "No search text was given.")

	default:
		s.log.Warn("searching failed", "root", open.Info.Root, "err", err)
		s.writeError(w, http.StatusInternalServerError, "cannot_search",
			"The content could not be searched. Check that you have permission to read it.")
	}
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

type renamePageRequest struct {
	From string `json:"from"`
	To   string `json:"to"`
}

type renamePageResponse struct {
	Path string `json:"path"`
}

func (s *Server) handleRenamePage(w http.ResponseWriter, r *http.Request) {
	open, ok := s.requireProject(w, r)
	if !ok {
		return
	}

	var req renamePageRequest
	dec := json.NewDecoder(http.MaxBytesReader(w, r.Body, 8<<10))
	dec.DisallowUnknownFields()
	if err := dec.Decode(&req); err != nil {
		s.writeError(w, http.StatusBadRequest, "bad_request", "The request could not be read.")
		return
	}
	if req.From == "" || req.To == "" {
		s.writeError(w, http.StatusBadRequest, "missing_path", "No page was given.")
		return
	}

	to, err := content.RenamePage(open.Root(), open.Info.ContentDir, req.From, req.To)
	switch {
	case err == nil:
		s.log.Info("page renamed", "from", req.From, "to", to)
		s.writeJSON(w, http.StatusOK, renamePageResponse{Path: to})

	case errors.Is(err, content.ErrNotEditable):
		s.writeError(w, http.StatusBadRequest, "not_editable",
			"This file can't be renamed here. Only pages inside the content folder can be moved, and they must keep a page file extension.")

	case errors.Is(err, content.ErrNotFound):
		s.writeError(w, http.StatusNotFound, "page_not_found",
			"This page no longer exists. It may have been moved or deleted outside the editor.")

	case errors.Is(err, content.ErrPageExists):
		s.writeError(w, http.StatusConflict, "page_exists",
			"A page with this name already exists in that folder.")

	default:
		s.log.Warn("renaming a page failed", "from", req.From, "to", req.To, "err", err)
		s.writeError(w, http.StatusInternalServerError, "cannot_rename_page",
			"The page could not be renamed. Check that you have permission to write to the content folder.")
	}
}

func (s *Server) handleDeletePage(w http.ResponseWriter, r *http.Request) {
	open, ok := s.requireProject(w, r)
	if !ok {
		return
	}

	name := r.URL.Query().Get("path")
	if name == "" {
		s.writeError(w, http.StatusBadRequest, "missing_path", "No page was given.")
		return
	}

	err := content.DeletePage(open.Root(), open.Info.ContentDir, name)
	switch {
	case err == nil:
		s.log.Info("page deleted", "path", name)
		w.WriteHeader(http.StatusNoContent)

	case errors.Is(err, content.ErrNotEditable):
		s.writeError(w, http.StatusBadRequest, "not_editable",
			"This file can't be deleted here. Only pages inside the content folder can be deleted.")

	case errors.Is(err, content.ErrNotFound):
		s.writeError(w, http.StatusNotFound, "page_not_found",
			"This page no longer exists. It may have been moved or deleted outside the editor.")

	default:
		s.log.Warn("deleting a page failed", "path", name, "err", err)
		s.writeError(w, http.StatusInternalServerError, "cannot_delete_page",
			"The page could not be deleted. Check that you have permission to write to the content folder.")
	}
}

type savePageRequest struct {
	Path string `json:"path"`
	Body string `json:"body"`
	Hash string `json:"hash"`
	// Fields, when present, replaces the known frontmatter fields.
	Fields *frontmatter.Fields `json:"fields,omitempty"`
}

type savePageResponse struct {
	Hash string `json:"hash"`
}

// maxSaveRequest leaves room for a maxPageSize body plus JSON escaping.
const maxSaveRequest = 8 << 20

func (s *Server) handleSavePage(w http.ResponseWriter, r *http.Request) {
	open, ok := s.requireProject(w, r)
	if !ok {
		return
	}

	var req savePageRequest
	dec := json.NewDecoder(http.MaxBytesReader(w, r.Body, maxSaveRequest))
	dec.DisallowUnknownFields()
	if err := dec.Decode(&req); err != nil {
		s.writeError(w, http.StatusBadRequest, "bad_request", "The request could not be read.")
		return
	}
	if req.Path == "" {
		s.writeError(w, http.StatusBadRequest, "missing_path", "No page was given.")
		return
	}

	hash, err := content.SavePage(open.Root(), open.Info.ContentDir, req.Path, req.Body, req.Hash, req.Fields)
	switch {
	case err == nil:
		s.writeJSON(w, http.StatusOK, savePageResponse{Hash: hash})

	case errors.Is(err, content.ErrNotEditable):
		s.writeError(w, http.StatusBadRequest, "not_editable",
			"This file can't be saved here. Only pages inside the content folder can be edited.")

	case errors.Is(err, content.ErrNotFound):
		s.writeError(w, http.StatusNotFound, "page_not_found",
			"This page no longer exists. It may have been moved or deleted outside the editor.")

	case errors.Is(err, content.ErrConflict):
		s.writeError(w, http.StatusConflict, "page_conflict",
			"This page was changed outside the editor since you opened it. Reload it to see the latest version.")

	case errors.Is(err, content.ErrTooLarge):
		s.writeError(w, http.StatusUnprocessableEntity, "page_too_large",
			"This page is too large to save in the editor.")

	case errors.Is(err, frontmatter.ErrInvalid), errors.Is(err, frontmatter.ErrUnterminated):
		s.writeError(w, http.StatusUnprocessableEntity, "invalid_frontmatter",
			"The settings block at the top of this page could not be read, so it was not saved.")

	default:
		s.log.Warn("saving page failed", "path", req.Path, "err", err)
		s.writeError(w, http.StatusInternalServerError, "cannot_save_page",
			"The page could not be saved. Check that you have permission to write to the content folder.")
	}
}

type assetResponse struct {
	Path string `json:"path"`
}

func (s *Server) handleAsset(w http.ResponseWriter, r *http.Request) {
	open, ok := s.requireProject(w, r)
	if !ok {
		return
	}

	r.Body = http.MaxBytesReader(w, r.Body, content.MaxAssetSize+(1<<20))
	file, header, err := r.FormFile("file")
	if err != nil {
		var tooBig *http.MaxBytesError
		if errors.As(err, &tooBig) {
			s.writeError(w, http.StatusRequestEntityTooLarge, "asset_too_large",
				"This image is too large. The limit is 10 MB.")
			return
		}
		s.writeError(w, http.StatusBadRequest, "bad_request", "The request could not be read.")
		return
	}
	defer func() { _ = file.Close() }()

	data, err := io.ReadAll(io.LimitReader(file, content.MaxAssetSize+1))
	if err != nil {
		s.writeError(w, http.StatusBadRequest, "bad_request", "The request could not be read.")
		return
	}

	urlPath, err := content.SaveAsset(open.Root(), open.Info.AssetDir, header.Filename, data)
	switch {
	case err == nil:
		s.log.Info("asset saved", "path", urlPath)
		s.writeJSON(w, http.StatusCreated, assetResponse{Path: urlPath})

	case errors.Is(err, content.ErrAssetTooLarge):
		s.writeError(w, http.StatusRequestEntityTooLarge, "asset_too_large",
			"This image is too large. The limit is 10 MB.")

	case errors.Is(err, content.ErrAssetType):
		s.writeError(w, http.StatusUnsupportedMediaType, "asset_type",
			"Only PNG, JPEG, GIF and WebP images can be added.")

	case errors.Is(err, content.ErrAssetName):
		s.writeError(w, http.StatusBadRequest, "asset_name",
			"This file name can't be used. Rename the image and try again.")

	case errors.Is(err, content.ErrAssetExists):
		s.writeError(w, http.StatusConflict, "asset_exists",
			"An image with this name already exists. Rename the image and try again.")

	case errors.Is(err, content.ErrNotEditable):
		s.writeError(w, http.StatusBadRequest, "not_editable",
			"The image folder can't be written to.")

	default:
		s.log.Warn("saving asset failed", "err", err)
		s.writeError(w, http.StatusInternalServerError, "cannot_save_asset",
			"The image could not be saved. Check that you have permission to write to the asset folder.")
	}
}
