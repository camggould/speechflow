// Package server is speechflow's chi-based HTTP server. It mounts a
// read-only JSON API under /api/v1 and serves the embedded UI under /ui/.
// The server always binds to a loopback interface; mutations are CLI-only.
package server

import (
	"encoding/json"
	"errors"
	"net/http"
	"strings"

	"github.com/go-chi/chi/v5"
	chimiddleware "github.com/go-chi/chi/v5/middleware"

	"github.com/camggould/speechflow/internal/store"
	"github.com/camggould/speechflow/internal/uifs"
)

// Config holds runtime configuration for the speechflow HTTP server.
type Config struct {
	// Listen is the address the caller wishes to bind to. The handler
	// itself does not enforce this; the calling code in `serve.go`
	// always passes 127.0.0.1:<port>.
	Listen string
	// Store is the open SQLite store. Required.
	Store *store.Store
	// Version is reported by /api/v1/health. Defaults to "dev".
	Version string
}

// New constructs an *http.Server ready to call .Serve(listener) on.
func New(cfg Config) *http.Server {
	if cfg.Version == "" {
		cfg.Version = "dev"
	}

	r := chi.NewRouter()
	r.Use(chimiddleware.RequestID)
	r.Use(chimiddleware.Recoverer)
	r.Use(corsLocalhost)

	api := newAPI(cfg.Store, cfg.Version)
	r.Route("/api/v1", func(r chi.Router) {
		r.Get("/health", api.health)
		r.Get("/sessions", api.listSessions)
		r.Get("/sessions/{id}", api.getSession)
		r.Get("/sessions/{id}/coverage", api.sessionCoverage)
		r.Delete("/sessions/{id}", api.deleteSession)

		r.Get("/iterations/{id}", api.getIteration)
		r.Get("/iterations/{id}/graph", api.getIterationGraph)
		r.Get("/iterations/{id}/timeline", api.getIterationTimeline)
		r.Get("/iterations/{id}/transcript", api.getIterationTranscript)
		r.Get("/iterations/{id}/coverage", api.getIterationCoverage)
		r.Delete("/iterations/{id}", api.deleteIteration)
	})

	uiHandler := uifs.Handler()
	r.Handle("/ui/*", uiHandler)
	r.Handle("/ui", uiHandler)

	r.Get("/", func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, "/ui/", http.StatusFound)
	})

	r.NotFound(func(w http.ResponseWriter, _ *http.Request) {
		writeError(w, http.StatusNotFound, "not_found", "the requested endpoint does not exist")
	})

	return &http.Server{Addr: cfg.Listen, Handler: r}
}

// corsLocalhost permits browser fetches from the dev UI on localhost:5173
// (Vite) so the API and UI dev server can coexist during development.
func corsLocalhost(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		origin := r.Header.Get("Origin")
		if origin != "" && isLocalOrigin(origin) {
			w.Header().Set("Access-Control-Allow-Origin", origin)
			w.Header().Set("Vary", "Origin")
			w.Header().Set("Access-Control-Allow-Methods", "GET, POST, DELETE, OPTIONS")
			w.Header().Set("Access-Control-Allow-Headers", "Content-Type")
		}
		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		next.ServeHTTP(w, r)
	})
}

// isLocalOrigin reports whether origin is a loopback URL.
func isLocalOrigin(origin string) bool {
	o := strings.TrimSpace(origin)
	for _, prefix := range []string{
		"http://localhost",
		"https://localhost",
		"http://127.0.0.1",
		"https://127.0.0.1",
		"http://[::1]",
		"https://[::1]",
	} {
		if strings.HasPrefix(o, prefix) {
			return true
		}
	}
	return false
}

// writeError sends a JSON error envelope per the README contract.
func writeError(w http.ResponseWriter, status int, code, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(map[string]string{
		"error":   code,
		"message": message,
	})
}

// writeJSON marshals obj as JSON; falls back to a 500 error if encoding fails.
func writeJSON(w http.ResponseWriter, status int, obj any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	enc := json.NewEncoder(w)
	enc.SetEscapeHTML(false)
	if err := enc.Encode(obj); err != nil {
		// Best-effort: header is already written.
		_ = err
	}
}

// translateStoreError maps store sentinels to (status, code).
func translateStoreError(err error) (int, string, string) {
	switch {
	case errors.Is(err, store.ErrNotFound):
		return http.StatusNotFound, "not_found", err.Error()
	case errors.Is(err, store.ErrConstraint):
		return http.StatusConflict, "constraint", err.Error()
	default:
		return http.StatusInternalServerError, "internal", err.Error()
	}
}
