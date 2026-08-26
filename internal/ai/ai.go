// Package ai defines the provider-agnostic contract for the optional
// explanation layer (`blast inspect --explain`). A Provider only ever
// turns already-computed deterministic findings into explanatory prose
// — it has no way to influence the risk score itself, which keeps
// ChangeBlast's "deterministic by default" guarantee intact even with
// AI explanation enabled.
//
// No Provider is ever called unless the user explicitly passes
// --explain: this package makes no network calls on its own.
package ai

import "context"

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
