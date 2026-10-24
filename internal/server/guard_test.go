package server

import (
	"net/http/httptest"
	"testing"
)

func TestGuards(t *testing.T) {
	tests := []struct {
		name, method, path, host, origin string
		want                             int
	}{
		{"get ok", "GET", "/api/health", "127.0.0.1:7777", "", 200},
		{"get foreign host", "GET", "/api/health", "evil.example:7777", "", 403},
		{"get localhost", "GET", "/api/health", "localhost:7777", "", 200},
		{"ui foreign host untouched", "GET", "/", "evil.example", "", 200},
		{"post same origin", "POST", "/api/project/close", "127.0.0.1:7777", "http://127.0.0.1:7777", 204},
		{"post localhost origin", "POST", "/api/project/close", "localhost:7777", "http://localhost:7777", 204},
		{"post vite dev origin", "POST", "/api/project/close", "localhost:5173", "http://localhost:5173", 204},
		{"post missing origin", "POST", "/api/project/close", "127.0.0.1:7777", "", 204},
		{"post foreign origin", "POST", "/api/project/close", "127.0.0.1:7777", "https://evil.example", 403},
		{"post other loopback port", "POST", "/api/project/close", "127.0.0.1:7777", "http://localhost:3000", 403},
		{"post foreign host", "POST", "/api/project/close", "evil.example:7777", "http://evil.example:7777", 403},
		{"post null origin", "POST", "/api/project/close", "127.0.0.1:7777", "null", 403},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(tt.method, tt.path, nil)
			req.Host = tt.host
			if tt.origin != "" {
				req.Header.Set("Origin", tt.origin)
			}
			rec := httptest.NewRecorder()
			newTestServer().Routes().ServeHTTP(rec, req)
			if rec.Code != tt.want {
				t.Errorf("status = %d, want %d", rec.Code, tt.want)
			}
		})
	}
}

func TestSecFetchSiteWithoutOrigin(t *testing.T) {
	tests := []struct {
		site string
		want int
	}{
		{"", 204},
		{"same-origin", 204},
		{"none", 204},
		{"cross-site", 403},
		{"same-site", 403},
	}
	for _, tt := range tests {
		t.Run("site="+tt.site, func(t *testing.T) {
			req := httptest.NewRequest("POST", "/api/project/close", nil)
			req.Host = "127.0.0.1:7777"
			if tt.site != "" {
				req.Header.Set("Sec-Fetch-Site", tt.site)
			}
			rec := httptest.NewRecorder()
			newTestServer().Routes().ServeHTTP(rec, req)
			if rec.Code != tt.want {
				t.Errorf("status = %d, want %d", rec.Code, tt.want)
			}
		})
	}
}
