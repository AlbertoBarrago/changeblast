package cmd

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"time"

	"github.com/spf13/cobra"

	"github.com/AlbertoBarrago/serval/internal/ai/ollama"
	githubci "github.com/AlbertoBarrago/serval/internal/ci/github"
	"github.com/AlbertoBarrago/serval/internal/output"
	"github.com/AlbertoBarrago/serval/internal/repository"
)

var doctorCmd = &cobra.Command{
	Use:   "doctor",
	Short: "Check the environment and repository for Serval compatibility",
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

	fmt.Fprintln(out, "Serval environment")
	fmt.Fprintln(out)

	allOK := true

	if v, err := exec.Command("git", "--version").Output(); err == nil {
		fmt.Fprintf(out, "%s git              %s", ok(), strings.TrimPrefix(string(v), "git version "))
	} else {
		fmt.Fprintf(out, "%s git              not found in PATH\n", fail())
		allOK = false
	}

	cwd, err := os.Getwd()
	if err != nil {
		return err
	}

	root, err := repositoryRoot(cwd)
	if err != nil {
		fmt.Fprintf(out, "%s repository       not a git repository\n", fail())
		allOK = false
	} else {
		fmt.Fprintf(out, "%s repository       detected\n", ok())

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

		if _, err := exec.Command("git", "-C", root, "log", "-1").Output(); err == nil {
			fmt.Fprintf(out, "%s git history      available\n", ok())
		} else {
			fmt.Fprintf(out, "%s git history      no commits yet (optional)\n", info())
		}
	}

	if models, err := ollamaModels(); err == nil {
		fmt.Fprintf(out, "%s Ollama           reachable, %d %s pulled: %s\n",
			ok(), len(models), pluralize(len(models), "model", "models"), strings.Join(models, ", "))
	} else {
		fmt.Fprintf(out, "%s Ollama           not reachable (optional, only needed for --explain)\n", info())
	}

	fmt.Fprintln(out)
	if allOK {
		fmt.Fprintln(out, "Ready.")
		return nil
	}
	return fmt.Errorf("environment checks failed")
}

// ollamaModels does a short-timeout check against the local Ollama
// daemon to list which models are pulled. This is the only network call
// serval doctor ever makes, and it is always to localhost/$OLLAMA_HOST,
// never a remote host.
func ollamaModels() ([]string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 500*time.Millisecond)
	defer cancel()
	return ollama.New("", "").ListModels(ctx)
}

// pluralize returns singular when n == 1, plural otherwise.
func pluralize(n int, singular, plural string) string {
	if n == 1 {
		return singular
	}
	return plural
}
