package server

import (
	"encoding/json"
	"errors"
	"net/http"

	"github.com/criscornea/static_studio/internal/publish"
)

func (s *Server) handlePublishSettings(w http.ResponseWriter, r *http.Request) {
	open, ok := s.requireProject(w, r)
	if !ok {
		return
	}
	s.writeJSON(w, http.StatusOK, s.pubset.Get(open.Info.Root))
}

func (s *Server) handlePublishSettingsSave(w http.ResponseWriter, r *http.Request) {
	open, ok := s.requireProject(w, r)
	if !ok {
		return
	}
	var in publish.Settings
	dec := json.NewDecoder(http.MaxBytesReader(w, r.Body, 8<<10))
	dec.DisallowUnknownFields()
	if err := dec.Decode(&in); err != nil {
		s.writeError(w, http.StatusBadRequest, "bad_request", "The request could not be read.")
		return
	}
	// The site address has its own endpoint; saving the target settings
	// keeps what was chosen there.
	saved, err := s.pubset.Update(open.Info.Root, func(cur publish.Settings) (publish.Settings, error) {
		in.SiteKind, in.CustomDomain = cur.SiteKind, cur.CustomDomain
		in.CNAMEChoice, in.CNAMEChoiceFor = cur.CNAMEChoice, cur.CNAMEChoiceFor
		return in.Validate()
	})
	switch {
	case err == nil:
		s.writeJSON(w, http.StatusOK, saved)
	case errors.Is(err, publish.ErrInvalid):
		s.writeError(w, http.StatusUnprocessableEntity, "invalid_publish_settings", invalidMessage(err))
	default:
		s.log.Warn("saving publish settings failed", "err", err)
		s.writeError(w, http.StatusInternalServerError, "internal",
			"The publish settings could not be saved.")
	}
}

// invalidMessage turns a validation error into a sentence for the UI.
func invalidMessage(err error) string {
	msg := err.Error()
	prefix := publish.ErrInvalid.Error() + ": "
	if len(msg) > len(prefix) {
		msg = msg[len(prefix):]
	}
	return "Check the publish settings: " + msg + "."
}

func (s *Server) handlePublish(w http.ResponseWriter, r *http.Request) {
	open, ok := s.requireProject(w, r)
	if !ok {
		return
	}

	switch err := s.publish.Start(s.build, open.Root(), s.pubset.Get(open.Info.Root)); {
	case err == nil:
		s.log.Info("publish started", "root", open.Info.Root)
		s.writeJSON(w, http.StatusAccepted, map[string]bool{"started": true})

	case errors.Is(err, publish.ErrBusy):
		s.writeError(w, http.StatusConflict, "publish_running",
			"A publish is already running.")

	case errors.Is(err, publish.ErrInvalid):
		s.writeError(w, http.StatusUnprocessableEntity, "invalid_publish_settings", invalidMessage(err))

	default:
		s.log.Warn("starting publish failed", "err", err)
		s.writeError(w, http.StatusInternalServerError, "internal",
			"Something went wrong. Please try again.")
	}
}

func (s *Server) handlePublishCancel(w http.ResponseWriter, r *http.Request) {
	if _, ok := s.requireProject(w, r); !ok {
		return
	}
	s.publish.Stop()
	w.WriteHeader(http.StatusNoContent)
}

func (s *Server) handlePublishStream(w http.ResponseWriter, r *http.Request) {
	if _, ok := s.requireProject(w, r); !ok {
		return
	}
	s.streamEvents(w, r, &s.publish.Events, s.publish.Running)
}

func (s *Server) handlePublishPreflight(w http.ResponseWriter, r *http.Request) {
	open, ok := s.requireProject(w, r)
	if !ok {
		return
	}
	s.writeJSON(w, http.StatusOK, publish.Preflight{}.Run(r.Context(), open.Root()))
}
