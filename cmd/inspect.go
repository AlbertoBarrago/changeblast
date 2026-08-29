package cmd

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"

	"github.com/AlbertoBarrago/serval/internal/ci"
	azureci "github.com/AlbertoBarrago/serval/internal/ci/azure"
	bitbucketci "github.com/AlbertoBarrago/serval/internal/ci/bitbucket"
	circleci "github.com/AlbertoBarrago/serval/internal/ci/circleci"
	githubci "github.com/AlbertoBarrago/serval/internal/ci/github"
	gitlabci "github.com/AlbertoBarrago/serval/internal/ci/gitlab"
	jenkinsci "github.com/AlbertoBarrago/serval/internal/ci/jenkins"
	"github.com/AlbertoBarrago/serval/internal/config"
	"github.com/AlbertoBarrago/serval/internal/git"
	"github.com/AlbertoBarrago/serval/internal/graph"
	"github.com/AlbertoBarrago/serval/internal/impact"
	"github.com/AlbertoBarrago/serval/internal/output"
	"github.com/AlbertoBarrago/serval/internal/repository"
	"github.com/AlbertoBarrago/serval/internal/risk"
	"github.com/AlbertoBarrago/serval/internal/testsignal"
)

var (
	inspectFlags   analysisFlags
	inspectExplain *explainFlags
)

var inspectCmd = &cobra.Command{
	Use:   "inspect <path>",
	Short: "Analyze the blast radius of a file or directory",
	Long: `Analyze a file's direct and indirect dependents within the dependency
graph, plus Git history, relevant CI workflows, and an explainable risk
score. Given a directory (e.g. "serval inspect ." or "serval
inspect src"), every recognized module inside it is analyzed and
reported as a risk-sorted summary instead of one full per-file report.
<path> defaults to "." (the current directory) if omitted. See
"serval --help" for the v0.1 module-resolution scope and
limitations (JavaScript/TypeScript, Go, Python).

An optional .serval.yml at the repository root can override the
critical-path keyword list (criticalPaths), the Git history window
(historyWindow), and floor specific paths' risk level at HIGH
regardless of their computed score (highRiskPaths, glob patterns with
"**" support, e.g. "**/migrations/**"). See docs/usage.md for the full
schema and defaults.`,
	Args: cobra.MaximumNArgs(1),
	RunE: runInspect,
}

func init() {
	addAnalysisFlags(inspectCmd, &inspectFlags)
	inspectExplain = addExplainFlags(inspectCmd, " (one call per file — can be slow across a directory)")
	rootCmd.AddCommand(inspectCmd)
}

func runInspect(c *cobra.Command, args []string) error {
	if err := validateFailOn(inspectFlags.failOn); err != nil {
		return err
	}

	target, root, err := resolveTarget(targetArg(args))
	if err != nil {
		return err
	}

	w, closeOut, err := openOutputTarget(c.OutOrStdout(), inspectFlags.output)
	if err != nil {
		return err
	}
	defer closeOut()

	info, err := os.Stat(target)
	if err != nil {
		return err
	}
	if info.IsDir() {
		return runInspectDirectory(c.Context(), w, root, target)
	}

	result, err := inspectTarget(root, target)
	if err != nil {
		return err
	}

	explanation, explainErr := explainResult(c.Context(), inspectExplain, result)

	if inspectFlags.json {
		enc := json.NewEncoder(w)
		enc.SetIndent("", "  ")

		// Only wrap the response in {analysis, explanation} when --explain
		// was actually requested, so the default --json shape is unchanged
		// for existing scripts/CI consumers.
		var encodeErr error
		if inspectExplain.enabled {
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
		renderExplanation(w, inspectExplain.provider, explanation, explainErr)
	}

	return applyFailOn(inspectFlags.failOn, result.Risk.Level)
}

// runInspectDirectory analyzes every recognized module found under dir (an
// absolute path within root) and renders a risk-sorted summary, since
// printing the full per-file report used for a single-file target would
// be unusable across potentially hundreds of files.
func runInspectDirectory(ctx context.Context, w io.Writer, root, dir string) error {
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
			// Unrecognized or unanalyzable module: note it and keep going
			// rather than silently underreporting the blast radius.
			fmt.Fprintf(os.Stderr, "warning: skipping %s: %v\n", node, err)
			continue
		}
		if riskLevelRank[result.Risk.Level] > riskLevelRank[worst] {
			worst = result.Risk.Level
		}
		results = append(results, result)
	}

	// One --explain call per file, sequentially: slow across a large
	// directory, but that's the documented tradeoff for getting
	// explanations at all here (see docs/architecture.md) rather than
	// guessing at a "reasonable" concurrency limit across three very
	// different backends (a local daemon, two agent CLIs).
	explanations := make([]string, len(results))
	explainErrs := make([]error, len(results))
	if inspectExplain.enabled {
		for i, r := range results {
			explanations[i], explainErrs[i] = explainResult(ctx, inspectExplain, r)
		}
	}

	if inspectFlags.json {
		if err := encodeResultsJSON(w, root, results, explanations, explainErrs, inspectExplain.enabled); err != nil {
			return err
		}
	} else {
		header := "Analyzed " + displayDir(root, dir)
		output.RenderSummary(w, root, header, results)

		for i, r := range results {
			if explanations[i] == "" && explainErrs[i] == nil {
				continue
			}
			rel, err := filepath.Rel(root, r.Impact.Target)
			if err != nil {
				rel = r.Impact.Target
			}
			fmt.Fprintln(w)
			fmt.Fprintln(w, rel)
			renderExplanation(w, inspectExplain.provider, explanations[i], explainErrs[i])
		}
	}

	return applyFailOn(inspectFlags.failOn, worst)
}

// encodeResultsJSON writes a multi-result JSON array: bare InspectFullJSON
// entries by default, or {analysis, explanation, explainError} wrappers
// when --explain was requested. Shared by inspect (directory mode) and
// diff so both commands emit the same JSON shape.
func encodeResultsJSON(w io.Writer, root string, results []output.InspectResult, explanations []string, explainErrs []error, explained bool) error {
	enc := json.NewEncoder(w)
	enc.SetIndent("", "  ")

	if explained {
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
		return enc.Encode(jsonResults)
	}

	jsonResults := make([]output.InspectFullJSON, len(results))
	for i, r := range results {
		jsonResults[i] = output.ToInspectFullJSON(root, r)
	}
	return enc.Encode(jsonResults)
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
// need to inspect multiple targets in the same repository (e.g. `serval
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
// .serval.yml overrides (critical-path keywords, history window)
// on top of their built-in defaults.
// ciProviders is the fixed set of CI providers serval checks for
// relevant workflows. Adding a provider here is the only step needed
// (ci.Provider exists specifically so this stays a flat list, no
// per-provider branching elsewhere).
var ciProviders = []ci.Provider{githubci.New(), gitlabci.New(), azureci.New(), jenkinsci.New(), circleci.New(), bitbucketci.New()}

// discoverWorkflows runs every registered CI provider against root and
// merges their results. A provider with no config file present (e.g. no
// .gitlab-ci.yml in a GitHub-only repository) contributes nothing, not
// an error; a provider that fails for a real reason (malformed config,
// unreadable file) is reported on stderr and skipped, so a broken
// workflow file cannot silently zero out the CI signal.
func discoverWorkflows(root string) []ci.Workflow {
	var workflows []ci.Workflow
	for _, p := range ciProviders {
		wf, err := p.Discover(root)
		if err != nil {
			fmt.Fprintf(os.Stderr, "warning: skipping CI provider %s: %v\n", p.Name(), err)
			continue
		}
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
	// a valid dependency-only inspect result. But a git that *fails*
	// (not merely "no history") must be visible: a silently dropped
	// history signal means churn and co-change scores of zero that look
	// like a quiet file, not a broken analysis.
	history, err := git.AnalyzeWithWindow(root, target, window)
	if err != nil {
		fmt.Fprintf(os.Stderr, "warning: skipping git history analysis: %v\n", err)
		history = git.FileHistory{Path: target, Window: window}
	}

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
		HighRiskPaths:         cfg.HighRiskPathsOr(risk.DefaultHighRiskPaths),
		NoCorrelatedTest:      !testsignal.HasCorrelatedTest(root, relTarget),
	})

	return output.InspectResult{
		Impact:            impactResult,
		History:           history,
		RelevantWorkflows: relevant,
		Risk:              score,
	}, nil
}
