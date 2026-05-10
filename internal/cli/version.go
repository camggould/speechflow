// `speechflow version`: print build metadata.
package cli

import (
	"encoding/json"
	"fmt"
	"runtime/debug"

	"github.com/spf13/cobra"
)

// Populated at build time via -ldflags. Defaults make the binary runnable
// from `go build` without ldflags during development.
var (
	buildVersion = "dev"
	buildCommit  = ""
	buildDate    = ""
)

type versionInfo struct {
	Version string `json:"version"`
	Commit  string `json:"commit"`
	Built   string `json:"built"`
}

func newVersionCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "version",
		Short: "Print version information",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			version, commit, date := resolveVersion()
			info := versionInfo{Version: version, Commit: commit, Built: date}
			if pretty(cmd) {
				w := cmd.OutOrStdout()
				fmt.Fprintf(w, "speechflow %s\n", info.Version)
				if info.Commit != "" {
					fmt.Fprintf(w, "  commit: %s\n", info.Commit)
				}
				if info.Built != "" {
					fmt.Fprintf(w, "  built:  %s\n", info.Built)
				}
				return nil
			}
			enc := json.NewEncoder(cmd.OutOrStdout())
			enc.SetIndent("", "  ")
			return enc.Encode(info)
		},
	}
}

// resolveVersion prefers ldflag-injected values, falling back to
// debug.ReadBuildInfo for `go install` builds.
func resolveVersion() (string, string, string) {
	version, commit, date := buildVersion, buildCommit, buildDate
	info, ok := debug.ReadBuildInfo()
	if !ok {
		return version, commit, date
	}
	if version == "dev" && info.Main.Version != "" && info.Main.Version != "(devel)" {
		version = info.Main.Version
	}
	for _, s := range info.Settings {
		switch s.Key {
		case "vcs.revision":
			if commit == "" {
				commit = s.Value
			}
		case "vcs.time":
			if date == "" {
				date = s.Value
			}
		}
	}
	return version, commit, date
}
