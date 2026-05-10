// Package cli wires the cobra command tree for speechflow. The CLI is
// the only writer for the data store; the HTTP API (in internal/server)
// is read-only.
package cli

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"

	"github.com/camggould/speechflow/internal/store"
)

// Exit codes per the README.
const (
	ExitSuccess    = 0
	ExitGeneric    = 1
	ExitUsage      = 2
	ExitNotFound   = 3
	ExitConstraint = 4
)

// ExitError carries a deferred exit code from a command's RunE up to main.
type ExitError struct {
	Code    int
	Message string
}

// Error implements the error interface so cobra can propagate the value.
func (e *ExitError) Error() string { return e.Message }

// Exit returns a fresh ExitError for the given code and message.
func Exit(code int, format string, args ...any) *ExitError {
	return &ExitError{Code: code, Message: fmt.Sprintf(format, args...)}
}

// NewRootCommand constructs the top-level speechflow command tree.
// Each call returns a fresh tree so tests can drive the CLI without
// stale flag state.
func NewRootCommand() *cobra.Command {
	root := &cobra.Command{
		Use:           "speechflow",
		Short:         "Structure spoken or written conversations as concept graphs",
		Long:          "speechflow lets an LLM agent record a session as a concept graph: roots intended, iterations rehearsed, nodes introduced, edges connecting them, and coverage measured.",
		SilenceErrors: true,
		SilenceUsage:  true,
	}

	root.PersistentFlags().Bool("pretty", false, "Render output human-readably instead of JSON")
	root.PersistentFlags().String("db", "", "Path to the SQLite database (default: ~/.speechflow/speechflow.db)")

	root.AddCommand(newInitCommand())
	root.AddCommand(newVersionCommand())
	root.AddCommand(newSessionCommand())
	root.AddCommand(newRootGroupCommand())
	root.AddCommand(newIterationCommand())
	root.AddCommand(newTranscriptCommand())
	root.AddCommand(newNodeCommand())
	root.AddCommand(newEdgeCommand())
	root.AddCommand(newCoverageCommand())
	root.AddCommand(newTimelineCommand())
	root.AddCommand(newExportCommand())
	root.AddCommand(newServeCommand())

	return root
}

// dataDir returns the user's speechflow data directory (typically
// ~/.speechflow). Honors $SPEECHFLOW_HOME for tests.
func dataDir() (string, error) {
	if h := os.Getenv("SPEECHFLOW_HOME"); h != "" {
		return h, nil
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("resolve home dir: %w", err)
	}
	return filepath.Join(home, ".speechflow"), nil
}

// dbPath returns the absolute path to the SQLite database. The --db flag
// overrides; otherwise we use <dataDir>/speechflow.db.
func dbPath(cmd *cobra.Command) (string, error) {
	override, _ := cmd.Root().PersistentFlags().GetString("db")
	if override != "" {
		return override, nil
	}
	dir, err := dataDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, "speechflow.db"), nil
}

// openStore opens the speechflow database. The data dir is created if missing.
func openStore(cmd *cobra.Command) (*store.Store, error) {
	dir, err := dataDir()
	if err != nil {
		return nil, err
	}
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return nil, fmt.Errorf("mkdir %s: %w", dir, err)
	}
	p, err := dbPath(cmd)
	if err != nil {
		return nil, err
	}
	if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
		return nil, fmt.Errorf("mkdir %s: %w", filepath.Dir(p), err)
	}
	return store.Open(p)
}

// pretty reports whether --pretty was passed on this invocation.
func pretty(cmd *cobra.Command) bool {
	b, _ := cmd.Root().PersistentFlags().GetBool("pretty")
	return b
}
