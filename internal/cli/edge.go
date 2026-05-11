// `speechflow edge ...`: create/delete graph edges.
package cli

import (
	"github.com/spf13/cobra"

	"github.com/camggould/speechflow/internal/core"
)

func newEdgeCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "edge",
		Short: "Manage edges on the active iteration",
	}
	cmd.AddCommand(newEdgeAddCommand())
	cmd.AddCommand(newEdgeDeleteCommand())
	return cmd
}

func newEdgeAddCommand() *cobra.Command {
	var kind string
	cmd := &cobra.Command{
		Use:   "add <from-slug> <to-slug>",
		Short: "Add a directed edge between two nodes",
		Args:  cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			k := core.EdgeKind(kind)
			switch k {
			case core.EdgeBranchesFrom, core.EdgeReferences, core.EdgeReturnsTo,
				core.EdgeSupports, core.EdgeContrasts:
			default:
				return Exit(ExitUsage, "--kind must be one of branches_from|references|returns_to|supports|contrasts")
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
			id, err := s.NextEdgeID(iter)
			if err != nil {
				return translateStoreErr(err)
			}
			e, err := s.CreateEdge(id, iter, args[0], args[1], k)
			if err != nil {
				return translateStoreErr(err)
			}
			return emit(cmd, e)
		},
	}
	cmd.Flags().StringVar(&kind, "kind", "", "Edge kind: branches_from|references|returns_to|supports|contrasts")
	return cmd
}

func newEdgeDeleteCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "delete <edge-id>",
		Short: "Delete an edge by id",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			s, err := openStore(cmd)
			if err != nil {
				return Exit(ExitGeneric, "open store: %v", err)
			}
			defer s.Close()
			if err := s.DeleteEdge(args[0]); err != nil {
				return translateStoreErr(err)
			}
			return emit(cmd, map[string]string{"deleted": args[0]})
		},
	}
}
