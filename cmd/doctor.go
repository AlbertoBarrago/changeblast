package cmd

import (
	"fmt"
	"os"
	"os/exec"

	"github.com/spf13/cobra"

	githubci "github.com/AlbertoBarrago/changeblast/internal/ci/github"
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
		isRepo := false
		if _, statErr := os.Stat(root + "/.git"); statErr == nil {
			isRepo = true
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

		if workflows, err := githubci.New().Discover(root); err == nil && len(workflows) > 0 {
			fmt.Fprintf(out, "✓ GitHub Actions   %d workflow(s)\n", len(workflows))
		} else {
			fmt.Fprintln(out, "- GitHub Actions   no workflows found (optional)")
		}

		if isRepo {
			if _, err := exec.Command("git", "-C", root, "log", "-1").Output(); err == nil {
				fmt.Fprintln(out, "✓ git history      available")
			} else {
				fmt.Fprintln(out, "- git history      no commits yet (optional)")
			}
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
