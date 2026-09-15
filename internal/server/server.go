// Package server creates an http server with routes.
package server

import (
	"encoding/json"
	"io/fs"
	"log/slog"
	"net/http"
	"path"
	"strings"
)

// Server holds the dependencies shared by all handlers.
type Server struct {
	log    *slog.Logger
	assets fs.FS // nil when the frontend is not embedded
}

// New returns a Server.
func New(log *slog.Logger, assets fs.FS) *Server {
	if log == nil {
		log = slog.Default()
	}
	return &Server{log: log, assets: assets}
}

// Routes returns the fully wired HTTP handler.
func (s *Server) Routes() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /api/health", s.handleHealth)
	mux.HandleFunc("/api/", s.handleAPINotFound)

	if s.assets != nil {
		mux.Handle("/", s.spaHandler(s.assets))
	} else {
		mux.HandleFunc("/", s.handleNoUI)
	}

	return mux
}

func (s *Server) handleAPINotFound(w http.ResponseWriter, r *http.Request) {
	s.writeJSON(w, http.StatusNotFound, map[string]string{"error": "not found"})
}

func (s *Server) handleNoUI(w http.ResponseWriter, r *http.Request) {
	http.Error(w, "frontend not embedded in this build; run the Vite dev server on :5173", http.StatusNotFound)
}

func (s *Server) spaHandler(assets fs.FS) http.Handler {
	fileServer := http.FileServerFS(assets)

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet && r.Method != http.MethodHead {
			w.Header().Set("Allow", "GET, HEAD")
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}

		name := strings.TrimPrefix(path.Clean(r.URL.Path), "/")

		if info, err := fs.Stat(assets, name); err == nil && !info.IsDir() {
			if strings.HasPrefix(name, "assets/") {
				// Vite fingerprints these filenames, so they can never go stale.
				w.Header().Set("Cache-Control", "public, max-age=31536000, immutable")
			}
			fileServer.ServeHTTP(w, r)
			return
		}

		// Unknown path: hand it to the client-side router
		w.Header().Set("Cache-Control", "no-cache")
		http.ServeFileFS(w, r, assets, "index.html")
	})
}

func (s *Server) handleHealth(w http.ResponseWriter, r *http.Request) {
	s.writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

func (s *Server) writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(v); err != nil {
		// Status and headers are already on the wire, so all we can do is log it.
		s.log.Error("encoding response failed", "err", err)
	}
}
