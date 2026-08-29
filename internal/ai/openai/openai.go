// Package openai implements the ai.Provider contract against the
// OpenAI Chat Completions API directly (bring-your-own-key), as an
// alternative to the `codex` local-CLI provider in internal/ai/localcli
// for users without the Codex CLI installed/authenticated.
package openai

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

// defaultAPIURL is the OpenAI Chat Completions API endpoint.
const defaultAPIURL = "https://api.openai.com/v1/chat/completions"

// requestTimeout bounds how long a single explanation request may take.
const requestTimeout = 60 * time.Second

// Provider calls the OpenAI Chat Completions API directly over HTTPS.
type Provider struct {
	apiKey string
	model  string
	apiURL string
	client *http.Client
}

// New builds an OpenAI API provider. Reads OPENAI_API_KEY from the
// environment — never from a flag or .serval.yml, since a
// committed/config-file secret is exactly what serval's security
// posture avoids. model is required: unlike the local Ollama provider's
// safe DefaultModel fallback, hardcoding a "current" hosted model id
// here would silently go stale as OpenAI ships new models.
func New(model string) (*Provider, error) {
	apiKey := os.Getenv("OPENAI_API_KEY")
	if apiKey == "" {
		return nil, fmt.Errorf("openai: OPENAI_API_KEY is not set")
	}
	if model == "" {
		return nil, fmt.Errorf("openai: --explain-model is required for --explain-provider=openai")
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
func (p *Provider) Name() string { return "openai" }

type chatRequest struct {
	Model    string        `json:"model"`
	Messages []chatMessage `json:"messages"`
}

type chatMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type chatResponse struct {
	Choices []struct {
		Message chatMessage `json:"message"`
	} `json:"choices"`
	Error *struct {
		Message string `json:"message"`
	} `json:"error"`
}

// Explain sends finding as a structured prompt to the configured model
// and returns its natural-language response.
func (p *Provider) Explain(ctx context.Context, finding ai.Finding) (string, error) {
	payload, err := json.Marshal(chatRequest{
		Model: p.model,
		Messages: []chatMessage{
			{Role: "user", Content: ai.BuildExplainPrompt(finding)},
		},
	})
	if err != nil {
		return "", fmt.Errorf("openai: encoding request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, p.apiURL, bytes.NewReader(payload))
	if err != nil {
		return "", fmt.Errorf("openai: building request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+p.apiKey)

	resp, err := p.client.Do(req)
	if err != nil {
		return "", fmt.Errorf("openai: request failed: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("openai: reading response: %w", err)
	}

	// OpenAI returns a JSON body with an "error" field on failure
	// (4xx/5xx), decoded before branching on status code so the error
	// message reaches the caller verbatim rather than a bare status line.
	var out chatResponse
	decodeErr := json.Unmarshal(body, &out)

	if out.Error != nil {
		return "", fmt.Errorf("openai: %s (is model %q valid and OPENAI_API_KEY correct?)", out.Error.Message, p.model)
	}
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("openai: %s returned %s: %s", p.apiURL, resp.Status, strings.TrimSpace(string(body)))
	}
	if decodeErr != nil {
		return "", fmt.Errorf("openai: decoding response: %w", decodeErr)
	}
	if len(out.Choices) == 0 {
		return "", fmt.Errorf("openai: empty response")
	}

	return strings.TrimSpace(out.Choices[0].Message.Content), nil
}
