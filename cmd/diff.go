package cmd

import (
	"encoding/json"
	"fmt"
	"io"
	"os"

	"github.com/spf13/cobra"

	"github.com/AlbertoBarrago/changeblast/internal/config"
	"github.com/AlbertoBarrago/changeblast/internal/git"
	"github.com/AlbertoBarrago/changeblast/internal/output"
	"github.com/AlbertoBarrago/changeblast/internal/risk"
)

var (
	diffJSON    bool
	diffFailOn  string
	diffOutput  string
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
	diffCmd.Flags().BoolVar(&diffJSON, "json", false, "output machine-readable JSON")
	diffCmd.Flags().StringVar(&diffFailOn, "fail-on", "", "exit with code 2 if any changed file's risk is at or above this level (low, medium, high)")
	diffExplain = addExplainFlags(diffCmd, " (one call per changed file — can be slow)")
	addOutputFlag(diffCmd, &diffOutput)
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

	w, closeOut, err := openOutputTarget(c.OutOrStdout(), diffOutput)
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
		if _, err := os.Stat(file); err != nil {
			// Deleted or renamed-away file: nothing to inspect on disk.
			continue
		}

		result, err := inspectWithGraph(root, g, file, cfg)
		if err != nil {
			// Not a recognized module (e.g. a config file changed):
			// skip rather than aborting the whole diff.
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

	if diffJSON {
		enc := json.NewEncoder(w)
		enc.SetIndent("", "  ")

		var encodeErr error
		if diffExplain.enabled {
			jsonResults := make([]explainedJSON, len(results))
			for i, r := range results {
				jsonResults[i] = explainedJSON{Analysis: output.ToInspectFullJSON(root, r)}
				if explanations[i] != "" {
					jsonResults[i].Explanation = explanations[i]
				}
				if explainErrs[i] != nil {
					jsonResults[i].ExplainError = explainErrs[i].Error()
				}
			}
			encodeErr = enc.Encode(jsonResults)
		} else {
			jsonResults := make([]output.InspectFullJSON, len(results))
			for i, r := range results {
				jsonResults[i] = output.ToInspectFullJSON(root, r)
			}
			encodeErr = enc.Encode(jsonResults)
		}
		if encodeErr != nil {
			return encodeErr
		}
	} else {
		renderDiffText(w, root, ref, results, explanations, explainErrs)
	}

	return applyFailOn(diffFailOn, worstLevel)
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
