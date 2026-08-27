// Package ollama implements the ai.Provider contract against a local
// Ollama daemon (https://ollama.com), the default target for
// Blast's optional explanation layer specifically because it runs
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

	"github.com/AlbertoBarrago/blast/internal/ai"
)

// DefaultHost is used when the OLLAMA_HOST environment variable is
// unset, matching Ollama's own default.
const DefaultHost = "http://localhost:11434"

// DefaultModel is used when no model is explicitly configured. It must
// already be pulled locally (`ollama pull <model>`); Blast never
// pulls a model on the user's behalf.
const DefaultModel = "llama3.2"

// requestTimeout bounds how long a single explanation request may take.
// Local generation can be slow on CPU-only machines, so this is
// deliberately generous rather than tuned for a snappy CLI.
const requestTimeout = 60 * time.Second

// Provider calls a local Ollama daemon's /api/generate endpoint.
type Provider struct {
	host  string
	model string
	// modelExplicit is true when the caller passed a model explicitly
	// (e.g. --explain-model), as opposed to falling back to
	// DefaultModel. It gates the auto-fallback in resolveModel: an
	// explicit choice is trusted as-is and never silently swapped for
	// another model, but the *default* is allowed to fall back to
	// whatever the user actually has pulled.
	modelExplicit bool
	client        *http.Client
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
	explicit := model != ""
	if model == "" {
		model = DefaultModel
	}
	return &Provider{
		host:          strings.TrimRight(host, "/"),
		model:         model,
		modelExplicit: explicit,
		client:        &http.Client{Timeout: requestTimeout},
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

// ListModels returns the names of models currently pulled into the
// Ollama daemon, via /api/tags.
func (p *Provider) ListModels(ctx context.Context) ([]string, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, p.host+"/api/tags", nil)
	if err != nil {
		return nil, fmt.Errorf("ollama: building request: %w", err)
	}

	resp, err := p.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("ollama: request to %s failed: %w", p.host, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("ollama: %s returned %s", p.host, resp.Status)
	}

	var out struct {
		Models []struct {
			Name string `json:"name"`
		} `json:"models"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		return nil, fmt.Errorf("ollama: decoding response: %w", err)
	}

	names := make([]string, len(out.Models))
	for i, m := range out.Models {
		names[i] = m.Name
	}
	return names, nil
}

// resolveModel picks which model an Explain call should actually use.
// An explicitly configured model (--explain-model) is always trusted
// as-is. Otherwise, DefaultModel is used if it's pulled; if not, the
// first model the daemon reports is used instead, so a fresh install
// with only non-default models pulled still works without requiring
// --explain-model on every call. If the model list can't be fetched for
// any reason, it falls through to DefaultModel and lets Explain's own
// request surface the real error.
func (p *Provider) resolveModel(ctx context.Context) string {
	if p.modelExplicit {
		return p.model
	}

	models, err := p.ListModels(ctx)
	if err != nil || len(models) == 0 {
		return p.model
	}
	for _, m := range models {
		if m == p.model {
			return p.model
		}
	}
	return models[0]
}

// Explain sends finding as a structured prompt to the configured Ollama
// model and returns its natural-language response.
func (p *Provider) Explain(ctx context.Context, finding ai.Finding) (string, error) {
	model := p.resolveModel(ctx)

	payload, err := json.Marshal(generateRequest{
		Model:  model,
		Prompt: ai.BuildExplainPrompt(finding),
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

	// Ollama returns a JSON body with an "error" field on failure too
	// (e.g. HTTP 404 for an unpulled model), not just on HTTP 200, so
	// this is decoded before branching on status code — otherwise the
	// "try `ollama pull`" hint would only ever fire for the (rarer)
	// case of a 200 response carrying an error field.
	var out generateResponse
	decodeErr := json.Unmarshal(body, &out)

	if out.Error != "" {
		return "", fmt.Errorf("ollama: %s (is model %q pulled? try `ollama pull %s`)", out.Error, model, model)
	}
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("ollama: %s returned %s: %s", p.host, resp.Status, strings.TrimSpace(string(body)))
	}
	if decodeErr != nil {
		return "", fmt.Errorf("ollama: decoding response: %w", decodeErr)
	}

	return strings.TrimSpace(out.Response), nil
}
