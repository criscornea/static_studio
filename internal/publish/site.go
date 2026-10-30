package publish

import (
	"errors"
	"fmt"
	"io"
	"os"
	"regexp"
	"strings"
)

// Site address kinds.
const (
	SiteProject = "project"
	SiteUser    = "user"
	SiteCustom  = "custom"
)

// Answers to a conflict between the saved domain and static/CNAME.
const (
	CNAMESetting = "setting"
	CNAMEStatic  = "static"
)

var hostLabelRe = regexp.MustCompile(`^[a-z0-9](?:[a-z0-9-]{0,61}[a-z0-9])?$`)

// ValidDomain reports whether d is a bare lowercase hostname: no scheme,
// port, path, wildcard or trailing dot, with at least two labels.
func ValidDomain(d string) bool {
	if len(d) > 253 {
		return false
	}
	labels := strings.Split(d, ".")
	if len(labels) < 2 {
		return false
	}
	for _, l := range labels {
		if !hostLabelRe.MatchString(l) {
			return false
		}
	}
	return true
}

func (s *Settings) validateSite() error {
	s.CustomDomain = strings.TrimSpace(s.CustomDomain)
	switch s.SiteKind {
	case "", SiteProject, SiteUser:
	case SiteCustom:
		if s.CustomDomain == "" {
			return fmt.Errorf("%w: the custom domain is empty", ErrInvalid)
		}
	default:
		return fmt.Errorf("%w: the site kind is not supported", ErrInvalid)
	}
	if s.CustomDomain != "" && !ValidDomain(s.CustomDomain) {
		return fmt.Errorf("%w: the custom domain must be a bare lowercase hostname such as www.example.com", ErrInvalid)
	}
	switch s.CNAMEChoice {
	case "", CNAMESetting, CNAMEStatic:
	default:
		return fmt.Errorf("%w: the CNAME choice is not valid", ErrInvalid)
	}
	return nil
}

// ParseGitHubRemote returns the owner and repository of an https or SSH
// github.com remote.
func ParseGitHubRemote(remote string) (owner, repo string, err error) {
	r := strings.TrimSpace(remote)
	var path string
	switch {
	case strings.HasPrefix(r, "https://github.com/"):
		path = strings.TrimPrefix(r, "https://github.com/")
	case strings.HasPrefix(r, "ssh://git@github.com/"):
		path = strings.TrimPrefix(r, "ssh://git@github.com/")
	case strings.HasPrefix(r, "git@github.com:"):
		path = strings.TrimPrefix(r, "git@github.com:")
	default:
		return "", "", errors.New("the remote is not a github.com address")
	}
	path = strings.TrimSuffix(strings.TrimSuffix(path, "/"), ".git")
	owner, repo, ok := strings.Cut(path, "/")
	if !ok || !ownerRe.MatchString(owner) || !repoRe.MatchString(repo) {
		return "", "", errors.New("the remote is not a github.com owner/repository address")
	}
	return owner, repo, nil
}

// Address derives the site's public URL (with trailing slash). owner and repo
// are only used for project and user sites.
func Address(kind, owner, repo, domain string) (string, error) {
	switch kind {
	case SiteProject:
		if owner == "" || repo == "" {
			return "", errors.New("the GitHub owner and repository are needed for a project site")
		}
		return "https://" + strings.ToLower(owner) + ".github.io/" + repo + "/", nil
	case SiteUser:
		if owner == "" || !strings.EqualFold(repo, owner+".github.io") {
			return "", fmt.Errorf("a user site needs the repository to be named %s.github.io", owner)
		}
		return "https://" + strings.ToLower(owner) + ".github.io/", nil
	case SiteCustom:
		if !ValidDomain(domain) {
			return "", errors.New("the custom domain is not valid")
		}
		return "https://" + domain + "/", nil
	}
	return "", errors.New("choose a site address first")
}

// Conflict says the saved domain and static/CNAME disagree.
type Conflict struct {
	Setting string `json:"setting"`
	Static  string `json:"static"`
}

// ReadStaticCNAME returns the trimmed, validated content of static/CNAME read
// through root, or "" when the file is absent, unreadable or not a hostname.
func ReadStaticCNAME(root *os.Root) string {
	f, err := root.Open("static/CNAME")
	if err != nil {
		return ""
	}
	defer func() { _ = f.Close() }()
	data, err := io.ReadAll(io.LimitReader(f, 1024))
	if err != nil {
		return ""
	}
	v := strings.TrimSpace(string(data))
	if !ValidDomain(v) {
		return ""
	}
	return v
}

// ResolveCNAME compares CustomDomain with static/CNAME. A stored choice made
// against the same static value settles a difference (adopting the static
// value when it says so); otherwise the difference is returned as a Conflict.
func (s Settings) ResolveCNAME(static string) (Settings, *Conflict) {
	if s.SiteKind != SiteCustom || static == "" || static == s.CustomDomain {
		return s, nil
	}
	if s.CNAMEChoiceFor == static {
		switch s.CNAMEChoice {
		case CNAMESetting:
			return s, nil
		case CNAMEStatic:
			s.CustomDomain = static
			return s, nil
		}
	}
	return s, &Conflict{Setting: s.CustomDomain, Static: static}
}

// ErrCNAMEConflict is wrapped by CheckSite when publishing must wait for an answer.
var ErrCNAMEConflict = errors.New("the custom domain and static/CNAME differ")

// CheckSite resolves the CNAME against root and returns the settings to
// publish with, or an ErrInvalid error describing the conflict.
func CheckSite(root *os.Root, s Settings) (Settings, error) {
	s, c := s.ResolveCNAME(ReadStaticCNAME(root))
	if c != nil {
		return s, fmt.Errorf("%w: %w: the saved custom domain is %s but static/CNAME says %s; choose which to keep in the site address settings",
			ErrInvalid, ErrCNAMEConflict, c.Setting, c.Static)
	}
	return s, nil
}

// SiteInput is what a site address change carries. CNAMEChoice is the user's
// explicit answer to a conflict, empty otherwise.
type SiteInput struct {
	SiteKind     string `json:"siteKind"`
	CustomDomain string `json:"customDomain"`
	CNAMEChoice  string `json:"cnameChoice"`
}

// UpdateSite applies in to cur against the current static/CNAME value. When
// the saved domain and static/CNAME differ and no choice settles it, it
// returns the Conflict and the unchanged cur: nothing should be saved. An
// explicit choice is recorded together with the static value it answers, and
// "static" adopts that value as the custom domain.
func UpdateSite(cur Settings, in SiteInput, static string) (Settings, *Conflict, error) {
	next := cur
	next.SiteKind, next.CustomDomain = in.SiteKind, in.CustomDomain
	if err := next.validateSite(); err != nil {
		return cur, nil, err
	}
	switch in.CNAMEChoice {
	case "":
		// A recorded choice answers a conflict about the domain it was made for.
		if in.CustomDomain != cur.CustomDomain {
			next.CNAMEChoice, next.CNAMEChoiceFor = "", ""
		}
	case CNAMESetting, CNAMEStatic:
		if static == "" {
			return cur, nil, fmt.Errorf("%w: there is no static/CNAME to choose", ErrInvalid)
		}
		next.CNAMEChoice, next.CNAMEChoiceFor = in.CNAMEChoice, static
	default:
		return cur, nil, fmt.Errorf("%w: the CNAME choice is not valid", ErrInvalid)
	}
	resolved, c := next.ResolveCNAME(static)
	if c != nil {
		return cur, c, nil
	}
	return resolved, nil, nil
}
