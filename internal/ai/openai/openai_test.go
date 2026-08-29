package openai

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
		if got := r.Header.Get("Authorization"); got != "Bearer test-key" {
			t.Errorf("Authorization = %q, want Bearer test-key", got)
		}
		if err := json.NewDecoder(r.Body).Decode(&gotBody); err != nil {
			t.Fatal(err)
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"choices": []map[string]interface{}{
				{"message": map[string]string{"role": "assistant", "content": "  This file is risky because...  "}},
			},
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
			"error": map[string]string{"message": "Incorrect API key provided"},
		})
	}))
	defer srv.Close()

	p := newProvider("bad-key", "test-model", srv.URL)
	_, err := p.Explain(context.Background(), ai.Finding{})
	if err == nil {
		t.Fatal("expected an error")
	}
	if !strings.Contains(err.Error(), "Incorrect API key") {
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
	t.Setenv("OPENAI_API_KEY", "")
	if _, err := New("test-model"); err == nil {
		t.Fatal("expected an error when OPENAI_API_KEY is unset")
	}
}

func TestNew_MissingModel(t *testing.T) {
	t.Setenv("OPENAI_API_KEY", "test-key")
	if _, err := New(""); err == nil {
		t.Fatal("expected an error when model is empty")
	}
}

func TestNew_Success(t *testing.T) {
	t.Setenv("OPENAI_API_KEY", "test-key")
	p, err := New("test-model")
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	if p.Name() != "openai" {
		t.Errorf("Name() = %q, want openai", p.Name())
	}
}
