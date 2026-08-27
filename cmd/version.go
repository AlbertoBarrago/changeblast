package cmd

import (
	"fmt"
	"runtime"

	"github.com/spf13/cobra"
)

// commit and date are set at build time via -ldflags (see
// .goreleaser.yml). They default to "unknown" for local/dev builds that
// don't pass them explicitly.
var (
	commit = "unknown"
	date   = "unknown"
)

var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Print the impactline version",
	Args:  cobra.NoArgs,
	RunE: func(c *cobra.Command, args []string) error {
		out := c.OutOrStdout()
		fmt.Fprintf(out, "impactline %s\n", version)
		fmt.Fprintf(out, "  commit:  %s\n", commit)
		fmt.Fprintf(out, "  built:   %s\n", date)
		fmt.Fprintf(out, "  go:      %s\n", runtime.Version())
		fmt.Fprintf(out, "  platform: %s/%s\n", runtime.GOOS, runtime.GOARCH)
		return nil
	},
}

func init() {
	rootCmd.AddCommand(versionCmd)
}
