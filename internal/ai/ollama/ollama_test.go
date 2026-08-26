package ollama_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/AlbertoBarrago/changeblast/internal/ai"
	"github.com/AlbertoBarrago/changeblast/internal/ai/ollama"
)

func TestExplain_Success(t *testing.T) {
	var gotBody map[string]interface{}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/generate" {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		if err := json.NewDecoder(r.Body).Decode(&gotBody); err != nil {
			t.Fatal(err)
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]string{"response": "  This file is risky because...  "})
	}))
	defer srv.Close()

	p := ollama.New(srv.URL, "test-model")
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
	if gotBody["stream"] != false {
		t.Errorf("stream = %v, want false", gotBody["stream"])
	}
	prompt, _ := gotBody["prompt"].(string)
	if !strings.Contains(prompt, "src/auth/token.ts") || !strings.Contains(prompt, "HIGH") {
		t.Errorf("prompt missing expected content: %q", prompt)
	}
}

func TestExplain_OllamaError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]string{"error": "model \"x\" not found"})
	}))
	defer srv.Close()

	p := ollama.New(srv.URL, "x")
	_, err := p.Explain(context.Background(), ai.Finding{})
	if err == nil {
		t.Fatal("expected an error when Ollama reports one")
	}
	if !strings.Contains(err.Error(), "not found") {
		t.Errorf("expected error to surface Ollama's message, got: %v", err)
	}
}

func TestExplain_ConnectionRefused(t *testing.T) {
	p := ollama.New("http://127.0.0.1:1", "any")
	_, err := p.Explain(context.Background(), ai.Finding{})
	if err == nil {
		t.Fatal("expected an error when Ollama is unreachable")
	}
}

func TestName(t *testing.T) {
	if got := ollama.New("", "").Name(); got != "ollama" {
		t.Errorf("Name() = %q, want ollama", got)
	}
}
