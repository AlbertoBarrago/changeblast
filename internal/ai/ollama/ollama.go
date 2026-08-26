// Package ollama implements the ai.Provider contract against a local
// Ollama daemon (https://ollama.com), the default target for
// ChangeBlast's optional explanation layer specifically because it runs
// on the user's machine — consistent with "source code never leaves the
// local machine by default", the request here carries only the already
// computed, non-sensitive Finding summary, not source code.
package ollama

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/AlbertoBarrago/changeblast/internal/ai"
)

// DefaultHost is used when the OLLAMA_HOST environment variable is
// unset, matching Ollama's own default.
const DefaultHost = "http://localhost:11434"

// DefaultModel is used when no model is explicitly configured. It must
// already be pulled locally (`ollama pull <model>`); ChangeBlast never
// pulls a model on the user's behalf.
const DefaultModel = "llama3.2"

// requestTimeout bounds how long a single explanation request may take.
// Local generation can be slow on CPU-only machines, so this is
// deliberately generous rather than tuned for a snappy CLI.
const requestTimeout = 60 * time.Second

// Provider calls a local Ollama daemon's /api/generate endpoint.
type Provider struct {
	host   string
	model  string
	client *http.Client
}

// New builds an Ollama provider. host defaults to $OLLAMA_HOST or
// DefaultHost if empty; model defaults to DefaultModel if empty.
func New(host, model string) *Provider {
	if host == "" {
		host = os.Getenv("OLLAMA_HOST")
	}
	if host == "" {
		host = DefaultHost
	}
	if model == "" {
		model = DefaultModel
	}
	return &Provider{
		host:   strings.TrimRight(host, "/"),
		model:  model,
		client: &http.Client{Timeout: requestTimeout},
	}
}

// Name identifies this provider.
func (p *Provider) Name() string { return "ollama" }

type generateRequest struct {
	Model  string `json:"model"`
	Prompt string `json:"prompt"`
	Stream bool   `json:"stream"`
}

type generateResponse struct {
	Response string `json:"response"`
	Error    string `json:"error"`
}

// Explain sends finding as a structured prompt to the configured Ollama
// model and returns its natural-language response.
func (p *Provider) Explain(ctx context.Context, finding ai.Finding) (string, error) {
	payload, err := json.Marshal(generateRequest{
		Model:  p.model,
		Prompt: buildPrompt(finding),
		Stream: false,
	})
	if err != nil {
		return "", fmt.Errorf("ollama: encoding request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, p.host+"/api/generate", bytes.NewReader(payload))
	if err != nil {
		return "", fmt.Errorf("ollama: building request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := p.client.Do(req)
	if err != nil {
		return "", fmt.Errorf("ollama: request to %s failed (is Ollama running? try `ollama serve`): %w", p.host, err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("ollama: reading response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("ollama: %s returned %s: %s", p.host, resp.Status, strings.TrimSpace(string(body)))
	}

	var out generateResponse
	if err := json.Unmarshal(body, &out); err != nil {
		return "", fmt.Errorf("ollama: decoding response: %w", err)
	}
	if out.Error != "" {
		return "", fmt.Errorf("ollama: %s (is model %q pulled? try `ollama pull %s`)", out.Error, p.model, p.model)
	}

	return strings.TrimSpace(out.Response), nil
}

// buildPrompt turns a Finding into a prompt asking for explanation only
// — it never asks the model to produce or revise a score.
func buildPrompt(f ai.Finding) string {
	var b strings.Builder
	b.WriteString("You are explaining a static code-change risk analysis to a software engineer. ")
	b.WriteString("Do not invent a different risk score or contradict the one given. ")
	b.WriteString("In 3-5 sentences, explain why this file has this risk level and what the engineer should be careful about. ")
	b.WriteString("Be specific, reference the actual signals given, and avoid generic advice.\n\n")

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
