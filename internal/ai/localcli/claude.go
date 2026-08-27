package localcli

import (
	"context"

	"github.com/AlbertoBarrago/impactline/internal/ai"
)

// ClaudeProvider calls the local Claude Code CLI (`claude -p`) in
// headless print mode. Requires the `claude` binary on PATH, already
// authenticated (`claude login` or an active subscription) — v0.1
// never manages credentials for it.
type ClaudeProvider struct {
	model string
}

// NewClaude builds a Claude Code CLI provider. model is passed via
// `--model` when non-empty, otherwise the CLI's own default applies.
func NewClaude(model string) *ClaudeProvider {
	return &ClaudeProvider{model: model}
}

// Name identifies this provider.
func (p *ClaudeProvider) Name() string { return "claude" }

// Explain runs `claude -p <prompt> --output-format text` and returns
// its stdout.
func (p *ClaudeProvider) Explain(ctx context.Context, finding ai.Finding) (string, error) {
	args := []string{"-p", ai.BuildExplainPrompt(finding), "--output-format", "text"}
	if p.model != "" {
		args = append(args, "--model", p.model)
	}
	return run(ctx, "claude", "claude", args)
}
