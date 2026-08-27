package localcli

import (
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/AlbertoBarrago/changeblast/internal/ai"
)

// CodexProvider calls the local Codex CLI (`codex exec`) in
// non-interactive mode. Requires the `codex` binary on PATH, already
// authenticated with a ChatGPT/OpenAI account — v0.1 never manages
// credentials for it.
type CodexProvider struct {
	model string
}

// NewCodex builds a Codex CLI provider. model is passed via `--model`
// when non-empty, otherwise the CLI's own default applies.
func NewCodex(model string) *CodexProvider {
	return &CodexProvider{model: model}
}

// Name identifies this provider.
func (p *CodexProvider) Name() string { return "codex" }

// Explain runs `codex exec <prompt>`, capturing the agent's final
// message via `--output-last-message <file>` rather than parsing
// stdout, since `codex exec`'s stdout also carries progress/log lines
// by default with no plain-text-only mode equivalent to Claude Code's
// `--output-format text`.
func (p *CodexProvider) Explain(ctx context.Context, finding ai.Finding) (string, error) {
	out, err := os.CreateTemp("", "blast-explain-codex-*.txt")
	if err != nil {
		return "", fmt.Errorf("codex: creating temp file: %w", err)
	}
	outPath := out.Name()
	out.Close()
	defer os.Remove(outPath)

	args := []string{"exec", "--skip-git-repo-check", "--output-last-message", outPath, ai.BuildExplainPrompt(finding)}
	if p.model != "" {
		args = append(args, "--model", p.model)
	}

	if _, err := run(ctx, "codex", "codex", args); err != nil {
		return "", err
	}

	content, err := os.ReadFile(outPath)
	if err != nil {
		return "", fmt.Errorf("codex: reading output: %w", err)
	}
	return strings.TrimSpace(string(content)), nil
}
