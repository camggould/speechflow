// `speechflow init`: bootstrap the data dir and run migrations.
package cli

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"
)

func newInitCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "init",
		Short: "Create the speechflow data directory and run migrations",
		Long:  "Ensure ~/.speechflow/ exists and the SQLite database is migrated to the latest schema. Idempotent.",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			dir, err := dataDir()
			if err != nil {
				return Exit(ExitGeneric, "%v", err)
			}
			if err := os.MkdirAll(dir, 0o755); err != nil {
				return Exit(ExitGeneric, "mkdir %s: %v", dir, err)
			}
			p, err := dbPath(cmd)
			if err != nil {
				return Exit(ExitGeneric, "%v", err)
			}
			if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
				return Exit(ExitGeneric, "mkdir %s: %v", filepath.Dir(p), err)
			}
			s, err := openStore(cmd)
			if err != nil {
				return Exit(ExitGeneric, "open db: %v", err)
			}
			defer s.Close()
			fmt.Fprintf(cmd.OutOrStdout(), "initialized speechflow at %s\n", dir)
			fmt.Fprintf(cmd.OutOrStdout(), "  db: %s\n", p)
			return nil
		},
	}
}
