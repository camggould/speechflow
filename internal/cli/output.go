// Output helpers: JSON encoding by default, pretty rendering on --pretty.
package cli

import (
	"encoding/json"
	"fmt"
	"io"
	"strings"

	"github.com/spf13/cobra"

	"github.com/camggould/speechflow/internal/core"
)

// emit writes obj as JSON (pretty-indented) to the command's stdout. When
// --pretty is set, it falls back to a type-aware short rendering for the
// well-known speechflow types and JSON for everything else.
func emit(cmd *cobra.Command, obj any) error {
	w := cmd.OutOrStdout()
	if pretty(cmd) {
		if rendered, ok := renderPretty(w, obj); ok {
			return rendered
		}
	}
	return writeJSON(w, obj)
}

// writeJSON encodes obj as indented JSON.
func writeJSON(w io.Writer, obj any) error {
	enc := json.NewEncoder(w)
	enc.SetIndent("", "  ")
	enc.SetEscapeHTML(false)
	return enc.Encode(obj)
}

// renderPretty handles a small set of well-known types. Returns
// (err, true) when handled or (nil, false) when the caller should fall
// back to JSON encoding.
func renderPretty(w io.Writer, obj any) (error, bool) {
	switch v := obj.(type) {
	case *core.Session:
		return renderSession(w, *v), true
	case core.Session:
		return renderSession(w, v), true
	case []core.Session:
		return renderSessions(w, v), true
	case *core.Root:
		return renderRoot(w, *v), true
	case []core.Root:
		return renderRoots(w, v), true
	case *core.Iteration:
		return renderIteration(w, *v), true
	case []core.Iteration:
		return renderIterations(w, v), true
	case *core.Node:
		return renderNode(w, *v), true
	case []core.Node:
		return renderNodes(w, v), true
	case *core.Edge:
		return renderEdge(w, *v), true
	case []core.Edge:
		return renderEdges(w, v), true
	case []core.CoverageRow:
		return renderCoverage(w, v), true
	case *core.CoverageMatrix:
		return renderMatrix(w, *v), true
	case []core.TimelineEvent:
		return renderTimeline(w, v), true
	}
	return nil, false
}

func renderSession(w io.Writer, s core.Session) error {
	fmt.Fprintf(w, "session %s\n", s.ID)
	fmt.Fprintf(w, "  title:        %s\n", s.Title)
	if s.Description != nil {
		fmt.Fprintf(w, "  description:  %s\n", *s.Description)
	}
	fmt.Fprintf(w, "  iterations:   %d\n", s.IterationCount)
	fmt.Fprintf(w, "  coverage:     %.0f%%\n", s.LatestCoveragePct*100)
	fmt.Fprintf(w, "  last_active:  %s\n", s.LastActivityAt.Format("2006-01-02 15:04:05"))
	return nil
}

func renderSessions(w io.Writer, list []core.Session) error {
	for _, s := range list {
		fmt.Fprintf(w, "%-30s %-3d iterations  %3.0f%%  %s\n",
			s.ID, s.IterationCount, s.LatestCoveragePct*100,
			s.LastActivityAt.Format("2006-01-02 15:04"))
	}
	return nil
}

func renderRoot(w io.Writer, r core.Root) error {
	fmt.Fprintf(w, "root %s\n  title: %s\n  session: %s\n", r.ID, r.Title, r.SessionID)
	return nil
}

func renderRoots(w io.Writer, list []core.Root) error {
	for _, r := range list {
		fmt.Fprintf(w, "%-30s %s\n", r.ID, r.Title)
	}
	return nil
}

func renderIteration(w io.Writer, it core.Iteration) error {
	fmt.Fprintf(w, "iteration %s\n", it.ID)
	fmt.Fprintf(w, "  session:  %s\n", it.SessionID)
	fmt.Fprintf(w, "  title:    %s\n", it.Title)
	fmt.Fprintf(w, "  started:  %s\n", it.StartedAt.Format("2006-01-02 15:04:05"))
	if it.EndedAt != nil {
		fmt.Fprintf(w, "  ended:    %s\n", it.EndedAt.Format("2006-01-02 15:04:05"))
	} else {
		fmt.Fprintln(w, "  ended:    (active)")
	}
	fmt.Fprintf(w, "  nodes:    %d\n", it.NodeCount)
	fmt.Fprintf(w, "  coverage: %.0f%%\n", it.CoveragePct*100)
	return nil
}

func renderIterations(w io.Writer, list []core.Iteration) error {
	for _, it := range list {
		state := "active"
		if it.EndedAt != nil {
			state = "ended"
		}
		fmt.Fprintf(w, "%-30s %-6s  %d nodes  %3.0f%%  %s\n",
			it.ID, state, it.NodeCount, it.CoveragePct*100,
			it.StartedAt.Format("2006-01-02 15:04"))
	}
	return nil
}

func renderNode(w io.Writer, n core.Node) error {
	fmt.Fprintf(w, "node %s\n", n.ID)
	fmt.Fprintf(w, "  kind:  %s\n", n.Kind)
	fmt.Fprintf(w, "  title: %s\n", n.Title)
	if n.Quote != nil {
		fmt.Fprintf(w, "  quote: %s\n", *n.Quote)
	}
	if n.RootID != nil {
		fmt.Fprintf(w, "  root:  %s\n", *n.RootID)
	}
	if n.ResolvedByNodeID != nil {
		fmt.Fprintf(w, "  resolved_by: %s\n", *n.ResolvedByNodeID)
	}
	if len(n.Tags) > 0 {
		fmt.Fprintf(w, "  tags:  %s\n", strings.Join(n.Tags, ", "))
	}
	return nil
}

func renderNodes(w io.Writer, list []core.Node) error {
	for _, n := range list {
		fmt.Fprintf(w, "%-30s %-10s %s\n", n.ID, n.Kind, n.Title)
	}
	return nil
}

func renderEdge(w io.Writer, e core.Edge) error {
	fmt.Fprintf(w, "edge %s\n  %s -[%s]-> %s\n", e.ID, e.FromNode, e.Kind, e.ToNode)
	return nil
}

func renderEdges(w io.Writer, list []core.Edge) error {
	for _, e := range list {
		fmt.Fprintf(w, "%-30s %s -[%s]-> %s\n", e.ID, e.FromNode, e.Kind, e.ToNode)
	}
	return nil
}

func renderCoverage(w io.Writer, rows []core.CoverageRow) error {
	for _, r := range rows {
		mark := "x"
		if r.Covered {
			mark = "v"
		}
		fmt.Fprintf(w, "[%s] %-30s %d supporting nodes\n", mark, r.RootID, len(r.SupportingNodeIDs))
	}
	return nil
}

func renderMatrix(w io.Writer, m core.CoverageMatrix) error {
	// Header.
	fmt.Fprintf(w, "session %s\n", m.SessionID)
	if len(m.Roots) == 0 {
		fmt.Fprintln(w, "  (no roots declared)")
		return nil
	}
	header := "iteration"
	for _, r := range m.Roots {
		header += "\t" + r.ID
	}
	fmt.Fprintln(w, header)
	for _, row := range m.Iterations {
		line := row.IterationID
		marks := map[string]string{}
		for _, c := range row.Rows {
			if c.Covered {
				marks[c.RootID] = "v"
			} else {
				marks[c.RootID] = "x"
			}
		}
		for _, r := range m.Roots {
			m := marks[r.ID]
			if m == "" {
				m = "-"
			}
			line += "\t" + m
		}
		fmt.Fprintln(w, line)
	}
	return nil
}

func renderTimeline(w io.Writer, list []core.TimelineEvent) error {
	for _, e := range list {
		target := ""
		if e.NodeID != nil {
			target = *e.NodeID
		} else if e.EdgeID != nil {
			target = *e.EdgeID
		}
		fmt.Fprintf(w, "%s  %-20s  %s\n", e.Ts.Format("2006-01-02 15:04:05"), e.Kind, target)
	}
	return nil
}
