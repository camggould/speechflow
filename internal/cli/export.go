// `speechflow export ...`: JSON or GraphML dumps.
package cli

import (
	"encoding/xml"
	"fmt"

	"github.com/spf13/cobra"

	"github.com/camggould/speechflow/internal/core"
	"github.com/camggould/speechflow/internal/store"
)

func newExportCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "export",
		Short: "Export data in JSON or GraphML",
	}
	cmd.AddCommand(newExportJSONCommand())
	cmd.AddCommand(newExportGraphMLCommand())
	return cmd
}

// iterationDump is the JSON shape for `export json --iteration <slug>`.
type iterationDump struct {
	Iteration  *core.Iteration `json:"iteration"`
	Nodes      []core.Node     `json:"nodes"`
	Edges      []core.Edge     `json:"edges"`
	Transcript string          `json:"transcript"`
}

// sessionDump is the JSON shape for `export json` (no --iteration).
type sessionDump struct {
	Session    *core.Session   `json:"session"`
	Roots      []core.Root     `json:"roots"`
	Iterations []iterationDump `json:"iterations"`
}

func newExportJSONCommand() *cobra.Command {
	var iteration string
	cmd := &cobra.Command{
		Use:   "json",
		Short: "Export the active session (or one iteration) as JSON",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			s, err := openStore(cmd)
			if err != nil {
				return Exit(ExitGeneric, "open store: %v", err)
			}
			defer s.Close()
			if iteration != "" {
				dump, err := buildIterationDump(s, iteration)
				if err != nil {
					return translateStoreErr(err)
				}
				return emit(cmd, dump)
			}
			sess, err := activeSession()
			if err != nil {
				return err
			}
			session, err := s.GetSession(sess)
			if err != nil {
				return translateStoreErr(err)
			}
			roots, err := s.ListRoots(sess)
			if err != nil {
				return translateStoreErr(err)
			}
			iters, err := s.ListIterations(sess)
			if err != nil {
				return translateStoreErr(err)
			}
			out := sessionDump{Session: session, Roots: roots}
			for _, it := range iters {
				d, err := buildIterationDump(s, it.ID)
				if err != nil {
					return translateStoreErr(err)
				}
				out.Iterations = append(out.Iterations, *d)
			}
			return emit(cmd, out)
		},
	}
	cmd.Flags().StringVar(&iteration, "iteration", "", "Export a single iteration by slug")
	return cmd
}

func buildIterationDump(s *store.Store, id string) (*iterationDump, error) {
	it, err := s.GetIteration(id)
	if err != nil {
		return nil, err
	}
	nodes, err := s.ListNodes(id)
	if err != nil {
		return nil, err
	}
	edges, err := s.ListEdges(id)
	if err != nil {
		return nil, err
	}
	tr, err := s.GetTranscript(id)
	if err != nil {
		return nil, err
	}
	return &iterationDump{
		Iteration:  it,
		Nodes:      nodes,
		Edges:      edges,
		Transcript: tr.Text,
	}, nil
}

// --- GraphML --------------------------------------------------------------

type graphML struct {
	XMLName xml.Name  `xml:"graphml"`
	Xmlns   string    `xml:"xmlns,attr"`
	Graph   graphMLG  `xml:"graph"`
}

type graphMLG struct {
	ID       string       `xml:"id,attr"`
	EdgeMode string       `xml:"edgedefault,attr"`
	Nodes    []graphMLN   `xml:"node"`
	Edges    []graphMLE   `xml:"edge"`
}

type graphMLN struct {
	ID   string `xml:"id,attr"`
	Kind string `xml:"data>kind,omitempty"`
}

type graphMLE struct {
	ID     string `xml:"id,attr"`
	Source string `xml:"source,attr"`
	Target string `xml:"target,attr"`
	Kind   string `xml:"data>kind,omitempty"`
}

func newExportGraphMLCommand() *cobra.Command {
	var iteration string
	cmd := &cobra.Command{
		Use:   "graphml",
		Short: "Export an iteration as GraphML",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			if iteration == "" {
				return Exit(ExitUsage, "--iteration is required")
			}
			s, err := openStore(cmd)
			if err != nil {
				return Exit(ExitGeneric, "open store: %v", err)
			}
			defer s.Close()
			nodes, err := s.ListNodes(iteration)
			if err != nil {
				return translateStoreErr(err)
			}
			edges, err := s.ListEdges(iteration)
			if err != nil {
				return translateStoreErr(err)
			}
			g := graphML{
				Xmlns: "http://graphml.graphdrawing.org/xmlns",
				Graph: graphMLG{
					ID:       iteration,
					EdgeMode: "directed",
				},
			}
			for _, n := range nodes {
				g.Graph.Nodes = append(g.Graph.Nodes, graphMLN{ID: n.ID, Kind: string(n.Kind)})
			}
			for _, e := range edges {
				g.Graph.Edges = append(g.Graph.Edges, graphMLE{ID: e.ID, Source: e.FromNode, Target: e.ToNode, Kind: string(e.Kind)})
			}
			data, err := xml.MarshalIndent(g, "", "  ")
			if err != nil {
				return Exit(ExitGeneric, "marshal graphml: %v", err)
			}
			fmt.Fprintln(cmd.OutOrStdout(), xml.Header+string(data))
			return nil
		},
	}
	cmd.Flags().StringVar(&iteration, "iteration", "", "Iteration slug to export (required)")
	return cmd
}
