package anthropic

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/AlbertoBarrago/serval/internal/ai"
)

func TestExplain_Success(t *testing.T) {
	var gotBody map[string]interface{}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if got := r.Header.Get("x-api-key"); got != "test-key" {
			t.Errorf("x-api-key = %q, want test-key", got)
		}
		if r.Header.Get("anthropic-version") == "" {
			t.Error("expected anthropic-version header to be set")
		}
		if err := json.NewDecoder(r.Body).Decode(&gotBody); err != nil {
			t.Fatal(err)
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"content": []map[string]string{{"type": "text", "text": "  This file is risky because...  "}},
		})
	}))
	defer srv.Close()

	p := newProvider("test-key", "test-model", srv.URL)
	got, err := p.Explain(context.Background(), ai.Finding{
		Target:    "src/auth/token.ts",
		RiskLevel: "HIGH",
		RiskScore: 82,
	})
	if err != nil {
		t.Fatalf("Explain: %v", err)
	}
	if got != "This file is risky because..." {
		t.Errorf("Explain() = %q, want trimmed response", got)
	}
	if gotBody["model"] != "test-model" {
		t.Errorf("model = %v, want test-model", gotBody["model"])
	}
}

func TestExplain_APIError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusUnauthorized)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"type":  "error",
			"error": map[string]string{"type": "authentication_error", "message": "invalid x-api-key"},
		})
	}))
	defer srv.Close()

	p := newProvider("bad-key", "test-model", srv.URL)
	_, err := p.Explain(context.Background(), ai.Finding{})
	if err == nil {
		t.Fatal("expected an error")
	}
	if !strings.Contains(err.Error(), "invalid x-api-key") {
		t.Errorf("expected error to surface API message, got: %v", err)
	}
}

func TestExplain_ConnectionRefused(t *testing.T) {
	p := newProvider("key", "model", "http://127.0.0.1:1")
	_, err := p.Explain(context.Background(), ai.Finding{})
	if err == nil {
		t.Fatal("expected an error when the API is unreachable")
	}
}

func TestNew_MissingAPIKey(t *testing.T) {
	t.Setenv("ANTHROPIC_API_KEY", "")
	if _, err := New("test-model"); err == nil {
		t.Fatal("expected an error when ANTHROPIC_API_KEY is unset")
	}
}

func TestNew_MissingModel(t *testing.T) {
	t.Setenv("ANTHROPIC_API_KEY", "test-key")
	if _, err := New(""); err == nil {
		t.Fatal("expected an error when model is empty")
	}
}

func TestNew_Success(t *testing.T) {
	t.Setenv("ANTHROPIC_API_KEY", "test-key")
	p, err := New("test-model")
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	if p.Name() != "anthropic" {
		t.Errorf("Name() = %q, want anthropic", p.Name())
	}
}
