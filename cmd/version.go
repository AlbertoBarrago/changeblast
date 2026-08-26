package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
)

var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Print the blast version",
	Args:  cobra.NoArgs,
	RunE: func(c *cobra.Command, args []string) error {
		fmt.Fprintf(c.OutOrStdout(), "blast %s\n", version)
		return nil
	},
}

func init() {
	rootCmd.AddCommand(versionCmd)
}
