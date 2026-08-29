// Package anthropic implements the ai.Provider contract against the
// Anthropic Messages API directly (bring-your-own-key), as an
// alternative to the `claude` local-CLI provider in internal/ai/localcli
// for users without the Claude Code CLI installed/authenticated.
package anthropic

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

	"github.com/AlbertoBarrago/serval/internal/ai"
)

// defaultAPIURL is the Anthropic Messages API endpoint.
const defaultAPIURL = "https://api.anthropic.com/v1/messages"

// anthropicVersion is the required API version header.
const anthropicVersion = "2023-06-01"

// requestTimeout bounds how long a single explanation request may take.
const requestTimeout = 60 * time.Second

// maxTokens caps the explanation response length; BuildExplainPrompt
// already asks for 3-5 sentences, so this is a safety bound, not a
// tuning knob.
const maxTokens = 512

// Provider calls the Anthropic Messages API directly over HTTPS.
type Provider struct {
	apiKey string
	model  string
	apiURL string
	client *http.Client
}

// New builds an Anthropic API provider. Reads ANTHROPIC_API_KEY from
// the environment — never from a flag or .serval.yml, since a
// committed/config-file secret is exactly what serval's security
// posture avoids. model is required: unlike the local Ollama provider's
// safe DefaultModel fallback, hardcoding a "current" hosted model id
// here would silently go stale as Anthropic ships new models.
func New(model string) (*Provider, error) {
	apiKey := os.Getenv("ANTHROPIC_API_KEY")
	if apiKey == "" {
		return nil, fmt.Errorf("anthropic: ANTHROPIC_API_KEY is not set")
	}
	if model == "" {
		return nil, fmt.Errorf("anthropic: --explain-model is required for --explain-provider=anthropic")
	}
	return newProvider(apiKey, model, defaultAPIURL), nil
}

// newProvider builds a Provider against an arbitrary apiURL, letting
// tests point it at an httptest.Server instead of the real API.
func newProvider(apiKey, model, apiURL string) *Provider {
	return &Provider{
		apiKey: apiKey,
		model:  model,
		apiURL: apiURL,
		client: &http.Client{Timeout: requestTimeout},
	}
}

// Name identifies this provider.
func (p *Provider) Name() string { return "anthropic" }

type messagesRequest struct {
	Model     string             `json:"model"`
	MaxTokens int                `json:"max_tokens"`
	Messages  []messagesReqEntry `json:"messages"`
}

type messagesReqEntry struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type messagesResponse struct {
	Content []struct {
		Text string `json:"text"`
	} `json:"content"`
	Error *struct {
		Message string `json:"message"`
	} `json:"error"`
}

// Explain sends finding as a structured prompt to the configured model
// and returns its natural-language response.
func (p *Provider) Explain(ctx context.Context, finding ai.Finding) (string, error) {
	payload, err := json.Marshal(messagesRequest{
		Model:     p.model,
		MaxTokens: maxTokens,
		Messages: []messagesReqEntry{
			{Role: "user", Content: ai.BuildExplainPrompt(finding)},
		},
	})
	if err != nil {
		return "", fmt.Errorf("anthropic: encoding request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, p.apiURL, bytes.NewReader(payload))
	if err != nil {
		return "", fmt.Errorf("anthropic: building request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("x-api-key", p.apiKey)
	req.Header.Set("anthropic-version", anthropicVersion)

	resp, err := p.client.Do(req)
	if err != nil {
		return "", fmt.Errorf("anthropic: request failed: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("anthropic: reading response: %w", err)
	}

	// Anthropic returns a JSON body with an "error" field on failure
	// (4xx/5xx), decoded before branching on status code so the error
	// message reaches the caller verbatim rather than a bare status line.
	var out messagesResponse
	decodeErr := json.Unmarshal(body, &out)

	if out.Error != nil {
		return "", fmt.Errorf("anthropic: %s (is model %q valid and ANTHROPIC_API_KEY correct?)", out.Error.Message, p.model)
	}
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("anthropic: %s returned %s: %s", p.apiURL, resp.Status, strings.TrimSpace(string(body)))
	}
	if decodeErr != nil {
		return "", fmt.Errorf("anthropic: decoding response: %w", decodeErr)
	}
	if len(out.Content) == 0 {
		return "", fmt.Errorf("anthropic: empty response")
	}

	return strings.TrimSpace(out.Content[0].Text), nil
}
