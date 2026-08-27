package cmd

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"
)

// version is set at build time via -ldflags (see Makefile/.goreleaser.yml).
// It defaults to "dev" for local builds.
var version = "dev"

// rootCmd is the base command. Running `serval <path>` with no
// matching subcommand name is a convenience alias for `serval
// inspect <path>`, resolved in RunE below. By the time RunE runs,
// Cobra has already dispatched any argument matching a registered
// subcommand name to that subcommand directly — RunE only ever sees an
// argument that did not match one.
var rootCmd = &cobra.Command{
	Use:   "serval",
	Short: "Estimate the blast radius of a code change",
	Long: `Serval analyzes a Git repository and estimates the impact of changing
a given file or directory, using deterministic evidence such as
dependency graphs, Git history, and CI configuration.

Canonical usage:
  serval inspect <path>

Convenience alias (root command):
  serval <path>

Resolution order for the alias:
  1. If the first argument matches a registered subcommand name exactly
     (diff, graph, doctor, history, version, completion, inspect), it is
     dispatched to that subcommand.
  2. Otherwise, if the first argument resolves to an existing path in the
     working tree, it is treated as "serval inspect <path>".
  3. Otherwise, serval errors with "unknown command or path" rather
     than silently guessing.

CI pipelines and scripts should prefer the canonical "serval
inspect" form to avoid any ambiguity.`,
	Args:          cobra.ArbitraryArgs,
	SilenceUsage:  true,
	SilenceErrors: true,
	RunE: func(c *cobra.Command, args []string) error {
		if len(args) == 0 {
			return c.Help()
		}

		first := args[0]
		if _, err := os.Stat(first); err != nil {
			return fmt.Errorf("unknown command or path: %q", first)
		}

		if err := validateFailOn(inspectFlags.failOn); err != nil {
			return err
		}

		return runInspect(c, args)
	},
}

func init() {
	// The `serval <path>` alias forwards to inspect, so it accepts
	// the same flags as `serval inspect <path>`.
	addAnalysisFlags(rootCmd, &inspectFlags)
}

// targetArg returns args[0], or "." (the current directory) when no
// argument was given, so commands that support a directory target don't
// force the user to type "serval <cmd> ." explicitly.
func targetArg(args []string) string {
	if len(args) == 0 {
		return "."
	}
	return args[0]
}

// resolveTarget resolves a user-supplied path argument to an absolute
// path and its enclosing repository root, verifying the path exists.
// Shared by every command that takes a single file/directory target.
func resolveTarget(arg string) (target, root string, err error) {
	target, err = filepath.Abs(arg)
	if err != nil {
		return "", "", err
	}
	if _, err := os.Stat(target); err != nil {
		return "", "", fmt.Errorf("target not found or unreadable: %q", arg)
	}

	root, err = repositoryRoot(target)
	if err != nil {
		return "", "", err
	}
	return target, root, nil
}

// repositoryRoot walks upward from path looking for a .git entry. It
// accepts both a .git directory (normal checkout) and a .git file
// (worktrees, submodules). Outside any repository it returns an error so
// callers can fail explicitly instead of silently degrading to git-less
// results.
func repositoryRoot(path string) (string, error) {
	dir := path
	if info, err := os.Stat(path); err == nil && !info.IsDir() {
		dir = filepath.Dir(path)
	}

	for {
		if info, err := os.Stat(filepath.Join(dir, ".git")); err == nil && (info.IsDir() || info.Mode().IsRegular()) {
			return dir, nil
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			return "", fmt.Errorf("no git repository found above %s", path)
		}
		dir = parent
	}
}

// RootCmd exposes the root command for tooling (e.g. man page
// generation via cobra/doc) that needs to walk the full command tree.
func RootCmd() *cobra.Command {
	return rootCmd
}

// Execute runs the root command and maps errors to the documented exit
// code contract: 0 success, 1 execution error, 2 risk threshold exceeded
// (only when a failOnError is returned, i.e. --fail-on tripped).
func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, "Error:", err)
		if _, ok := err.(failOnError); ok {
			os.Exit(2)
		}
		os.Exit(1)
	}
}
