package cmd

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"

	"github.com/AlbertoBarrago/changeblast/internal/ai"
	"github.com/AlbertoBarrago/changeblast/internal/ai/ollama"
	"github.com/AlbertoBarrago/changeblast/internal/ci"
	githubci "github.com/AlbertoBarrago/changeblast/internal/ci/github"
	gitlabci "github.com/AlbertoBarrago/changeblast/internal/ci/gitlab"
	"github.com/AlbertoBarrago/changeblast/internal/config"
	"github.com/AlbertoBarrago/changeblast/internal/git"
	"github.com/AlbertoBarrago/changeblast/internal/graph"
	"github.com/AlbertoBarrago/changeblast/internal/impact"
	"github.com/AlbertoBarrago/changeblast/internal/output"
	"github.com/AlbertoBarrago/changeblast/internal/repository"
	"github.com/AlbertoBarrago/changeblast/internal/risk"
)

var (
	inspectJSON         bool
	inspectFailOn       string
	inspectOutput       string
	inspectExplain      bool
	inspectExplainHost  string
	inspectExplainModel string
)

var inspectCmd = &cobra.Command{
	Use:   "inspect <path>",
	Short: "Analyze the blast radius of a file or directory",
	Long: `Analyze a file's direct and indirect dependents within the dependency
graph, plus Git history, relevant CI workflows, and an explainable risk
score. Given a directory (e.g. "blast inspect ." or "blast inspect src"),
every recognized module inside it is analyzed and reported as a
risk-sorted summary instead of one full per-file report. <path> defaults
to "." (the current directory) if omitted. See "blast --help" for the
v0.1 module-resolution scope and limitations (JavaScript/TypeScript, Go,
Python).`,
	Args: cobra.MaximumNArgs(1),
	RunE: runInspect,
}

func init() {
	inspectCmd.Flags().BoolVar(&inspectJSON, "json", false, "output machine-readable JSON")
	inspectCmd.Flags().StringVar(&inspectFailOn, "fail-on", "", "exit with code 2 if risk is at or above this level (low, medium, high)")
	inspectCmd.Flags().BoolVar(&inspectExplain, "explain", false, "ask a local Ollama model to explain the risk in natural language (single-file target only; requires Ollama running locally)")
	inspectCmd.Flags().StringVar(&inspectExplainHost, "explain-host", "", "Ollama host (default: $OLLAMA_HOST or http://localhost:11434)")
	inspectCmd.Flags().StringVar(&inspectExplainModel, "explain-model", "", "Ollama model to use (default: "+ollama.DefaultModel+")")
	addOutputFlag(inspectCmd, &inspectOutput)
	rootCmd.AddCommand(inspectCmd)
}

func runInspect(c *cobra.Command, args []string) error {
	target, root, err := resolveTarget(targetArg(args))
	if err != nil {
		return err
	}

	w, closeOut, err := openOutputTarget(c.OutOrStdout(), inspectOutput)
	if err != nil {
		return err
	}
	defer closeOut()

	info, err := os.Stat(target)
	if err != nil {
		return err
	}
	if info.IsDir() {
		return runInspectDirectory(w, root, target)
	}

	result, err := inspectTarget(root, target)
	if err != nil {
		return err
	}

	explanation, explainErr := maybeExplain(c, result)

	if inspectJSON {
		enc := json.NewEncoder(w)
		enc.SetIndent("", "  ")

		// Only wrap the response in {analysis, explanation} when --explain
		// was actually requested, so the default --json shape is unchanged
		// for existing scripts/CI consumers.
		var encodeErr error
		if inspectExplain {
			body := explainedJSON{Analysis: output.ToInspectFullJSON(root, result)}
			if explanation != "" {
				body.Explanation = explanation
			}
			if explainErr != nil {
				body.ExplainError = explainErr.Error()
			}
			encodeErr = enc.Encode(body)
		} else {
			encodeErr = enc.Encode(output.ToInspectFullJSON(root, result))
		}
		if encodeErr != nil {
			return encodeErr
		}
	} else {
		output.RenderInspectFull(w, root, result)
		renderExplanation(w, explanation, explainErr)
	}

	return applyFailOn(inspectFailOn, result.Risk.Level)
}

// explainedJSON wraps the deterministic analysis with an optional AI
// explanation. Kept in cmd rather than internal/output so that package
// stays free of any dependency on internal/ai.
type explainedJSON struct {
	Analysis     output.InspectFullJSON `json:"analysis"`
	Explanation  string                 `json:"explanation,omitempty"`
	ExplainError string                 `json:"explainError,omitempty"`
}

// maybeExplain calls the configured AI provider when --explain was
// passed, translating an InspectResult into an ai.Finding. It returns
// ("", nil) when --explain was not requested — no network call is made
// in that case.
func maybeExplain(c *cobra.Command, result output.InspectResult) (string, error) {
	if !inspectExplain {
		return "", nil
	}

	breakdown := make([]string, len(result.Risk.Breakdown))
	for i, e := range result.Risk.Breakdown {
		breakdown[i] = fmt.Sprintf("+%d %s", e.Points, e.Reason)
	}

	relTarget := result.Impact.Target
	workflowPaths := make([]string, len(result.RelevantWorkflows))
	for i, wf := range result.RelevantWorkflows {
		workflowPaths[i] = wf.Path
	}

	finding := ai.Finding{
		Target:            relTarget,
		DirectImpact:      result.Impact.Direct,
		IndirectImpact:    result.Impact.Indirect,
		RiskLevel:         string(result.Risk.Level),
		RiskScore:         result.Risk.Total,
		RiskBreakdown:     breakdown,
		HistoryChanges:    result.History.Changes,
		HistoryWindow:     result.History.Window.Days,
		RelevantWorkflows: workflowPaths,
	}

	provider := ollama.New(inspectExplainHost, inspectExplainModel)
	return provider.Explain(c.Context(), finding)
}

// renderExplanation prints the AI explanation section (or a warning if
// it failed) to w. A failed explanation is never fatal: the deterministic
// analysis above it stands on its own.
func renderExplanation(w io.Writer, explanation string, err error) {
	if explanation == "" && err == nil {
		return
	}
	fmt.Fprintln(w)
	fmt.Fprintln(w, "Explanation (ollama)")
	if err != nil {
		fmt.Fprintf(w, "  unavailable: %v\n", err)
		return
	}
	for _, line := range strings.Split(output.StripMarkdown(w, explanation), "\n") {
		fmt.Fprintf(w, "  %s\n", line)
	}
}

// runInspectDirectory analyzes every recognized module found under dir (an
// absolute path within root) and renders a risk-sorted summary, since
// printing the full per-file report used for a single-file target would
// be unusable across potentially hundreds of files.
func runInspectDirectory(w io.Writer, root, dir string) error {
	g, err := buildGraph(root)
	if err != nil {
		return err
	}

	cfg, err := config.Load(root)
	if err != nil {
		return err
	}

	var results []output.InspectResult
	worst := risk.LevelLow
	for _, node := range g.Nodes() {
		if !isWithinDir(node, dir) {
			continue
		}
		result, err := inspectWithGraph(root, g, node, cfg)
		if err != nil {
			continue
		}
		if riskLevelRank[result.Risk.Level] > riskLevelRank[worst] {
			worst = result.Risk.Level
		}
		results = append(results, result)
	}

	if inspectJSON {
		jsonResults := make([]output.InspectFullJSON, len(results))
		for i, r := range results {
			jsonResults[i] = output.ToInspectFullJSON(root, r)
		}
		enc := json.NewEncoder(w)
		enc.SetIndent("", "  ")
		if err := enc.Encode(jsonResults); err != nil {
			return err
		}
	} else {
		header := "Analyzed " + displayDir(root, dir)
		output.RenderSummary(w, root, header, results)
	}

	return applyFailOn(inspectFailOn, worst)
}

// displayDir renders dir relative to root for the summary header,
// falling back to "." when dir is root itself.
func displayDir(root, dir string) string {
	rel, err := filepath.Rel(root, dir)
	if err != nil || rel == "" {
		return "."
	}
	return rel
}

// isWithinDir reports whether path is dir itself or lives inside it.
func isWithinDir(path, dir string) bool {
	rel, err := filepath.Rel(dir, path)
	if err != nil {
		return false
	}
	return rel == "." || (rel != ".." && !strings.HasPrefix(rel, ".."+string(filepath.Separator)))
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
	cfg, err := config.Load(root)
	if err != nil {
		return output.InspectResult{}, err
	}
	return inspectWithGraph(root, g, target, cfg)
}

// inspectWithGraph runs the full analysis pipeline for target against an
// already-scanned repository graph g, using cfg to resolve
// .changeblast.yml overrides (critical-path keywords, history window)
// on top of their built-in defaults.
// ciProviders is the fixed set of CI providers blast checks for
// relevant workflows. Adding a provider here is the only step needed
// (ci.Provider exists specifically so this stays a flat list, no
// per-provider branching elsewhere).
var ciProviders = []ci.Provider{githubci.New(), gitlabci.New()}

// discoverWorkflows runs every registered CI provider against root and
// merges their results. A provider with no config file present (e.g. no
// .gitlab-ci.yml in a GitHub-only repository) contributes nothing, not
// an error.
func discoverWorkflows(root string) []ci.Workflow {
	var workflows []ci.Workflow
	for _, p := range ciProviders {
		wf, _ := p.Discover(root)
		workflows = append(workflows, wf...)
	}
	return workflows
}

func inspectWithGraph(root string, g *graph.Graph, target string, cfg config.Config) (output.InspectResult, error) {
	if !g.HasNode(target) {
		rel, _ := filepath.Rel(root, target)
		return output.InspectResult{}, fmt.Errorf("%s is not a recognized module in this repository", rel)
	}

	impactResult := impact.Compute(g, target)

	window := git.Window{
		Days:       cfg.HistoryWindowDaysOr(git.HistoryWindowDays),
		MaxCommits: cfg.HistoryWindowMaxCommitsOr(git.HistoryWindowMaxCommits),
	}

	// Git history and CI relevance are best-effort: a target outside a
	// Git repository, or a repository with no CI workflows, still gets
	// a valid dependency-only inspect result.
	history, _ := git.AnalyzeWithWindow(root, target, window)

	relTarget, err := filepath.Rel(root, target)
	if err != nil {
		relTarget = target
	}
	relTarget = filepath.ToSlash(relTarget)

	workflows := discoverWorkflows(root)
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
		CriticalPathKeywords:  cfg.CriticalPathsOr(risk.DefaultCriticalPathKeywords),
	})

	return output.InspectResult{
		Impact:            impactResult,
		History:           history,
		RelevantWorkflows: relevant,
		Risk:              score,
	}, nil
}
