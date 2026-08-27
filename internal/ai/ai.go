// Package ai defines the provider-agnostic contract for the optional
// explanation layer (`impactline inspect --explain`). A Provider only ever
// turns already-computed deterministic findings into explanatory prose
// — it has no way to influence the risk score itself, which keeps
// Impactline's "deterministic by default" guarantee intact even with
// AI explanation enabled.
//
// No Provider is ever called unless the user explicitly passes
// --explain: this package makes no network calls on its own.
package ai

import (
	"context"
	"fmt"
	"strings"
)

// Finding is the read-only summary of a deterministic analysis result
// that a Provider may explain. It intentionally carries no method that
// could be used to alter the underlying risk score.
type Finding struct {
	Target            string   `json:"target"`
	DirectImpact      []string `json:"directImpact"`
	IndirectImpact    []string `json:"indirectImpact"`
	RiskLevel         string   `json:"riskLevel"`
	RiskScore         int      `json:"riskScore"`
	RiskBreakdown     []string `json:"riskBreakdown"`
	HistoryChanges    int      `json:"historyChanges"`
	HistoryWindow     int      `json:"historyWindowDays"`
	RelevantWorkflows []string `json:"relevantWorkflows"`
}

// Provider explains a Finding in natural language. Implementations must
// not perform any action beyond producing text — they receive no way to
// write back into the analysis.
type Provider interface {
	// Name identifies the provider (e.g. "ollama").
	Name() string
	// Explain returns a short natural-language explanation of finding.
	Explain(ctx context.Context, finding Finding) (string, error)
}

// BuildExplainPrompt turns a Finding into a prompt asking for
// explanation only — every Provider uses this same prompt, so the
// instructions a model receives ("don't invent a score", "plain prose
// only") stay in one place rather than drifting between
// implementations. It never asks the model to produce or revise a
// score.
func BuildExplainPrompt(f Finding) string {
	var b strings.Builder
	b.WriteString("You are explaining a static code-change risk analysis to a software engineer. ")
	b.WriteString("Do not invent a different risk score or contradict the one given. ")
	b.WriteString("In 3-5 sentences, explain why this file has this risk level and what the engineer should be careful about. ")
	b.WriteString("Be specific, reference the actual signals given, and avoid generic advice. ")
	b.WriteString("Write plain prose only: no markdown formatting, no **bold**, no bullet points, no headers, no backticks.\n\n")

	fmt.Fprintf(&b, "Target file: %s\n", f.Target)
	fmt.Fprintf(&b, "Risk: %s (%d/100)\n", f.RiskLevel, f.RiskScore)
	if len(f.RiskBreakdown) > 0 {
		b.WriteString("Risk breakdown:\n")
		for _, r := range f.RiskBreakdown {
			fmt.Fprintf(&b, "- %s\n", r)
		}
	}
	fmt.Fprintf(&b, "Direct dependents: %d\n", len(f.DirectImpact))
	fmt.Fprintf(&b, "Indirect dependents: %d\n", len(f.IndirectImpact))
	fmt.Fprintf(&b, "Git changes in the last %d days: %d\n", f.HistoryWindow, f.HistoryChanges)
	if len(f.RelevantWorkflows) > 0 {
		fmt.Fprintf(&b, "Relevant CI workflows: %s\n", strings.Join(f.RelevantWorkflows, ", "))
	}

	return b.String()
}
