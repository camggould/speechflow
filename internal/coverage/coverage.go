// Package coverage computes the structural coverage of an iteration
// against its session's roots. The algorithm is purely graph-based:
// a root is covered iff at least one node in the iteration can reach
// (via any directed edge) a root_ref node pointing at that root.
//
// "Reach" is computed by reverse BFS from each root_ref node along the
// edges directed *toward* it. Concretely: if there is an edge from N to
// a root_ref R, then N supports R. Transitively, anything reaching N
// also supports R. The whole walk follows edges backward.
package coverage

import (
	"time"

	"github.com/camggould/speechflow/internal/core"
	"github.com/camggould/speechflow/internal/store"
)

// Compute returns one CoverageRow per session root for the iteration,
// per the algorithm described in the README. The iteration must exist.
func Compute(s *store.Store, iterationID string) ([]core.CoverageRow, error) {
	it, err := s.GetIteration(iterationID)
	if err != nil {
		return nil, err
	}
	roots, err := s.ListRoots(it.SessionID)
	if err != nil {
		return nil, err
	}
	nodes, err := s.ListNodes(iterationID)
	if err != nil {
		return nil, err
	}
	edges, err := s.ListEdges(iterationID)
	if err != nil {
		return nil, err
	}
	return computeFromData(it, roots, nodes, edges), nil
}

// computeFromData is the pure-function core, separated out for unit testing.
func computeFromData(it *core.Iteration, roots []core.Root, nodes []core.Node, edges []core.Edge) []core.CoverageRow {
	// Time horizon: include only roots created at or before the
	// iteration's effective end (ended_at or now).
	cutoff := time.Now().UTC()
	if it.EndedAt != nil {
		cutoff = *it.EndedAt
	}

	// Index nodes by ID for createdAt lookups.
	nodeByID := make(map[string]core.Node, len(nodes))
	for _, n := range nodes {
		nodeByID[n.ID] = n
	}

	// Reverse adjacency: for each node id, the set of nodes that have
	// an edge directly *into* it.
	rev := make(map[string][]string, len(nodes))
	for _, e := range edges {
		rev[e.ToNode] = append(rev[e.ToNode], e.FromNode)
	}

	rows := make([]core.CoverageRow, 0, len(roots))
	for _, r := range roots {
		if r.CreatedAt.After(cutoff) {
			continue
		}
		// Seed: every root_ref node pointing at this root.
		var seeds []string
		for _, n := range nodes {
			if n.Kind == core.NodeKindRootRef && n.RootID != nil && *n.RootID == r.ID {
				seeds = append(seeds, n.ID)
			}
		}

		// BFS over reverse edges from each seed; collect the union.
		supportSet := map[string]struct{}{}
		queue := make([]string, 0, len(seeds))
		for _, sid := range seeds {
			if _, ok := supportSet[sid]; !ok {
				supportSet[sid] = struct{}{}
				queue = append(queue, sid)
			}
		}
		for len(queue) > 0 {
			cur := queue[0]
			queue = queue[1:]
			for _, parent := range rev[cur] {
				if _, seen := supportSet[parent]; seen {
					continue
				}
				supportSet[parent] = struct{}{}
				queue = append(queue, parent)
			}
		}

		// Preserve insertion order matching nodes slice (creation order).
		supporting := make([]string, 0, len(supportSet))
		var firstTouched *time.Time
		for _, n := range nodes {
			if _, ok := supportSet[n.ID]; !ok {
				continue
			}
			supporting = append(supporting, n.ID)
			if firstTouched == nil || n.CreatedAt.Before(*firstTouched) {
				ts := n.CreatedAt
				firstTouched = &ts
			}
		}

		rows = append(rows, core.CoverageRow{
			RootID:            r.ID,
			RootTitle:         r.Title,
			Covered:           len(supporting) > 0,
			SupportingNodeIDs: supporting,
			FirstTouchedAt:    firstTouched,
		})
	}
	return rows
}

// Percent returns the fraction of rows where Covered is true. 0 if empty.
func Percent(rows []core.CoverageRow) float64 {
	if len(rows) == 0 {
		return 0
	}
	covered := 0
	for _, r := range rows {
		if r.Covered {
			covered++
		}
	}
	return float64(covered) / float64(len(rows))
}

// Matrix builds the session-level coverage matrix: one row per iteration
// with one CoverageRow per root.
func Matrix(s *store.Store, sessionID string) (*core.CoverageMatrix, error) {
	roots, err := s.ListRoots(sessionID)
	if err != nil {
		return nil, err
	}
	iters, err := s.ListIterations(sessionID)
	if err != nil {
		return nil, err
	}
	out := &core.CoverageMatrix{
		SessionID:  sessionID,
		Roots:      roots,
		Iterations: make([]core.CoverageMatrixRow, 0, len(iters)),
	}
	for _, it := range iters {
		rows, err := Compute(s, it.ID)
		if err != nil {
			return nil, err
		}
		out.Iterations = append(out.Iterations, core.CoverageMatrixRow{
			IterationID:    it.ID,
			IterationTitle: it.Title,
			StartedAt:      it.StartedAt,
			Rows:           rows,
		})
	}
	return out, nil
}
