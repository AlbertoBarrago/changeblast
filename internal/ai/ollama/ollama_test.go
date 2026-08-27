package ollama_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/AlbertoBarrago/blast/internal/ai"
	"github.com/AlbertoBarrago/blast/internal/ai/ollama"
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

func TestExplain_404WithErrorBody(t *testing.T) {
	// Ollama's real behavior for an unpulled model: HTTP 404 with a JSON
	// body carrying "error", not a 200 response. The "try `ollama pull`"
	// hint must still fire in this case, not just for a 200-with-error
	// response.
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusNotFound)
		json.NewEncoder(w).Encode(map[string]string{"error": "model 'llama3.2' not found"})
	}))
	defer srv.Close()

	p := ollama.New(srv.URL, "llama3.2")
	_, err := p.Explain(context.Background(), ai.Finding{})
	if err == nil {
		t.Fatal("expected an error")
	}
	if !strings.Contains(err.Error(), "ollama pull llama3.2") {
		t.Errorf("expected the pull hint in the error, got: %v", err)
	}
}

func TestExplain_FallsBackToPulledModel(t *testing.T) {
	var gotModel string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch r.URL.Path {
		case "/api/tags":
			json.NewEncoder(w).Encode(map[string]interface{}{
				"models": []map[string]string{{"name": "qwen3:8b"}},
			})
		case "/api/generate":
			var body map[string]interface{}
			json.NewDecoder(r.Body).Decode(&body)
			gotModel, _ = body["model"].(string)
			json.NewEncoder(w).Encode(map[string]string{"response": "ok"})
		}
	}))
	defer srv.Close()

	// No explicit model: DefaultModel ("llama3.2") isn't in the pulled
	// list, so it should fall back to the one model that is.
	p := ollama.New(srv.URL, "")
	if _, err := p.Explain(context.Background(), ai.Finding{}); err != nil {
		t.Fatalf("Explain: %v", err)
	}
	if gotModel != "qwen3:8b" {
		t.Errorf("model sent = %q, want fallback to qwen3:8b", gotModel)
	}
}

func TestExplain_ExplicitModelNeverFallsBack(t *testing.T) {
	var gotModel string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch r.URL.Path {
		case "/api/tags":
			json.NewEncoder(w).Encode(map[string]interface{}{
				"models": []map[string]string{{"name": "qwen3:8b"}},
			})
		case "/api/generate":
			var body map[string]interface{}
			json.NewDecoder(r.Body).Decode(&body)
			gotModel, _ = body["model"].(string)
			json.NewEncoder(w).Encode(map[string]string{"response": "ok"})
		}
	}))
	defer srv.Close()

	// Explicit --explain-model must never be silently swapped, even if
	// it isn't in the pulled list (that's what surfaces as a clear
	// "not found" error instead of silent substitution).
	p := ollama.New(srv.URL, "custom-model")
	if _, err := p.Explain(context.Background(), ai.Finding{}); err != nil {
		t.Fatalf("Explain: %v", err)
	}
	if gotModel != "custom-model" {
		t.Errorf("model sent = %q, want explicit custom-model preserved", gotModel)
	}
}

func TestListModels(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/tags" {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"models": []map[string]string{{"name": "a"}, {"name": "b"}},
		})
	}))
	defer srv.Close()

	got, err := ollama.New(srv.URL, "").ListModels(context.Background())
	if err != nil {
		t.Fatalf("ListModels: %v", err)
	}
	if len(got) != 2 || got[0] != "a" || got[1] != "b" {
		t.Errorf("ListModels() = %v, want [a b]", got)
	}
}

func TestName(t *testing.T) {
	if got := ollama.New("", "").Name(); got != "ollama" {
		t.Errorf("Name() = %q, want ollama", got)
	}
}
