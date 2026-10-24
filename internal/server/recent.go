package server

import (
	"encoding/json"
	"net/http"
	"os"

	"github.com/criscornea/static_studio/internal/recent"
)

// recentEntry is a recent project plus whether its folder still exists.
type recentEntry struct {
	recent.Entry
	Available bool `json:"available"`
}

type recentResponse struct {
	Recent []recentEntry `json:"recent"`
}

func (s *Server) handleRecent(w http.ResponseWriter, _ *http.Request) {
	entries := s.recent.List()
	out := make([]recentEntry, 0, len(entries))
	for _, e := range entries {
		info, err := os.Stat(e.Path)
		out = append(out, recentEntry{Entry: e, Available: err == nil && info.IsDir()})
	}
	s.writeJSON(w, http.StatusOK, recentResponse{Recent: out})
}

func (s *Server) handleRecentRemove(w http.ResponseWriter, r *http.Request) {
	var req openProjectRequest
	dec := json.NewDecoder(http.MaxBytesReader(w, r.Body, 8<<10))
	dec.DisallowUnknownFields()
	if err := dec.Decode(&req); err != nil || req.Path == "" {
		s.writeError(w, http.StatusBadRequest, "bad_request", "The request could not be read.")
		return
	}
	if err := s.recent.Remove(req.Path); err != nil {
		s.log.Warn("removing recent project failed", "err", err)
		s.writeError(w, http.StatusInternalServerError, "recent_failed", "The recent project could not be removed.")
		return
	}
	w.WriteHeader(http.StatusNoContent)
}
