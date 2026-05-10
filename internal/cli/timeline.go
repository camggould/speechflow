// `speechflow timeline`: ordered events for the active iteration.
package cli

import "github.com/spf13/cobra"

func newTimelineCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "timeline",
		Short: "Show the ordered event timeline for the active iteration",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			iter, err := activeIteration()
			if err != nil {
				return err
			}
			s, err := openStore(cmd)
			if err != nil {
				return Exit(ExitGeneric, "open store: %v", err)
			}
			defer s.Close()
			events, err := s.Timeline(iter)
			if err != nil {
				return translateStoreErr(err)
			}
			return emit(cmd, events)
		},
	}
}
