package gemini

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
	var gotPath string
	var gotBody map[string]interface{}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		if got := r.URL.Query().Get("key"); got != "test-key" {
			t.Errorf("key query param = %q, want test-key", got)
		}
		if err := json.NewDecoder(r.Body).Decode(&gotBody); err != nil {
			t.Fatal(err)
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"candidates": []map[string]interface{}{
				{"content": map[string]interface{}{
					"parts": []map[string]string{{"text": "  This file is risky because...  "}},
				}},
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
	if gotPath != "/test-model:generateContent" {
		t.Errorf("path = %q, want /test-model:generateContent", gotPath)
	}
}

func TestExplain_APIError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"error": map[string]string{"message": "API key not valid"},
		})
	}))
	defer srv.Close()

	p := newProvider("bad-key", "test-model", srv.URL)
	_, err := p.Explain(context.Background(), ai.Finding{})
	if err == nil {
		t.Fatal("expected an error")
	}
	if !strings.Contains(err.Error(), "API key not valid") {
		t.Errorf("expected error to surface API message, got: %v", err)
	}
	if strings.Contains(err.Error(), "bad-key") {
		t.Errorf("expected the API key to never appear in an error message, got: %v", err)
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
	t.Setenv("GEMINI_API_KEY", "")
	t.Setenv("GOOGLE_API_KEY", "")
	if _, err := New("test-model"); err == nil {
		t.Fatal("expected an error when no API key is set")
	}
}

func TestNew_FallsBackToGoogleAPIKey(t *testing.T) {
	t.Setenv("GEMINI_API_KEY", "")
	t.Setenv("GOOGLE_API_KEY", "test-key")
	if _, err := New("test-model"); err != nil {
		t.Fatalf("New: %v", err)
	}
}

func TestNew_MissingModel(t *testing.T) {
	t.Setenv("GEMINI_API_KEY", "test-key")
	if _, err := New(""); err == nil {
		t.Fatal("expected an error when model is empty")
	}
}

func TestNew_Success(t *testing.T) {
	t.Setenv("GEMINI_API_KEY", "test-key")
	p, err := New("test-model")
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	if p.Name() != "gemini-api" {
		t.Errorf("Name() = %q, want gemini-api", p.Name())
	}
}
