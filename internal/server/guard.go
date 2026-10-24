package server

import (
	"net"
	"net/http"
	"net/url"
	"strings"
)

// loopbackHost reports whether hostport names a loopback host (any port).
func loopbackHost(hostport string) bool {
	host := hostport
	if h, _, err := net.SplitHostPort(hostport); err == nil {
		host = h
	}
	host = strings.Trim(host, "[]")
	return host == "127.0.0.1" || host == "::1" || strings.EqualFold(host, "localhost")
}

// hostGuard rejects /api requests whose Host is not loopback. This blocks
// DNS rebinding, where a foreign name resolves to 127.0.0.1.
func (s *Server) hostGuard(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !loopbackHost(r.Host) {
			s.writeJSON(w, http.StatusForbidden, map[string]string{"error": "forbidden host"})
			return
		}
		next.ServeHTTP(w, r)
	})
}

// originGuard rejects mutating requests whose Origin is not the request's own
// loopback origin. This also admits the Vite dev proxy, which forwards both
// Origin and Host as localhost:5173. A missing Origin (non-browser client)
// is allowed unless Sec-Fetch-Site marks it as a cross-site browser request.
func (s *Server) originGuard(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodGet, http.MethodHead, http.MethodOptions:
			next.ServeHTTP(w, r)
			return
		}
		if origin := r.Header.Get("Origin"); origin != "" {
			u, err := url.Parse(origin)
			if err != nil || u.Scheme != "http" || !loopbackHost(u.Host) || !strings.EqualFold(u.Host, r.Host) {
				s.writeJSON(w, http.StatusForbidden, map[string]string{"error": "forbidden origin"})
				return
			}
		} else if site := r.Header.Get("Sec-Fetch-Site"); site != "" && site != "same-origin" && site != "none" {
			// Browser request without Origin: trust only same-origin or
			// user-initiated fetches.
			s.writeJSON(w, http.StatusForbidden, map[string]string{"error": "forbidden origin"})
			return
		}
		next.ServeHTTP(w, r)
	})
}

// apiGuard applies the host and origin checks to the /api subtree only.
func (s *Server) apiGuard(next http.Handler) http.Handler {
	guarded := s.hostGuard(s.originGuard(next))
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.HasPrefix(r.URL.Path, "/api/") {
			guarded.ServeHTTP(w, r)
			return
		}
		next.ServeHTTP(w, r)
	})
}
