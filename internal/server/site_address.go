package server

import (
	"encoding/json"
	"errors"
	"net/http"
	"os"

	"github.com/criscornea/static_studio/internal/project"
	"github.com/criscornea/static_studio/internal/publish"
	"github.com/criscornea/static_studio/internal/ssg"
)

// siteAddressResponse describes the chosen site address, what it changes and
// how the saved domain relates to static/CNAME.
type siteAddressResponse struct {
	SiteKind     string `json:"siteKind"`
	CustomDomain string `json:"customDomain"`
	// Address is the derived public URL, empty while it cannot be derived;
	// AddressError then says why.
	Address      string `json:"address"`
	AddressError string `json:"addressError"`
	// ConfigFile is the file whose baseURL is changed, CurrentBaseURL its value.
	ConfigFile     string   `json:"configFile"`
	CurrentBaseURL string   `json:"currentBaseUrl"`
	Warnings       []string `json:"warnings"`
	// StaticCNAME is the value of static/CNAME, empty when absent or invalid.
	StaticCNAME string `json:"staticCname"`
	// Conflict is set while the saved domain and static/CNAME disagree and
	// no recorded choice settles it.
	Conflict *publish.Conflict `json:"conflict"`
	// Suggestion is the static/CNAME domain offered when no domain is saved;
	// it is never applied without a request.
	Suggestion string `json:"suggestion"`
}

func derive(s publish.Settings) (addr, why string) {
	var owner, repo string
	if s.SiteKind == publish.SiteProject || s.SiteKind == publish.SiteUser {
		var err error
		if owner, repo, err = publish.ParseGitHubRemote(s.Remote); err != nil {
			return "", err.Error()
		}
	}
	addr, err := publish.Address(s.SiteKind, owner, repo, s.CustomDomain)
	if err != nil {
		return "", err.Error()
	}
	return addr, ""
}

func siteState(open *project.Open, s publish.Settings) (siteAddressResponse, *ssg.BaseURLConfig) {
	static := publish.ReadStaticCNAME(open.Root())
	resolved, conflict := s.ResolveCNAME(static)
	resp := siteAddressResponse{
		SiteKind: resolved.SiteKind, CustomDomain: resolved.CustomDomain,
		StaticCNAME: static, Conflict: conflict, Warnings: []string{},
	}
	if resolved.CustomDomain == "" {
		resp.Suggestion = static
	}
	resp.Address, resp.AddressError = derive(resolved)
	cfg, err := ssg.ReadBaseURL(open.Root(), open.Info.ConfigFile, os.Getenv)
	if err != nil {
		resp.AddressError = err.Error()
		return resp, nil
	}
	resp.ConfigFile, resp.CurrentBaseURL = cfg.File, cfg.Current
	resp.Warnings = append(resp.Warnings, cfg.Warnings...)
	return resp, cfg
}

func (s *Server) handleSiteAddress(w http.ResponseWriter, r *http.Request) {
	open, ok := s.requireProject(w, r)
	if !ok {
		return
	}
	resp, _ := siteState(open, s.pubset.Get(open.Info.Root))
	s.writeJSON(w, http.StatusOK, resp)
}

func (s *Server) handleSiteAddressSave(w http.ResponseWriter, r *http.Request) {
	open, ok := s.requireProject(w, r)
	if !ok {
		return
	}
	var in publish.SiteInput
	dec := json.NewDecoder(http.MaxBytesReader(w, r.Body, 8<<10))
	dec.DisallowUnknownFields()
	if err := dec.Decode(&in); err != nil {
		s.writeError(w, http.StatusBadRequest, "bad_request", "The request could not be read.")
		return
	}

	cur := s.pubset.Get(open.Info.Root)
	next, conflict, err := publish.UpdateSite(cur, in, publish.ReadStaticCNAME(open.Root()))
	switch {
	case errors.Is(err, publish.ErrInvalid):
		s.writeError(w, http.StatusUnprocessableEntity, "invalid_publish_settings", invalidMessage(err))
		return
	case err != nil:
		s.writeError(w, http.StatusInternalServerError, "internal", "Something went wrong. Please try again.")
		return
	case conflict != nil:
		s.writeJSON(w, http.StatusConflict, map[string]any{
			"code":     "cname_conflict",
			"message":  "The custom domain you entered differs from static/CNAME. Choose which one to keep.",
			"conflict": conflict,
		})
		return
	}

	// Fail on invalid settings before the config file is touched.
	if _, err := next.Validate(); err != nil {
		s.writeError(w, http.StatusUnprocessableEntity, "invalid_publish_settings", invalidMessage(err))
		return
	}

	resp, cfg := siteState(open, next)
	if next.SiteKind != "" {
		if resp.Address == "" {
			s.writeError(w, http.StatusUnprocessableEntity, "invalid_publish_settings",
				"Check the site address: "+resp.AddressError)
			return
		}
		if cfg == nil {
			s.writeError(w, http.StatusUnprocessableEntity, "invalid_site_config",
				"The site configuration cannot be changed: "+resp.AddressError)
			return
		}
		if cfg.Current != resp.Address {
			if err := cfg.Write(open.Root(), resp.Address); err != nil {
				if errors.Is(err, ssg.ErrConfigChanged) {
					s.writeError(w, http.StatusConflict, "config_changed", err.Error())
					return
				}
				s.log.Warn("writing baseURL failed", "err", err)
				s.writeError(w, http.StatusInternalServerError, "internal",
					"The site configuration could not be changed.")
				return
			}
			resp.CurrentBaseURL = resp.Address
		}
	}
	saved, err := s.pubset.Set(open.Info.Root, next)
	if err != nil {
		if errors.Is(err, publish.ErrInvalid) {
			s.writeError(w, http.StatusUnprocessableEntity, "invalid_publish_settings", invalidMessage(err))
			return
		}
		s.log.Warn("saving publish settings failed", "err", err)
		s.writeError(w, http.StatusInternalServerError, "internal", "The publish settings could not be saved.")
		return
	}
	resp.SiteKind, resp.CustomDomain = saved.SiteKind, saved.CustomDomain
	s.writeJSON(w, http.StatusOK, resp)
}
