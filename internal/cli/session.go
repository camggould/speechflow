// `speechflow session ...`: CRUD for sessions, plus session activation.
package cli

import (
	"github.com/spf13/cobra"

	"github.com/camggould/speechflow/internal/slug"
	"github.com/camggould/speechflow/internal/state"
)

func newSessionCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "session",
		Short: "Manage sessions",
	}
	cmd.AddCommand(newSessionNewCommand())
	cmd.AddCommand(newSessionListCommand())
	cmd.AddCommand(newSessionShowCommand())
	cmd.AddCommand(newSessionUseCommand())
	cmd.AddCommand(newSessionDeleteCommand())
	return cmd
}

func newSessionNewCommand() *cobra.Command {
	var title, description string
	cmd := &cobra.Command{
		Use:   "new",
		Short: "Create a new session",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			if title == "" {
				return Exit(ExitUsage, "--title is required")
			}
			s, err := openStore(cmd)
			if err != nil {
				return Exit(ExitGeneric, "open store: %v", err)
			}
			defer s.Close()

			id, err := slug.Unique(title, func(c string) (bool, error) {
				return s.SlugExists("sessions", "", c)
			})
			if err != nil {
				return Exit(ExitGeneric, "%v", err)
			}
			var descPtr *string
			if description != "" {
				descPtr = &description
			}
			sess, err := s.CreateSession(id, title, descPtr)
			if err != nil {
				return translateStoreErr(err)
			}
			// Switch active session to the newly created one.
			dir, _ := dataDir()
			_ = state.SetActiveSession(dir, sess.ID)
			return emit(cmd, sess)
		},
	}
	cmd.Flags().StringVar(&title, "title", "", "Session title (required)")
	cmd.Flags().StringVar(&description, "description", "", "Optional description")
	return cmd
}

func newSessionListCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "list",
		Short: "List all sessions",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			s, err := openStore(cmd)
			if err != nil {
				return Exit(ExitGeneric, "open store: %v", err)
			}
			defer s.Close()
			list, err := s.ListSessions()
			if err != nil {
				return translateStoreErr(err)
			}
			for i := range list {
				_ = decorateSession(s, &list[i])
			}
			return emit(cmd, list)
		},
	}
}

func newSessionShowCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "show [session-slug]",
		Short: "Show a session (default: active session)",
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
			sess, err := s.GetSession(id)
			if err != nil {
				return translateStoreErr(err)
			}
			_ = decorateSession(s, sess)
			return emit(cmd, sess)
		},
	}
}

func newSessionUseCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "use <session-slug>",
		Short: "Set the active session",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			s, err := openStore(cmd)
			if err != nil {
				return Exit(ExitGeneric, "open store: %v", err)
			}
			defer s.Close()
			if _, err := s.GetSession(args[0]); err != nil {
				return translateStoreErr(err)
			}
			dir, _ := dataDir()
			if err := state.SetActiveSession(dir, args[0]); err != nil {
				return Exit(ExitGeneric, "%v", err)
			}
			sess, _ := s.GetSession(args[0])
			_ = decorateSession(s, sess)
			return emit(cmd, sess)
		},
	}
}

func newSessionDeleteCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "delete <session-slug>",
		Short: "Delete a session (cascades to iterations)",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			s, err := openStore(cmd)
			if err != nil {
				return Exit(ExitGeneric, "open store: %v", err)
			}
			defer s.Close()
			if err := s.DeleteSession(args[0]); err != nil {
				return translateStoreErr(err)
			}
			// Clear active session if it pointed here.
			dir, _ := dataDir()
			st, _ := state.Load(dir)
			if st.ActiveSession == args[0] {
				_ = state.SetActiveSession(dir, "")
			}
			return emit(cmd, map[string]string{"deleted": args[0]})
		},
	}
}

// resolveSessionArg returns args[0] if provided, otherwise the active session.
func resolveSessionArg(args []string) (string, error) {
	if len(args) == 1 {
		return args[0], nil
	}
	return activeSession()
}
