package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

// version is set at build time via -ldflags (see Makefile/.goreleaser.yml).
// It defaults to "dev" for local builds.
var version = "dev"

// registeredSubcommands lists subcommand names that take precedence over
// the `blast <path>` convenience alias, per the documented resolution
// order: exact subcommand match wins before path resolution is attempted.
var registeredSubcommands = map[string]bool{
	"inspect":    true,
	"diff":       true,
	"graph":      true,
	"doctor":     true,
	"history":    true,
	"version":    true,
	"completion": true,
	"help":       true,
}

// rootCmd is the base command. Running `blast <path>` with no matching
// subcommand name is a convenience alias for `blast inspect <path>`,
// resolved in Args below.
var rootCmd = &cobra.Command{
	Use:   "blast",
	Short: "Estimate the blast radius of a code change",
	Long: `ChangeBlast (blast) analyzes a Git repository and estimates the impact of
changing a given file or directory, using deterministic evidence such as
dependency graphs, Git history, and CI configuration.

Canonical usage:
  blast inspect <path>

Convenience alias (root command):
  blast <path>

Resolution order for the alias:
  1. If the first argument matches a registered subcommand name exactly
     (diff, graph, doctor, history, version, completion, inspect), it is
     treated as that subcommand.
  2. Otherwise, if the first argument resolves to an existing path in the
     working tree, it is treated as "blast inspect <path>".
  3. Otherwise, blast errors with "unknown command or path" rather than
     silently guessing.

CI pipelines and scripts should prefer the canonical "blast inspect" form
to avoid any ambiguity.`,
	Args:               cobra.ArbitraryArgs,
	DisableFlagParsing: false,
	SilenceUsage:       true,
	SilenceErrors:      true,
	RunE: func(c *cobra.Command, args []string) error {
		if len(args) == 0 {
			return c.Help()
		}

		first := args[0]
		if registeredSubcommands[first] {
			// Cobra's normal dispatch would have already routed this to the
			// matching subcommand; reaching here means it didn't match
			// (e.g. typo), so fall through to the path check below.
			if _, err := os.Stat(first); err != nil {
				return fmt.Errorf("unknown command or path: %q", first)
			}
		}

		if _, err := os.Stat(first); err == nil {
			return runInspect(c, args)
		}

		return fmt.Errorf("unknown command or path: %q", first)
	},
}

// Execute runs the root command.
func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, "Error:", err)
		os.Exit(1)
	}
}
