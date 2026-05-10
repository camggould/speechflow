// `speechflow coverage`: per-iteration or per-session structural coverage.
package cli

import (
	"github.com/spf13/cobra"

	"github.com/camggould/speechflow/internal/coverage"
)

func newCoverageCommand() *cobra.Command {
	var sessionMode bool
	cmd := &cobra.Command{
		Use:   "coverage",
		Short: "Report root coverage for the active iteration or whole session",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			s, err := openStore(cmd)
			if err != nil {
				return Exit(ExitGeneric, "open store: %v", err)
			}
			defer s.Close()
			if sessionMode {
				sess, err := activeSession()
				if err != nil {
					return err
				}
				m, err := coverage.Matrix(s, sess)
				if err != nil {
					return translateStoreErr(err)
				}
				return emit(cmd, m)
			}
			iter, err := activeIteration()
			if err != nil {
				return err
			}
			rows, err := coverage.Compute(s, iter)
			if err != nil {
				return translateStoreErr(err)
			}
			return emit(cmd, rows)
		},
	}
	cmd.Flags().BoolVar(&sessionMode, "session", false, "Return the full session coverage matrix")
	return cmd
}
