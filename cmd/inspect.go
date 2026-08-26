package cmd

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"

	"github.com/AlbertoBarrago/changeblast/internal/ci"
	githubci "github.com/AlbertoBarrago/changeblast/internal/ci/github"
	"github.com/AlbertoBarrago/changeblast/internal/git"
	"github.com/AlbertoBarrago/changeblast/internal/impact"
	"github.com/AlbertoBarrago/changeblast/internal/output"
	"github.com/AlbertoBarrago/changeblast/internal/repository"
	"github.com/AlbertoBarrago/changeblast/internal/risk"
)

var (
	inspectJSON   bool
	inspectFailOn string
)

var inspectCmd = &cobra.Command{
	Use:   "inspect <path>",
	Short: "Analyze the blast radius of a file",
	Long: `Analyze a file's direct and indirect dependents within the JS/TS module
graph, plus Git history, relevant CI workflows, and an explainable risk
score. See "blast --help" for the v0.1 module-resolution scope and
limitations.`,
	Args: cobra.ExactArgs(1),
	RunE: runInspect,
}

func init() {
	inspectCmd.Flags().BoolVar(&inspectJSON, "json", false, "output machine-readable JSON")
	inspectCmd.Flags().StringVar(&inspectFailOn, "fail-on", "", "exit with code 2 if risk is at or above this level (low, medium, high)")
	rootCmd.AddCommand(inspectCmd)
}

func runInspect(c *cobra.Command, args []string) error {
	target, err := filepath.Abs(args[0])
	if err != nil {
		return err
	}
	if _, err := os.Stat(target); err != nil {
		return fmt.Errorf("target not found: %s", args[0])
	}

	root, err := repositoryRoot(target)
	if err != nil {
		return err
	}

	result, err := inspectTarget(root, target)
	if err != nil {
		return err
	}

	if inspectJSON {
		enc := json.NewEncoder(c.OutOrStdout())
		enc.SetIndent("", "  ")
		if err := enc.Encode(output.ToInspectFullJSON(root, result)); err != nil {
			return err
		}
	} else {
		output.RenderInspectFull(c.OutOrStdout(), root, result)
	}

	return applyFailOn(inspectFailOn, result.Risk.Level)
}

// inspectTarget runs the full analysis pipeline (scan, impact, history,
// CI relevance, risk) for a single target file.
func inspectTarget(root, target string) (output.InspectResult, error) {
	scanner, err := repository.NewScanner(root)
	if err != nil {
		return output.InspectResult{}, fmt.Errorf("failed to initialize scanner: %w", err)
	}

	g, err := scanner.Scan()
	if err != nil {
		return output.InspectResult{}, fmt.Errorf("failed to scan repository: %w", err)
	}

	if !g.HasNode(target) {
		rel, _ := filepath.Rel(root, target)
		return output.InspectResult{}, fmt.Errorf("%s is not a recognized JS/TS module in this repository", rel)
	}

	impactResult := impact.Compute(g, target)

	// Git history and CI relevance are best-effort: a target outside a
	// Git repository, or a repository with no GitHub Actions workflows,
	// still gets a valid dependency-only inspect result.
	history, _ := git.Analyze(root, target)

	relTarget, err := filepath.Rel(root, target)
	if err != nil {
		relTarget = target
	}
	relTarget = filepath.ToSlash(relTarget)

	workflows, _ := githubci.New().Discover(root)
	relevant := ci.Relevant(workflows, []string{relTarget})

	frequent := 0
	for _, co := range history.CoChanged {
		if co.Count >= 2 {
			frequent++
		}
	}

	workflowNames := make([]string, len(relevant))
	for i, wf := range relevant {
		workflowNames[i] = wf.Path
	}

	score := risk.Compute(risk.Input{
		TargetPath:            relTarget,
		DownstreamCount:       len(impactResult.Direct) + len(impactResult.Indirect),
		ChurnCount:            history.Changes,
		FrequentCoChangeCount: frequent,
		RelevantWorkflows:     workflowNames,
	})

	return output.InspectResult{
		Impact:            impactResult,
		History:           history,
		RelevantWorkflows: relevant,
		Risk:              score,
	}, nil
}

// applyFailOn returns an error carrying exit code 2 semantics (handled
// in Execute) when failOn is set and level meets or exceeds it.
func applyFailOn(failOn string, level risk.Level) error {
	if failOn == "" {
		return nil
	}

	threshold, ok := riskLevelRank[risk.Level(normalizeLevel(failOn))]
	if !ok {
		return fmt.Errorf("invalid --fail-on value %q: must be one of low, medium, high", failOn)
	}

	if riskLevelRank[level] >= threshold {
		return failOnError{level: level}
	}
	return nil
}

var riskLevelRank = map[risk.Level]int{
	risk.LevelLow:    1,
	risk.LevelMedium: 2,
	risk.LevelHigh:   3,
}

func normalizeLevel(s string) string {
	switch s {
	case "low", "LOW":
		return string(risk.LevelLow)
	case "medium", "MEDIUM":
		return string(risk.LevelMedium)
	case "high", "HIGH":
		return string(risk.LevelHigh)
	default:
		return s
	}
}

// failOnError signals that risk threshold gating (--fail-on) tripped.
// Execute() maps this to exit code 2, per the documented exit code
// contract.
type failOnError struct {
	level risk.Level
}

func (e failOnError) Error() string {
	return fmt.Sprintf("risk level %s meets or exceeds --fail-on threshold", e.level)
}
