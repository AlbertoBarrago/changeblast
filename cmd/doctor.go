package cmd

import (
	"fmt"
	"os"
	"os/exec"

	"github.com/spf13/cobra"

	"github.com/AlbertoBarrago/changeblast/internal/repository"
)

var doctorCmd = &cobra.Command{
	Use:   "doctor",
	Short: "Check the environment and repository for ChangeBlast compatibility",
	Args:  cobra.NoArgs,
	RunE:  runDoctor,
}

func init() {
	rootCmd.AddCommand(doctorCmd)
}

func runDoctor(c *cobra.Command, args []string) error {
	out := c.OutOrStdout()
	fmt.Fprintln(out, "ChangeBlast environment")
	fmt.Fprintln(out)

	ok := true

	if v, err := exec.Command("git", "--version").Output(); err == nil {
		fmt.Fprintf(out, "✓ git              %s", trimPrefix(string(v), "git version "))
	} else {
		fmt.Fprintln(out, "✗ git              not found in PATH")
		ok = false
	}

	cwd, err := os.Getwd()
	if err != nil {
		return err
	}

	if root, err := repositoryRoot(cwd); err == nil {
		if _, statErr := os.Stat(root + "/.git"); statErr == nil {
			fmt.Fprintln(out, "✓ repository       detected")
		} else {
			fmt.Fprintln(out, "✗ repository       not a git repository")
			ok = false
		}

		if cfg, err := repository.FindTSConfig(root); err == nil && cfg != nil {
			fmt.Fprintln(out, "✓ tsconfig.json    detected")
		} else {
			fmt.Fprintln(out, "- tsconfig.json    not found (optional)")
		}
	}

	fmt.Fprintln(out)
	if ok {
		fmt.Fprintln(out, "Ready.")
		return nil
	}
	return fmt.Errorf("environment checks failed")
}

func trimPrefix(s, prefix string) string {
	if len(s) >= len(prefix) && s[:len(prefix)] == prefix {
		return s[len(prefix):]
	}
	return s
}
