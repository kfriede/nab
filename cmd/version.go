package cmd

import (
	"encoding/json"
	"fmt"
	"runtime"

	"github.com/spf13/cobra"
)

// Set via ldflags at build time
var (
	Version   = "dev"
	Commit    = "unknown"
	BuildDate = "unknown"
)

var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Print nab version information",
	Long:  "Print the version, commit hash, and build date of nab.",
	Args:  cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		if flagJSON || flagOutput == "json" {
			info := map[string]string{
				"version":   Version,
				"commit":    Commit,
				"buildDate": BuildDate,
				"go":        runtime.Version(),
				"os":        runtime.GOOS,
				"arch":      runtime.GOARCH,
			}
			enc := json.NewEncoder(cmd.OutOrStdout())
			enc.SetIndent("", "  ")
			return enc.Encode(info)
		}

		_, _ = fmt.Fprintf(cmd.OutOrStdout(), "nab %s (%s) built %s\n", Version, Commit, BuildDate)
		return nil
	},
}
