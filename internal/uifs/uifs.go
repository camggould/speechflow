// Package uifs embeds speechflow's compiled UI assets and exposes an
// http.Handler that serves them, with SPA fallback to index.html for
// non-API routes. The dist directory is populated by `make ui`; a
// `.gitkeep` placeholder keeps the package compilable on a fresh checkout.
package uifs

import (
	"embed"
	"io/fs"
	"net/http"
	"strings"
)

//go:embed all:dist
var distFS embed.FS

// FS returns the embedded filesystem rooted at the dist/ directory.
func FS() fs.FS {
	sub, err := fs.Sub(distFS, "dist")
	if err != nil {
		// Should never fail since dist/ is always embedded.
		panic("uifs: failed to sub dist: " + err.Error())
	}
	return sub
}

// Handler returns an http.Handler that serves the embedded UI assets.
// Requests under /api/ are returned as 404 so the caller's router can
// take precedence. Unmatched paths fall back to index.html for SPA
// client-side routing.
func Handler() http.Handler {
	uiFS := FS()
	fileServer := http.FileServer(http.FS(uiFS))

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Never serve /api/ paths from the UI embed.
		if strings.HasPrefix(r.URL.Path, "/api/") {
			http.NotFound(w, r)
			return
		}

		// Strip /ui/ prefix before looking up the file.
		path := strings.TrimPrefix(r.URL.Path, "/ui")
		if path == "" {
			path = "/"
		}

		// Try to open the file. If it doesn't exist, serve index.html.
		if path != "/" {
			cleaned := strings.TrimPrefix(path, "/")
			if cleaned == "" {
				cleaned = "."
			}
			f, err := uiFS.Open(cleaned)
			if err != nil {
				serveIndexHTML(w, r, uiFS)
				return
			}
			_ = f.Close()
		}

		r2 := r.Clone(r.Context())
		r2.URL.Path = path
		fileServer.ServeHTTP(w, r2)
	})
}

// serveIndexHTML reads index.html from the embedded FS and writes it to w.
func serveIndexHTML(w http.ResponseWriter, _ *http.Request, uiFS fs.FS) {
	data, err := fs.ReadFile(uiFS, "index.html")
	if err != nil {
		http.Error(w, "UI not built. Run: make ui", http.StatusServiceUnavailable)
		return
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Header().Set("Cache-Control", "no-cache")
	_, _ = w.Write(data)
}
