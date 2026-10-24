package publish

import (
	"fmt"
	"net"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"unicode"
)

var (
	rsyncUserRe = regexp.MustCompile(`^[A-Za-z_][A-Za-z0-9._-]{0,31}$`)
	rsyncHostRe = regexp.MustCompile(`^([A-Za-z0-9]([A-Za-z0-9-]{0,61}[A-Za-z0-9])?)(\.[A-Za-z0-9]([A-Za-z0-9-]{0,61}[A-Za-z0-9])?)*$`)
	rsyncPathRe = regexp.MustCompile(`^[A-Za-z0-9._/+=,@%-]+$`)
	ipv4Re      = regexp.MustCompile(`^[0-9]{1,3}(\.[0-9]{1,3}){3}$`)
)

// shellMeta are characters that no field may contain, even though no shell
// is ever involved: the remote side of rsync and ssh may run one.
const shellMeta = "\\\"'`$;&|<>(){}*?!#^"

// checkPlain refuses a leading "-", control characters, whitespace and shell
// metacharacters.
func checkPlain(name, v string) error {
	switch {
	case strings.HasPrefix(v, "-"):
		return fmt.Errorf("%w: %s may not start with \"-\"", ErrInvalid, name)
	case strings.IndexFunc(v, func(r rune) bool { return unicode.IsControl(r) || unicode.IsSpace(r) }) >= 0:
		return fmt.Errorf("%w: %s may not contain spaces or control characters", ErrInvalid, name)
	case strings.ContainsAny(v, shellMeta):
		return fmt.Errorf("%w: %s contains a character that is not allowed", ErrInvalid, name)
	}
	return nil
}

func validHost(h string) bool {
	if strings.HasPrefix(h, "[") && strings.HasSuffix(h, "]") {
		ip := net.ParseIP(h[1 : len(h)-1])
		return ip != nil && ip.To4() == nil
	}
	if ipv4Re.MatchString(h) {
		ip := net.ParseIP(h)
		return ip != nil
	}
	return len(h) <= 253 && rsyncHostRe.MatchString(h)
}

// splitDest splits "user@host:path" into its parts. The host may be a
// bracketed IPv6 address.
func splitDest(d string) (user, host, path string, ok bool) {
	rest := d
	if u, r, found := strings.Cut(rest, "@"); found {
		user, rest = u, r
	}
	if strings.HasPrefix(rest, "[") {
		end := strings.Index(rest, "]:")
		if end < 0 {
			return "", "", "", false
		}
		return user, rest[:end+1], rest[end+2:], true
	}
	i := strings.Index(rest, ":")
	if i < 0 {
		return "", "", "", false
	}
	return user, rest[:i], rest[i+1:], true
}

// IsRemote reports whether a validated rsync destination is user@host:path
// rather than a local path.
func IsRemote(dest string) bool { return !filepath.IsAbs(dest) }

func (s Settings) validateRsync() (Settings, error) {
	s.RsyncDest = strings.TrimSpace(s.RsyncDest)
	s.RsyncIdentity = strings.TrimSpace(s.RsyncIdentity)
	d := s.RsyncDest
	if d == "" {
		return s, fmt.Errorf("%w: the destination is empty", ErrInvalid)
	}
	if err := checkPlain("the destination", d); err != nil {
		return s, err
	}
	if filepath.IsAbs(d) {
		if !rsyncPathRe.MatchString(d) {
			return s, fmt.Errorf("%w: the destination path has a character that is not allowed", ErrInvalid)
		}
	} else {
		user, host, path, ok := splitDest(d)
		switch {
		case !ok:
			return s, fmt.Errorf("%w: the destination must be user@host:path or a full local path", ErrInvalid)
		case strings.Contains(d, "@") && !rsyncUserRe.MatchString(user):
			return s, fmt.Errorf("%w: the destination user is not valid", ErrInvalid)
		case !validHost(host):
			return s, fmt.Errorf("%w: the destination host is not valid", ErrInvalid)
		case path == "" || strings.HasPrefix(path, "-") || !rsyncPathRe.MatchString(path):
			return s, fmt.Errorf("%w: the destination path is empty or has a character that is not allowed", ErrInvalid)
		}
	}
	// 0 means "not set": ssh uses its own default port.
	if s.RsyncPort < 0 || s.RsyncPort > 65535 {
		return s, fmt.Errorf("%w: the port must be between 1 and 65535 (leave it empty for the ssh default)", ErrInvalid)
	}
	if id := s.RsyncIdentity; id != "" {
		if err := checkPlain("the identity file", id); err != nil {
			return s, err
		}
		if !filepath.IsAbs(id) || !rsyncPathRe.MatchString(id) {
			return s, fmt.Errorf("%w: the identity file must be a full path without spaces or special characters", ErrInvalid)
		}
		if fi, err := os.Stat(id); err != nil || !fi.Mode().IsRegular() {
			return s, fmt.Errorf("%w: the identity file does not exist or is not a regular file", ErrInvalid)
		}
	}
	return s, nil
}
