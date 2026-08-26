package cmd

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"

	"github.com/spf13/cobra"

	githubci "github.com/AlbertoBarrago/changeblast/internal/ci/github"
	"github.com/AlbertoBarrago/changeblast/internal/output"
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
	ok := func() string { return output.StatusOK.Colorize(out) }
	fail := func() string { return output.StatusFail.Colorize(out) }
	info := func() string { return output.StatusInfo.Colorize(out) }

	fmt.Fprintln(out, "ChangeBlast environment")
	fmt.Fprintln(out)

	allOK := true

	if v, err := exec.Command("git", "--version").Output(); err == nil {
		fmt.Fprintf(out, "%s git              %s", ok(), trimPrefix(string(v), "git version "))
	} else {
		fmt.Fprintf(out, "%s git              not found in PATH\n", fail())
		allOK = false
	}

	cwd, err := os.Getwd()
	if err != nil {
		return err
	}

	if root, err := repositoryRoot(cwd); err == nil {
		isRepo := false
		if _, statErr := os.Stat(filepath.Join(root, ".git")); statErr == nil {
			isRepo = true
			fmt.Fprintf(out, "%s repository       detected\n", ok())
		} else {
			fmt.Fprintf(out, "%s repository       not a git repository\n", fail())
			allOK = false
		}

		if cfg, err := repository.FindTSConfig(root); err == nil && cfg != nil {
			fmt.Fprintf(out, "%s tsconfig.json    detected\n", ok())
		} else {
			fmt.Fprintf(out, "%s tsconfig.json    not found (optional)\n", info())
		}

		if workflows, err := githubci.New().Discover(root); err == nil && len(workflows) > 0 {
			fmt.Fprintf(out, "%s GitHub Actions   %d %s\n", ok(), len(workflows), pluralize(len(workflows), "workflow", "workflows"))
		} else {
			fmt.Fprintf(out, "%s GitHub Actions   no workflows found (optional)\n", info())
		}

		if isRepo {
			if _, err := exec.Command("git", "-C", root, "log", "-1").Output(); err == nil {
				fmt.Fprintf(out, "%s git history      available\n", ok())
			} else {
				fmt.Fprintf(out, "%s git history      no commits yet (optional)\n", info())
			}
		}
	}

	fmt.Fprintln(out)
	if allOK {
		fmt.Fprintln(out, "Ready.")
		return nil
	}
	return fmt.Errorf("environment checks failed")
}

// pluralize returns singular when n == 1, plural otherwise.
func pluralize(n int, singular, plural string) string {
	if n == 1 {
		return singular
	}
	return plural
}

func trimPrefix(s, prefix string) string {
	if len(s) >= len(prefix) && s[:len(prefix)] == prefix {
		return s[len(prefix):]
	}
	return s
}
