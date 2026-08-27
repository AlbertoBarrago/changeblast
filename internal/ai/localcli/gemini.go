package localcli

import (
	"context"

	"github.com/AlbertoBarrago/blast/internal/ai"
)

// GeminiProvider calls the local Gemini CLI (`gemini -p`) in
// non-interactive mode. Requires the `gemini` binary on PATH, already
// authenticated with a Google account — v0.1 never manages credentials
// for it.
type GeminiProvider struct {
	model string
}

// NewGemini builds a Gemini CLI provider. model is passed via `--model`
// when non-empty, otherwise the CLI's own default applies.
func NewGemini(model string) *GeminiProvider {
	return &GeminiProvider{model: model}
}

// Name identifies this provider.
func (p *GeminiProvider) Name() string { return "gemini" }

// Explain runs `gemini -p <prompt>` and returns its stdout.
func (p *GeminiProvider) Explain(ctx context.Context, finding ai.Finding) (string, error) {
	args := []string{"-p", ai.BuildExplainPrompt(finding)}
	if p.model != "" {
		args = append(args, "--model", p.model)
	}
	return run(ctx, "gemini", "gemini", args)
}
