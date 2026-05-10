// `speechflow transcript ...`: append/set/show the active iteration's transcript.
package cli

import (
	"io"
	"os"
	"strings"

	"github.com/spf13/cobra"
)

func newTranscriptCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "transcript",
		Short: "Manage the active iteration transcript",
	}
	cmd.AddCommand(newTranscriptAppendCommand())
	cmd.AddCommand(newTranscriptSetCommand())
	cmd.AddCommand(newTranscriptShowCommand())
	return cmd
}

func newTranscriptAppendCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "append <text>",
		Short: "Append text to the active iteration transcript",
		Args:  cobra.MinimumNArgs(1),
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
			text := strings.Join(args, " ")
			it, err := s.AppendTranscript(iter, text)
			if err != nil {
				return translateStoreErr(err)
			}
			return emit(cmd, it)
		},
	}
}

func newTranscriptSetCommand() *cobra.Command {
	var file string
	cmd := &cobra.Command{
		Use:   "set",
		Short: "Replace the transcript from --file or stdin",
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
			var data []byte
			if file != "" {
				b, err := os.ReadFile(file)
				if err != nil {
					return Exit(ExitGeneric, "read %s: %v", file, err)
				}
				data = b
			} else {
				b, err := io.ReadAll(cmd.InOrStdin())
				if err != nil {
					return Exit(ExitGeneric, "read stdin: %v", err)
				}
				data = b
			}
			it, err := s.SetTranscript(iter, strings.TrimRight(string(data), "\n"))
			if err != nil {
				return translateStoreErr(err)
			}
			return emit(cmd, it)
		},
	}
	cmd.Flags().StringVar(&file, "file", "", "Read transcript from file (otherwise stdin)")
	return cmd
}

func newTranscriptShowCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "show",
		Short: "Show the active iteration transcript",
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
			tr, err := s.GetTranscript(iter)
			if err != nil {
				return translateStoreErr(err)
			}
			if pretty(cmd) {
				cmd.Println(tr.Text)
				return nil
			}
			return emit(cmd, tr)
		},
	}
}
