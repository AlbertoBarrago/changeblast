// Package gemini implements the ai.Provider contract against the
// Google Generative Language API directly (bring-your-own-key), as an
// alternative to the `gemini` local-CLI provider in internal/ai/localcli
// for users without the Gemini CLI installed/authenticated. It is
// registered under the distinct --explain-provider name "gemini-api" to
// avoid ambiguity with that local-CLI provider.
package gemini

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"

	"github.com/AlbertoBarrago/serval/internal/ai"
)

// defaultAPIBase is the Google Generative Language API base URL; the
// model and API key are appended per request.
const defaultAPIBase = "https://generativelanguage.googleapis.com/v1beta/models"

// requestTimeout bounds how long a single explanation request may take.
const requestTimeout = 60 * time.Second

// Provider calls the Google Generative Language API directly over HTTPS.
type Provider struct {
	apiKey  string
	model   string
	apiBase string
	client  *http.Client
}

// New builds a Gemini API provider. Reads GEMINI_API_KEY (falling back
// to GOOGLE_API_KEY) from the environment — never from a flag or
// .serval.yml, since a committed/config-file secret is exactly what
// serval's security posture avoids. model is required: unlike the local
// Ollama provider's safe DefaultModel fallback, hardcoding a "current"
// hosted model id here would silently go stale as Google ships new
// models.
func New(model string) (*Provider, error) {
	apiKey := os.Getenv("GEMINI_API_KEY")
	if apiKey == "" {
		apiKey = os.Getenv("GOOGLE_API_KEY")
	}
	if apiKey == "" {
		return nil, fmt.Errorf("gemini-api: GEMINI_API_KEY (or GOOGLE_API_KEY) is not set")
	}
	if model == "" {
		return nil, fmt.Errorf("gemini-api: --explain-model is required for --explain-provider=gemini-api")
	}
	return newProvider(apiKey, model, defaultAPIBase), nil
}

// newProvider builds a Provider against an arbitrary apiBase, letting
// tests point it at an httptest.Server instead of the real API.
func newProvider(apiKey, model, apiBase string) *Provider {
	return &Provider{
		apiKey:  apiKey,
		model:   model,
		apiBase: apiBase,
		client:  &http.Client{Timeout: requestTimeout},
	}
}

// Name identifies this provider.
func (p *Provider) Name() string { return "gemini-api" }

type generateContentRequest struct {
	Contents []content `json:"contents"`
}

type content struct {
	Parts []part `json:"parts"`
}

type part struct {
	Text string `json:"text"`
}

type generateContentResponse struct {
	Candidates []struct {
		Content content `json:"content"`
	} `json:"candidates"`
	Error *struct {
		Message string `json:"message"`
	} `json:"error"`
}

// Explain sends finding as a structured prompt to the configured model
// and returns its natural-language response.
func (p *Provider) Explain(ctx context.Context, finding ai.Finding) (string, error) {
	payload, err := json.Marshal(generateContentRequest{
		Contents: []content{{Parts: []part{{Text: ai.BuildExplainPrompt(finding)}}}},
	})
	if err != nil {
		return "", fmt.Errorf("gemini-api: encoding request: %w", err)
	}

	// safeEndpoint omits the API key so it never ends up in an error
	// message (printed to stdout/logs); the real request URL below
	// carries the key as Google's API requires.
	safeEndpoint := fmt.Sprintf("%s/%s:generateContent", p.apiBase, p.model)
	endpoint := safeEndpoint + "?key=" + url.QueryEscape(p.apiKey)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, bytes.NewReader(payload))
	if err != nil {
		return "", fmt.Errorf("gemini-api: building request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := p.client.Do(req)
	if err != nil {
		return "", fmt.Errorf("gemini-api: request failed: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("gemini-api: reading response: %w", err)
	}

	// Gemini returns a JSON body with an "error" field on failure
	// (4xx/5xx), decoded before branching on status code so the error
	// message reaches the caller verbatim rather than a bare status line.
	var out generateContentResponse
	decodeErr := json.Unmarshal(body, &out)

	if out.Error != nil {
		return "", fmt.Errorf("gemini-api: %s (is model %q valid and GEMINI_API_KEY correct?)", out.Error.Message, p.model)
	}
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("gemini-api: %s returned %s: %s", safeEndpoint, resp.Status, strings.TrimSpace(string(body)))
	}
	if decodeErr != nil {
		return "", fmt.Errorf("gemini-api: decoding response: %w", decodeErr)
	}
	if len(out.Candidates) == 0 || len(out.Candidates[0].Content.Parts) == 0 {
		return "", fmt.Errorf("gemini-api: empty response")
	}

	return strings.TrimSpace(out.Candidates[0].Content.Parts[0].Text), nil
}
