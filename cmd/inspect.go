package cmd

import (
	"encoding/json"
	"fmt"
	"path/filepath"

	"github.com/spf13/cobra"

	"github.com/AlbertoBarrago/changeblast/internal/ci"
	githubci "github.com/AlbertoBarrago/changeblast/internal/ci/github"
	"github.com/AlbertoBarrago/changeblast/internal/git"
	"github.com/AlbertoBarrago/changeblast/internal/graph"
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
	target, root, err := resolveTarget(args[0])
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

// buildGraph scans root and returns its dependency graph. Callers that
// need to inspect multiple targets in the same repository (e.g. `blast
// diff`) should call this once and reuse the graph via inspectWithGraph,
// rather than rescanning per target.
func buildGraph(root string) (*graph.Graph, error) {
	scanner, err := repository.NewScanner(root)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize scanner: %w", err)
	}

	g, err := scanner.Scan()
	if err != nil {
		return nil, fmt.Errorf("failed to scan repository: %w", err)
	}
	return g, nil
}

// inspectTarget runs the full analysis pipeline (scan, impact, history,
// CI relevance, risk) for a single target file, scanning the repository
// fresh. For analyzing multiple targets in the same repository, prefer
// buildGraph once followed by inspectWithGraph per target.
func inspectTarget(root, target string) (output.InspectResult, error) {
	g, err := buildGraph(root)
	if err != nil {
		return output.InspectResult{}, err
	}
	return inspectWithGraph(root, g, target)
}

// inspectWithGraph runs the full analysis pipeline for target against an
// already-scanned repository graph g.
func inspectWithGraph(root string, g *graph.Graph, target string) (output.InspectResult, error) {
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

	frequent := output.FrequentCoChangeCount(history)

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
