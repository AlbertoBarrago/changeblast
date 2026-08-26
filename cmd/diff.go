package cmd

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/spf13/cobra"

	"github.com/AlbertoBarrago/changeblast/internal/git"
	"github.com/AlbertoBarrago/changeblast/internal/output"
	"github.com/AlbertoBarrago/changeblast/internal/risk"
)

var (
	diffJSON   bool
	diffFailOn string
)

var diffCmd = &cobra.Command{
	Use:   "diff [<ref>]",
	Short: "Analyze the blast radius of changed files",
	Long: `Compute impact for the set of JS/TS files changed between <ref> and the
current working tree (including uncommitted changes). Default <ref> is
HEAD, i.e. uncommitted changes only.`,
	Args: cobra.MaximumNArgs(1),
	RunE: runDiff,
}

func init() {
	diffCmd.Flags().BoolVar(&diffJSON, "json", false, "output machine-readable JSON")
	diffCmd.Flags().StringVar(&diffFailOn, "fail-on", "", "exit with code 2 if any changed file's risk is at or above this level (low, medium, high)")
	rootCmd.AddCommand(diffCmd)
}

func runDiff(c *cobra.Command, args []string) error {
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

	var results []output.InspectResult
	var jsonResults []output.InspectFullJSON
	worstLevel := risk.LevelLow

	for _, file := range changed {
		if _, err := os.Stat(file); err != nil {
			// Deleted or renamed-away file: nothing to inspect on disk.
			continue
		}

		result, err := inspectWithGraph(root, g, file)
		if err != nil {
			// Not a recognized JS/TS module (e.g. a config file changed):
			// skip rather than aborting the whole diff.
			continue
		}

		if riskLevelRank[result.Risk.Level] > riskLevelRank[worstLevel] {
			worstLevel = result.Risk.Level
		}

		if diffJSON {
			jsonResults = append(jsonResults, output.ToInspectFullJSON(root, result))
		} else {
			results = append(results, result)
		}
	}

	if diffJSON {
		enc := json.NewEncoder(c.OutOrStdout())
		enc.SetIndent("", "  ")
		if err := enc.Encode(jsonResults); err != nil {
			return err
		}
	} else {
		renderDiffText(c, root, ref, results)
	}

	return applyFailOn(diffFailOn, worstLevel)
}

func renderDiffText(c *cobra.Command, root, ref string, results []output.InspectResult) {
	w := c.OutOrStdout()
	if len(results) == 0 {
		fmt.Fprintf(w, "No JS/TS module changes found against %s.\n", ref)
		return
	}

	for i, r := range results {
		if i > 0 {
			fmt.Fprintln(w)
			fmt.Fprintln(w, "---")
			fmt.Fprintln(w)
		}
		output.RenderInspectFull(w, root, r)
	}
}
