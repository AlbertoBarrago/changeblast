package cmd

import (
	"fmt"
	"io"
	"os"

	"github.com/spf13/cobra"

	"github.com/AlbertoBarrago/serval/internal/config"
	"github.com/AlbertoBarrago/serval/internal/git"
	"github.com/AlbertoBarrago/serval/internal/output"
	"github.com/AlbertoBarrago/serval/internal/risk"
)

var (
	diffFlags   analysisFlags
	diffExplain *explainFlags
)

var diffCmd = &cobra.Command{
	Use:   "diff [<ref>]",
	Short: "Analyze the blast radius of changed files",
	Long: `Compute impact for the set of recognized-module files changed between
<ref> and the current working tree (including uncommitted changes).
Default <ref> is HEAD, i.e. uncommitted changes only.`,
	Args: cobra.MaximumNArgs(1),
	RunE: runDiff,
}

func init() {
	addAnalysisFlags(diffCmd, &diffFlags)
	diffExplain = addExplainFlags(diffCmd, " (one call per changed file — can be slow)")
	rootCmd.AddCommand(diffCmd)
}

func runDiff(c *cobra.Command, args []string) error {
	if err := validateFailOn(diffFlags.failOn); err != nil {
		return err
	}

	ref := "HEAD"
	if len(args) == 1 {
		ref = args[0]
	}

	cwd, err := os.Getwd()
	if err != nil {
		return err
	}
	root, err := repositoryRoot(cwd)
	if err != nil {
		return err
	}

	w, closeOut, err := openOutputTarget(c.OutOrStdout(), diffFlags.output)
	if err != nil {
		return err
	}
	defer closeOut()

	changed, err := git.ChangedFiles(root, ref)
	if err != nil {
		return fmt.Errorf("failed to compute changed files against %q: %w", ref, err)
	}

	// Scan the repository once and reuse the resulting graph for every
	// changed file, rather than rescanning per file.
	g, err := buildGraph(root)
	if err != nil {
		return err
	}

	cfg, err := config.Load(root)
	if err != nil {
		return err
	}

	var results []output.InspectResult
	worstLevel := risk.LevelLow

	for _, file := range changed {
		// ChangedFiles returns absolute paths, so this is cwd-independent.
		if _, err := os.Stat(file); err != nil {
			// Deleted or renamed-away file: nothing to inspect on disk.
			continue
		}

		result, err := inspectWithGraph(root, g, file, cfg)
		if err != nil {
			// Not a recognized module (e.g. a config file changed):
			// skip rather than aborting the whole diff, but say so.
			fmt.Fprintf(os.Stderr, "warning: skipping %s: %v\n", file, err)
			continue
		}

		if riskLevelRank[result.Risk.Level] > riskLevelRank[worstLevel] {
			worstLevel = result.Risk.Level
		}

		results = append(results, result)
	}

	// One --explain call per changed file, sequentially: see
	// runInspectDirectory's identical tradeoff note in cmd/inspect.go.
	explanations := make([]string, len(results))
	explainErrs := make([]error, len(results))
	if diffExplain.enabled {
		for i, r := range results {
			explanations[i], explainErrs[i] = explainResult(c.Context(), diffExplain, r)
		}
	}

	if diffFlags.json {
		if err := encodeResultsJSON(w, root, results, explanations, explainErrs, diffExplain.enabled); err != nil {
			return err
		}
	} else {
		renderDiffText(w, root, ref, results, explanations, explainErrs)
	}

	return applyFailOn(diffFlags.failOn, worstLevel)
}

func renderDiffText(w io.Writer, root, ref string, results []output.InspectResult, explanations []string, explainErrs []error) {
	if len(results) == 0 {
		fmt.Fprintf(w, "No recognized-module changes found against %s.\n", ref)
		return
	}

	for i, r := range results {
		if i > 0 {
			fmt.Fprintln(w)
			fmt.Fprintln(w, "---")
			fmt.Fprintln(w)
		}
		output.RenderInspectFull(w, root, r)
		if diffExplain.enabled {
			renderExplanation(w, diffExplain.provider, explanations[i], explainErrs[i])
		}
	}
}
