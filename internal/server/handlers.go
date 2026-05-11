// HTTP handlers for speechflow's read-only /api/v1 surface.
package server

import (
	"net/http"

	"github.com/go-chi/chi/v5"

	"github.com/camggould/speechflow/internal/core"
	"github.com/camggould/speechflow/internal/coverage"
	"github.com/camggould/speechflow/internal/store"
)

// api carries the dependencies every handler needs.
type api struct {
	store   *store.Store
	version string
}

func newAPI(s *store.Store, version string) *api {
	return &api{store: s, version: version}
}

func (a *api) health(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, map[string]string{
		"status":  "ok",
		"version": a.version,
	})
}

func (a *api) listSessions(w http.ResponseWriter, _ *http.Request) {
	list, err := a.store.ListSessions()
	if err != nil {
		status, code, msg := translateStoreError(err)
		writeError(w, status, code, msg)
		return
	}
	for i := range list {
		// Best-effort decoration; coverage failure shouldn't 500 the dashboard.
		iters, _ := a.store.ListIterations(list[i].ID)
		if len(iters) > 0 {
			rows, err := coverage.Compute(a.store, iters[0].ID)
			if err == nil {
				list[i].LatestCoveragePct = coverage.Percent(rows)
			}
		}
	}
	if list == nil {
		list = []core.Session{}
	}
	writeJSON(w, http.StatusOK, list)
}

func (a *api) getSession(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	sess, err := a.store.GetSession(id)
	if err != nil {
		status, code, msg := translateStoreError(err)
		writeError(w, status, code, msg)
		return
	}
	roots, err := a.store.ListRoots(id)
	if err != nil {
		status, code, msg := translateStoreError(err)
		writeError(w, status, code, msg)
		return
	}
	iters, err := a.store.ListIterations(id)
	if err != nil {
		status, code, msg := translateStoreError(err)
		writeError(w, status, code, msg)
		return
	}
	for i := range iters {
		rows, err := coverage.Compute(a.store, iters[i].ID)
		if err == nil {
			iters[i].CoveragePct = coverage.Percent(rows)
		}
	}
	if len(iters) > 0 {
		sess.LatestCoveragePct = iters[0].CoveragePct
	}
	if roots == nil {
		roots = []core.Root{}
	}
	if iters == nil {
		iters = []core.Iteration{}
	}
	writeJSON(w, http.StatusOK, core.SessionDetail{
		Session:    *sess,
		Roots:      roots,
		Iterations: iters,
	})
}

// flattenIterationDetail wraps an Iteration with the roots that were
// in scope at iteration end. JSON marshals to a flat object thanks to the
// embedded Iteration in core.IterationDetail.
func flattenIterationDetail(it *core.Iteration, roots []core.Root) core.IterationDetail {
	return core.IterationDetail{Iteration: *it, Roots: roots}
}

func (a *api) sessionCoverage(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	if _, err := a.store.GetSession(id); err != nil {
		status, code, msg := translateStoreError(err)
		writeError(w, status, code, msg)
		return
	}
	m, err := coverage.Matrix(a.store, id)
	if err != nil {
		status, code, msg := translateStoreError(err)
		writeError(w, status, code, msg)
		return
	}
	writeJSON(w, http.StatusOK, m)
}

func (a *api) deleteSession(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	if err := a.store.DeleteSession(id); err != nil {
		status, code, msg := translateStoreError(err)
		writeError(w, status, code, msg)
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"deleted": id})
}

func (a *api) getIteration(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	it, err := a.store.GetIteration(id)
	if err != nil {
		status, code, msg := translateStoreError(err)
		writeError(w, status, code, msg)
		return
	}
	rows, err := coverage.Compute(a.store, id)
	if err == nil {
		it.CoveragePct = coverage.Percent(rows)
	}
	// Roots-at-the-time = roots created at or before iteration's effective end.
	roots, err := a.store.ListRoots(it.SessionID)
	if err != nil {
		status, code, msg := translateStoreError(err)
		writeError(w, status, code, msg)
		return
	}
	cutoff := it.StartedAt
	if it.EndedAt != nil {
		cutoff = *it.EndedAt
	}
	filtered := make([]core.Root, 0, len(roots))
	for _, r := range roots {
		if !r.CreatedAt.After(cutoff) {
			filtered = append(filtered, r)
		}
	}
	writeJSON(w, http.StatusOK, flattenIterationDetail(it, filtered))
}

func (a *api) getIterationGraph(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	if _, err := a.store.GetIteration(id); err != nil {
		status, code, msg := translateStoreError(err)
		writeError(w, status, code, msg)
		return
	}
	nodes, err := a.store.ListNodes(id)
	if err != nil {
		status, code, msg := translateStoreError(err)
		writeError(w, status, code, msg)
		return
	}
	edges, err := a.store.ListEdges(id)
	if err != nil {
		status, code, msg := translateStoreError(err)
		writeError(w, status, code, msg)
		return
	}
	if nodes == nil {
		nodes = []core.Node{}
	}
	if edges == nil {
		edges = []core.Edge{}
	}
	writeJSON(w, http.StatusOK, core.Graph{Nodes: nodes, Edges: edges})
}

func (a *api) getIterationTimeline(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	if _, err := a.store.GetIteration(id); err != nil {
		status, code, msg := translateStoreError(err)
		writeError(w, status, code, msg)
		return
	}
	events, err := a.store.Timeline(id)
	if err != nil {
		status, code, msg := translateStoreError(err)
		writeError(w, status, code, msg)
		return
	}
	if events == nil {
		events = []core.TimelineEvent{}
	}
	writeJSON(w, http.StatusOK, events)
}

func (a *api) getIterationTranscript(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	tr, err := a.store.GetTranscript(id)
	if err != nil {
		status, code, msg := translateStoreError(err)
		writeError(w, status, code, msg)
		return
	}
	if tr.Spans == nil {
		tr.Spans = []core.TranscriptSpan{}
	}
	writeJSON(w, http.StatusOK, tr)
}

func (a *api) getIterationCoverage(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	rows, err := coverage.Compute(a.store, id)
	if err != nil {
		status, code, msg := translateStoreError(err)
		writeError(w, status, code, msg)
		return
	}
	if rows == nil {
		rows = []core.CoverageRow{}
	}
	writeJSON(w, http.StatusOK, rows)
}

func (a *api) deleteIteration(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	if err := a.store.DeleteIteration(id); err != nil {
		status, code, msg := translateStoreError(err)
		writeError(w, status, code, msg)
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"deleted": id})
}
