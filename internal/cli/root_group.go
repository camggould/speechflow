// `speechflow root ...`: session-scoped roots.
package cli

import (
	"github.com/spf13/cobra"

	"github.com/camggould/speechflow/internal/core"
	"github.com/camggould/speechflow/internal/slug"
)

func newRootGroupCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "root",
		Short: "Manage roots (topics) on the active session",
	}
	cmd.AddCommand(newRootAddCommand())
	cmd.AddCommand(newRootListCommand())
	cmd.AddCommand(newRootDeleteCommand())
	return cmd
}

func newRootAddCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "add <title> [<title>...]",
		Short: "Add one or more roots to the active session",
		Args:  cobra.MinimumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			sess, err := activeSession()
			if err != nil {
				return err
			}
			s, err := openStore(cmd)
			if err != nil {
				return Exit(ExitGeneric, "open store: %v", err)
			}
			defer s.Close()
			created := make([]core.Root, 0, len(args))
			for _, title := range args {
				id, err := slug.Unique(title, func(c string) (bool, error) {
					return s.SlugExists("roots", sess, c)
				})
				if err != nil {
					return Exit(ExitGeneric, "%v", err)
				}
				r, err := s.CreateRoot(id, sess, title)
				if err != nil {
					return translateStoreErr(err)
				}
				created = append(created, *r)
			}
			if len(created) == 1 {
				return emit(cmd, &created[0])
			}
			return emit(cmd, created)
		},
	}
}

func newRootListCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "list [session-slug]",
		Short: "List roots (default: active session)",
		Args:  cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			s, err := openStore(cmd)
			if err != nil {
				return Exit(ExitGeneric, "open store: %v", err)
			}
			defer s.Close()
			id, err := resolveSessionArg(args)
			if err != nil {
				return err
			}
			list, err := s.ListRoots(id)
			if err != nil {
				return translateStoreErr(err)
			}
			return emit(cmd, list)
		},
	}
}

func newRootDeleteCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "delete <root-slug>",
		Short: "Delete a root by slug",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			s, err := openStore(cmd)
			if err != nil {
				return Exit(ExitGeneric, "open store: %v", err)
			}
			defer s.Close()
			if err := s.DeleteRoot(args[0]); err != nil {
				return translateStoreErr(err)
			}
			return emit(cmd, map[string]string{"deleted": args[0]})
		},
	}
}
