// `speechflow node ...`: graph node creation, tagging, resolution.
package cli

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/spf13/cobra"

	"github.com/camggould/speechflow/internal/core"
	"github.com/camggould/speechflow/internal/slug"
	"github.com/camggould/speechflow/internal/store"
)

func newNodeCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "node",
		Short: "Manage nodes on the active iteration",
	}
	cmd.AddCommand(newNodeAddCommand())
	cmd.AddCommand(newNodeTouchRootCommand())
	cmd.AddCommand(newNodeResolveCommand())
	cmd.AddCommand(newNodeTagCommand())
	cmd.AddCommand(newNodeUntagCommand())
	cmd.AddCommand(newNodeDeleteCommand())
	return cmd
}

// parseSpan parses "S,E" into (start, end). Returns nil pointers if empty.
func parseSpan(span string) (*int, *int, error) {
	if span == "" {
		return nil, nil, nil
	}
	parts := strings.SplitN(span, ",", 2)
	if len(parts) != 2 {
		return nil, nil, fmt.Errorf("--span must be 'S,E'")
	}
	s, err := strconv.Atoi(strings.TrimSpace(parts[0]))
	if err != nil {
		return nil, nil, fmt.Errorf("--span start: %w", err)
	}
	e, err := strconv.Atoi(strings.TrimSpace(parts[1]))
	if err != nil {
		return nil, nil, fmt.Errorf("--span end: %w", err)
	}
	return &s, &e, nil
}

func newNodeAddCommand() *cobra.Command {
	var (
		title, quote, span, from string
		tags                     []string
		refs                     []string
	)
	cmd := &cobra.Command{
		Use:   "add <kind>",
		Short: "Add a concept or curiosity node",
		Long:  "Add a node of kind 'concept' or 'curiosity'. Use `node touch-root` for root_ref nodes.",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			kind := core.NodeKind(args[0])
			if kind != core.NodeKindConcept && kind != core.NodeKindCuriosity {
				return Exit(ExitUsage, "kind must be 'concept' or 'curiosity'")
			}
			if title == "" {
				return Exit(ExitUsage, "--title is required")
			}
			if kind == core.NodeKindCuriosity && from == "" {
				return Exit(ExitUsage, "curiosity nodes require --from <slug>")
			}
			iter, err := activeIteration()
			if err != nil {
				return err
			}
			s, err := openStore(cmd)
			if err != nil {
				return Exit(ExitGeneric, "open store: %v", err)
			}
			defer s.Close()

			ts, te, err := parseSpan(span)
			if err != nil {
				return Exit(ExitUsage, "%v", err)
			}

			id, err := slug.Unique(title, func(c string) (bool, error) {
				return s.SlugExists("nodes", iter, c)
			})
			if err != nil {
				return Exit(ExitGeneric, "%v", err)
			}

			in := store.NodeInput{
				ID:              id,
				IterationID:     iter,
				Kind:            kind,
				Title:           title,
				TranscriptStart: ts,
				TranscriptEnd:   te,
				Tags:            tags,
				Source:          core.SourceAgent,
			}
			if quote != "" {
				in.Quote = &quote
			}
			node, err := s.CreateNode(in)
			if err != nil {
				return translateStoreErr(err)
			}

			// Wire up --from (branches_from) and --refs (references).
			if from != "" {
				eid, err := s.NextEdgeID(iter)
				if err != nil {
					return translateStoreErr(err)
				}
				if _, err := s.CreateEdge(eid, iter, node.ID, from, core.EdgeBranchesFrom); err != nil {
					return translateStoreErr(err)
				}
			}
			for _, r := range refs {
				r = strings.TrimSpace(r)
				if r == "" {
					continue
				}
				eid, err := s.NextEdgeID(iter)
				if err != nil {
					return translateStoreErr(err)
				}
				if _, err := s.CreateEdge(eid, iter, node.ID, r, core.EdgeReferences); err != nil {
					return translateStoreErr(err)
				}
			}
			return emit(cmd, node)
		},
	}
	cmd.Flags().StringVar(&title, "title", "", "Node title (required)")
	cmd.Flags().StringVar(&quote, "quote", "", "Optional quote from the transcript")
	cmd.Flags().StringVar(&span, "span", "", "Transcript span as 'start,end' char offsets")
	cmd.Flags().StringVar(&from, "from", "", "Parent node slug (creates branches_from edge)")
	cmd.Flags().StringSliceVar(&tags, "tag", nil, "Tag(s) to apply (repeatable or comma-separated)")
	cmd.Flags().StringSliceVar(&refs, "refs", nil, "Other node slug(s) this node references")
	return cmd
}

func newNodeTouchRootCommand() *cobra.Command {
	var span string
	cmd := &cobra.Command{
		Use:   "touch-root <root-slug>",
		Short: "Record that the active iteration touched a root",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			iter, err := activeIteration()
			if err != nil {
				return err
			}
			s, err := openStore(cmd)
			if err != nil {
				return Exit(ExitGeneric, "open store: %v", err)
			}
			defer s.Close()

			rootSlug := args[0]
			root, err := s.GetRoot(rootSlug)
			if err != nil {
				return translateStoreErr(err)
			}
			ts, te, err := parseSpan(span)
			if err != nil {
				return Exit(ExitUsage, "%v", err)
			}

			title := "touched: " + root.Title
			id, err := slug.Unique("touch-"+root.ID, func(c string) (bool, error) {
				return s.SlugExists("nodes", iter, c)
			})
			if err != nil {
				return Exit(ExitGeneric, "%v", err)
			}
			rid := root.ID
			node, err := s.CreateNode(store.NodeInput{
				ID:              id,
				IterationID:     iter,
				Kind:            core.NodeKindRootRef,
				Title:           title,
				RootID:          &rid,
				TranscriptStart: ts,
				TranscriptEnd:   te,
				Source:          core.SourceAgent,
			})
			if err != nil {
				return translateStoreErr(err)
			}
			return emit(cmd, node)
		},
	}
	cmd.Flags().StringVar(&span, "span", "", "Transcript span as 'start,end'")
	return cmd
}

func newNodeResolveCommand() *cobra.Command {
	var by string
	cmd := &cobra.Command{
		Use:   "resolve <curiosity-slug>",
		Short: "Mark a curiosity as resolved by another node",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if by == "" {
				return Exit(ExitUsage, "--by <node-slug> is required")
			}
			s, err := openStore(cmd)
			if err != nil {
				return Exit(ExitGeneric, "open store: %v", err)
			}
			defer s.Close()
			n, err := s.ResolveCuriosity(args[0], by)
			if err != nil {
				return translateStoreErr(err)
			}
			return emit(cmd, n)
		},
	}
	cmd.Flags().StringVar(&by, "by", "", "Slug of the node that resolves the curiosity")
	return cmd
}

func newNodeTagCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "tag <node-slug> <tag> [<tag>...]",
		Short: "Add tag(s) to a node",
		Args:  cobra.MinimumNArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			s, err := openStore(cmd)
			if err != nil {
				return Exit(ExitGeneric, "open store: %v", err)
			}
			defer s.Close()
			n, err := s.AddTags(args[0], args[1:])
			if err != nil {
				return translateStoreErr(err)
			}
			return emit(cmd, n)
		},
	}
}

func newNodeUntagCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "untag <node-slug> <tag>",
		Short: "Remove a tag from a node",
		Args:  cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			s, err := openStore(cmd)
			if err != nil {
				return Exit(ExitGeneric, "open store: %v", err)
			}
			defer s.Close()
			n, err := s.RemoveTag(args[0], args[1])
			if err != nil {
				return translateStoreErr(err)
			}
			return emit(cmd, n)
		},
	}
}

func newNodeDeleteCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "delete <node-slug>",
		Short: "Delete a node (cascades to its edges)",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			s, err := openStore(cmd)
			if err != nil {
				return Exit(ExitGeneric, "open store: %v", err)
			}
			defer s.Close()
			if err := s.DeleteNode(args[0]); err != nil {
				return translateStoreErr(err)
			}
			return emit(cmd, map[string]string{"deleted": args[0]})
		},
	}
}
