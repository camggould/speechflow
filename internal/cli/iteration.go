// `speechflow iteration ...`: rehearsals/passes of a session.
package cli

import (
	"github.com/spf13/cobra"

	"github.com/camggould/speechflow/internal/slug"
	"github.com/camggould/speechflow/internal/state"
)

func newIterationCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "iteration",
		Short: "Manage iterations on the active session",
	}
	cmd.AddCommand(newIterationStartCommand())
	cmd.AddCommand(newIterationEndCommand())
	cmd.AddCommand(newIterationListCommand())
	cmd.AddCommand(newIterationUseCommand())
	cmd.AddCommand(newIterationDeleteCommand())
	return cmd
}

func newIterationStartCommand() *cobra.Command {
	var title string
	cmd := &cobra.Command{
		Use:   "start",
		Short: "Start a new iteration on the active session",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			sess, err := activeSession()
			if err != nil {
				return err
			}
			s, err := openStore(cmd)
			if err != nil {
				return Exit(ExitGeneric, "open store: %v", err)
			}
			defer s.Close()
			if title == "" {
				// Default title is "iteration N+1".
				existing, _ := s.ListIterations(sess)
				title = generateIterationTitle(len(existing) + 1)
			}
			// Iterations use random IDs rather than slugs. Titles are not
			// constrained to be unique within or across sessions ("Rehearsal 1"
			// happens many times), and the database's PRIMARY KEY on
			// iterations.id is global, so slug suffixing would still collide
			// across sessions. Random IDs side-step the whole problem.
			id := slug.Random("it_")
			it, err := s.CreateIteration(id, sess, title)
			if err != nil {
				return translateStoreErr(err)
			}
			_ = decorateIteration(s, it)
			dir, _ := dataDir()
			_ = state.SetActiveIteration(dir, it.ID)
			return emit(cmd, it)
		},
	}
	cmd.Flags().StringVar(&title, "title", "", "Iteration title (default: 'iteration N')")
	return cmd
}

func generateIterationTitle(n int) string {
	if n < 1 {
		n = 1
	}
	switch n {
	case 1:
		return "iteration 1"
	default:
		return "iteration " + itoa(n)
	}
}

func itoa(n int) string {
	if n == 0 {
		return "0"
	}
	digits := []byte{}
	neg := n < 0
	if neg {
		n = -n
	}
	for n > 0 {
		digits = append([]byte{byte('0' + n%10)}, digits...)
		n /= 10
	}
	if neg {
		return "-" + string(digits)
	}
	return string(digits)
}

func newIterationEndCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "end [iteration-slug]",
		Short: "End an iteration (default: active)",
		Args:  cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			s, err := openStore(cmd)
			if err != nil {
				return Exit(ExitGeneric, "open store: %v", err)
			}
			defer s.Close()
			id, err := resolveIterationArg(args)
			if err != nil {
				return err
			}
			it, err := s.EndIteration(id)
			if err != nil {
				return translateStoreErr(err)
			}
			_ = decorateIteration(s, it)
			return emit(cmd, it)
		},
	}
}

func newIterationListCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "list [session-slug]",
		Short: "List iterations for a session (default: active)",
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
			list, err := s.ListIterations(id)
			if err != nil {
				return translateStoreErr(err)
			}
			for i := range list {
				_ = decorateIteration(s, &list[i])
			}
			return emit(cmd, list)
		},
	}
}

func newIterationUseCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "use <iteration-slug>",
		Short: "Set the active iteration",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			s, err := openStore(cmd)
			if err != nil {
				return Exit(ExitGeneric, "open store: %v", err)
			}
			defer s.Close()
			it, err := s.GetIteration(args[0])
			if err != nil {
				return translateStoreErr(err)
			}
			dir, _ := dataDir()
			// Also switch the active session so iteration is consistent.
			if err := state.SetActiveSession(dir, it.SessionID); err != nil {
				return Exit(ExitGeneric, "%v", err)
			}
			if err := state.SetActiveIteration(dir, it.ID); err != nil {
				return Exit(ExitGeneric, "%v", err)
			}
			_ = decorateIteration(s, it)
			return emit(cmd, it)
		},
	}
}

func newIterationDeleteCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "delete <iteration-slug>",
		Short: "Delete an iteration (cascades to nodes/edges/transcript)",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			s, err := openStore(cmd)
			if err != nil {
				return Exit(ExitGeneric, "open store: %v", err)
			}
			defer s.Close()
			if err := s.DeleteIteration(args[0]); err != nil {
				return translateStoreErr(err)
			}
			dir, _ := dataDir()
			st, _ := state.Load(dir)
			if st.ActiveIteration == args[0] {
				_ = state.SetActiveIteration(dir, "")
			}
			return emit(cmd, map[string]string{"deleted": args[0]})
		},
	}
}

// resolveIterationArg returns args[0] or the active iteration.
func resolveIterationArg(args []string) (string, error) {
	if len(args) == 1 {
		return args[0], nil
	}
	return activeIteration()
}
