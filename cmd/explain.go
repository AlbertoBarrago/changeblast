package cmd

import (
	"context"
	"fmt"
	"io"
	"strings"

	"github.com/spf13/cobra"

	"github.com/AlbertoBarrago/serval/internal/ai"
	"github.com/AlbertoBarrago/serval/internal/ai/localcli"
	"github.com/AlbertoBarrago/serval/internal/ai/ollama"
	"github.com/AlbertoBarrago/serval/internal/output"
)

// explainFlags holds the --explain* flag values for a command.
// inspect and diff each register their own instance (cobra flags are
// per-command), but share every other piece of --explain machinery in
// this file — findingFor, newExplainProvider, explainResult,
// renderExplanation, explainedJSON — so the two commands can't drift
// in how they build a Finding, pick a provider, or render a failure.
type explainFlags struct {
	enabled  bool
	provider string
	host     string
	model    string
}

// addExplainFlags registers the --explain flag set on cmd, backed by a
// fresh explainFlags (one instance per command, never shared).
func addExplainFlags(cmd *cobra.Command, helpSuffix string) *explainFlags {
	f := &explainFlags{}
	cmd.Flags().BoolVar(&f.enabled, "explain", false, "ask an AI provider to explain each result's risk in natural language"+helpSuffix)
	cmd.Flags().StringVar(&f.provider, "explain-provider", "ollama", "explain provider: ollama (local daemon), claude, codex, or gemini (local CLI, already authenticated)")
	cmd.Flags().StringVar(&f.host, "explain-host", "", "Ollama host, --explain-provider=ollama only (default: $OLLAMA_HOST or http://localhost:11434)")
	cmd.Flags().StringVar(&f.model, "explain-model", "", "model to use (default: "+ollama.DefaultModel+" for ollama, the provider's own default otherwise)")
	return f
}

// newExplainProvider builds the ai.Provider named by provider. host is
// only used by "ollama"; the local-CLI providers ignore it.
func newExplainProvider(provider, host, model string) (ai.Provider, error) {
	switch provider {
	case "", "ollama":
		return ollama.New(host, model), nil
	case "claude":
		return localcli.NewClaude(model), nil
	case "codex":
		return localcli.NewCodex(model), nil
	case "gemini":
		return localcli.NewGemini(model), nil
	default:
		return nil, fmt.Errorf("unknown --explain-provider %q (choices: ollama, claude, codex, gemini)", provider)
	}
}

// findingFor translates an InspectResult into the read-only ai.Finding
// summary a Provider explains — never source code, never anything the
// provider could use to influence the analysis.
func findingFor(result output.InspectResult) ai.Finding {
	breakdown := make([]string, len(result.Risk.Breakdown))
	for i, e := range result.Risk.Breakdown {
		breakdown[i] = fmt.Sprintf("+%d %s", e.Points, e.Reason)
	}

	workflowPaths := make([]string, len(result.RelevantWorkflows))
	for i, wf := range result.RelevantWorkflows {
		workflowPaths[i] = wf.Path
	}

	return ai.Finding{
		Target:            result.Impact.Target,
		DirectImpact:      result.Impact.Direct,
		IndirectImpact:    result.Impact.Indirect,
		RiskLevel:         string(result.Risk.Level),
		RiskScore:         result.Risk.Total,
		RiskBreakdown:     breakdown,
		HistoryChanges:    result.History.Changes,
		HistoryWindow:     result.History.Window.Days,
		RelevantWorkflows: workflowPaths,
	}
}

// explainResult calls f's configured provider when f.enabled, returning
// ("", nil) otherwise — no network call or subprocess is started unless
// --explain was actually passed.
func explainResult(ctx context.Context, f *explainFlags, result output.InspectResult) (string, error) {
	if !f.enabled {
		return "", nil
	}

	provider, err := newExplainProvider(f.provider, f.host, f.model)
	if err != nil {
		return "", err
	}
	return provider.Explain(ctx, findingFor(result))
}

// explainedJSON wraps a deterministic analysis with an optional AI
// explanation. Kept in cmd rather than internal/output so that package
// stays free of any dependency on internal/ai.
type explainedJSON struct {
	Analysis     output.InspectFullJSON `json:"analysis"`
	Explanation  string                 `json:"explanation,omitempty"`
	ExplainError string                 `json:"explainError,omitempty"`
}

// renderExplanation prints the AI explanation section (or a warning if
// it failed) to w. A failed explanation is never fatal: the
// deterministic analysis above it stands on its own.
func renderExplanation(w io.Writer, provider, explanation string, err error) {
	if explanation == "" && err == nil {
		return
	}
	fmt.Fprintln(w)
	fmt.Fprintf(w, "Explanation (%s)\n", provider)
	if err != nil {
		fmt.Fprintf(w, "  unavailable: %v\n", err)
		return
	}
	for _, line := range strings.Split(output.StripMarkdown(w, explanation), "\n") {
		fmt.Fprintf(w, "  %s\n", line)
	}
}
