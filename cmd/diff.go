package cmd

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"

	"github.com/AlbertoBarrago/serval/internal/config"
	"github.com/AlbertoBarrago/serval/internal/git"
	"github.com/AlbertoBarrago/serval/internal/graph"
	"github.com/AlbertoBarrago/serval/internal/impact"
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
Default <ref> is HEAD, i.e. uncommitted changes only.

Each changed file gets its own report, and when at least two files are
analyzed, the whole change set is also scored as one: downstream impact
is deduplicated across the set, dependency edges between changed files
(interacting changes) and unchanged modules importing multiple changed
files (shared dependents) are surfaced separately, and CI relevance is
unioned. The set-level risk follows the same explainable rule-based
model as the per-file score.

An optional .serval.yml at the repository root can override the
critical-path keyword list (criticalPaths), the Git history window
(historyWindow), and floor specific paths' risk level at HIGH
regardless of their computed score (highRiskPaths, glob patterns with
"**" support, e.g. "**/migrations/**"). See docs/usage.md for the full
schema and defaults.`,
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

	format, err := resolveFormat(diffFlags)
	if err != nil {
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

	// Set-level analysis only makes sense with at least two analyzed
	// files; a single-file diff's set result would just duplicate the
	// per-file report.
	var setResult impact.SetResult
	var setScore risk.Score
	hasSet := len(results) >= 2
	if hasSet {
		setResult, setScore = analyzeChangeSet(root, g, results, cfg)
		if riskLevelRank[setScore.Level] > riskLevelRank[worstLevel] {
			worstLevel = setScore.Level
		}
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

	switch format {
	case "sarif":
		enc := json.NewEncoder(w)
		enc.SetIndent("", "  ")
		if err := enc.Encode(output.ToSARIF(root, results)); err != nil {
			return err
		}
	case "json":
		if err := encodeDiffJSON(w, root, results, setResult, setScore, hasSet, explanations, explainErrs, diffExplain.enabled); err != nil {
			return err
		}
	default:
		renderDiffText(w, root, ref, results, hasSet, setResult, setScore, explanations, explainErrs)
	}

	return applyFailOn(diffFlags.failOn, worstLevel)
}

// analyzeChangeSet aggregates the per-file results into a set-level
// impact and risk score: downstream impact deduplicated across the set,
// interactions between changed files, shared dependents, the union of
// relevant CI workflows, and the maximum churn across the set.
func analyzeChangeSet(root string, g *graph.Graph, results []output.InspectResult, cfg config.Config) (impact.SetResult, risk.Score) {
	targets := make([]string, len(results))
	maxChurn := 0
	workflowSet := make(map[string]bool)
	var workflows []string
	for i, r := range results {
		targets[i] = r.Impact.Target
		if r.History.Changes > maxChurn {
			maxChurn = r.History.Changes
		}
		for _, wf := range r.RelevantWorkflows {
			if !workflowSet[wf.Path] {
				workflowSet[wf.Path] = true
				workflows = append(workflows, wf.Path)
			}
		}
	}

	setResult := impact.ComputeSet(g, targets)

	relTargets := make([]string, len(targets))
	for i, t := range targets {
		rel, err := filepath.Rel(root, t)
		if err != nil {
			rel = t
		}
		relTargets[i] = filepath.ToSlash(rel)
	}

	score := risk.ComputeSet(risk.SetInput{
		TargetPaths:          relTargets,
		DownstreamCount:      len(setResult.Direct) + len(setResult.Indirect),
		InternalEdgeCount:    len(setResult.InternalEdges),
		SharedDependentCount: len(setResult.SharedDependents),
		ChurnCount:           maxChurn,
		RelevantWorkflows:    workflows,
		CriticalPathKeywords: cfg.CriticalPathsOr(risk.DefaultCriticalPathKeywords),
		HighRiskPaths:        cfg.HighRiskPathsOr(risk.DefaultHighRiskPaths),
	})

	return setResult, score
}

// diffJSON is the --json envelope for `serval diff`: the per-file results
// plus, when at least two files were analyzed, the change-set-level
// result. This shape replaced the bare results array emitted before
// 0.1.21 — see CHANGELOG.
type diffJSON struct {
	ChangeSet *output.SetJSON `json:"changeSet,omitempty"`
	Files     []any           `json:"files"`
}

func encodeDiffJSON(w io.Writer, root string, results []output.InspectResult, setResult impact.SetResult, setScore risk.Score, hasSet bool, explanations []string, explainErrs []error, explained bool) error {
	enc := json.NewEncoder(w)
	enc.SetIndent("", "  ")

	body := diffJSON{Files: make([]any, len(results))}
	if hasSet {
		set := output.ToSetJSON(root, setResult, setScore)
		body.ChangeSet = &set
	}

	for i, r := range results {
		if !explained {
			body.Files[i] = output.ToInspectFullJSON(root, r)
			continue
		}
		entry := explainedJSON{Analysis: output.ToInspectFullJSON(root, r)}
		if explanations[i] != "" {
			entry.Explanation = explanations[i]
		}
		if explainErrs[i] != nil {
			entry.ExplainError = explainErrs[i].Error()
		}
		body.Files[i] = entry
	}

	return enc.Encode(body)
}

func renderDiffText(w io.Writer, root, ref string, results []output.InspectResult, hasSet bool, setResult impact.SetResult, setScore risk.Score, explanations []string, explainErrs []error) {
	if len(results) == 0 {
		fmt.Fprintf(w, "No recognized-module changes found against %s.\n", ref)
		return
	}

	if hasSet {
		output.RenderSetText(w, root, setResult, setScore)
		fmt.Fprintln(w)
		fmt.Fprintln(w, "===")
		fmt.Fprintln(w)
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
