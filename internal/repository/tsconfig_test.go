package repository_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/AlbertoBarrago/serval/internal/repository"
)

func writeFile(t *testing.T, path, content string) {
	t.Helper()
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}

func TestFindTSConfig_NearestAncestor(t *testing.T) {
	root := t.TempDir()
	nested := filepath.Join(root, "src", "app")
	if err := os.MkdirAll(nested, 0o755); err != nil {
		t.Fatal(err)
	}

	writeFile(t, filepath.Join(root, "tsconfig.json"), `{"compilerOptions": {"baseUrl": "."}}`)

	cfg, err := repository.FindTSConfig(nested)
	if err != nil {
		t.Fatalf("FindTSConfig: %v", err)
	}
	if cfg == nil {
		t.Fatal("expected to find tsconfig.json in an ancestor directory")
	}
}

func TestFindTSConfig_NoneFound(t *testing.T) {
	root := t.TempDir()
	cfg, err := repository.FindTSConfig(root)
	if err != nil {
		t.Fatalf("FindTSConfig: %v", err)
	}
	if cfg != nil {
		t.Errorf("expected nil config, got %+v", cfg)
	}
}

func TestFindTSConfig_StripsJSONComments(t *testing.T) {
	root := t.TempDir()
	writeFile(t, filepath.Join(root, "tsconfig.json"), `{
  // line comment
  "compilerOptions": {
    "baseUrl": ".", /* block comment */
    "paths": { "@app/*": ["src/app/*"] }
  }
}`)

	cfg, err := repository.FindTSConfig(root)
	if err != nil {
		t.Fatalf("FindTSConfig: %v", err)
	}
	if cfg == nil {
		t.Fatal("expected config to parse despite comments")
	}
	if cfg.BaseURL != "." {
		t.Errorf("BaseURL = %q, want %q", cfg.BaseURL, ".")
	}
	if len(cfg.Paths["@app/*"]) != 1 || cfg.Paths["@app/*"][0] != "src/app/*" {
		t.Errorf("Paths[@app/*] = %v", cfg.Paths["@app/*"])
	}
}

func TestResolveAlias_WildcardPathMapping(t *testing.T) {
	root := t.TempDir()
	writeFile(t, filepath.Join(root, "tsconfig.json"), `{
  "compilerOptions": {
    "baseUrl": ".",
    "paths": { "@app/*": ["src/app/*"] }
  }
}`)

	cfg, err := repository.FindTSConfig(root)
	if err != nil || cfg == nil {
		t.Fatalf("FindTSConfig: cfg=%v err=%v", cfg, err)
	}

	resolved, ok := cfg.ResolveAlias("@app/token")
	if !ok {
		t.Fatal("expected @app/token to resolve via paths mapping")
	}
	want := filepath.Join(root, "src", "app", "token")
	if resolved != want {
		t.Errorf("ResolveAlias(@app/token) = %q, want %q", resolved, want)
	}
}

func TestResolveAlias_BaseURLOnly(t *testing.T) {
	root := t.TempDir()
	writeFile(t, filepath.Join(root, "tsconfig.json"), `{"compilerOptions": {"baseUrl": "src"}}`)

	cfg, err := repository.FindTSConfig(root)
	if err != nil || cfg == nil {
		t.Fatalf("FindTSConfig: cfg=%v err=%v", cfg, err)
	}

	resolved, ok := cfg.ResolveAlias("utils/format")
	if !ok {
		t.Fatal("expected non-relative specifier to resolve via baseUrl")
	}
	want := filepath.Join(root, "src", "utils", "format")
	if resolved != want {
		t.Errorf("ResolveAlias(utils/format) = %q, want %q", resolved, want)
	}

	if _, ok := cfg.ResolveAlias("./relative"); ok {
		t.Error("expected relative specifiers to be left unresolved by baseUrl fallback")
	}
}

func TestResolveAlias_NilConfig(t *testing.T) {
	var cfg *repository.TSConfig
	if _, ok := cfg.ResolveAlias("anything"); ok {
		t.Error("expected nil TSConfig to never resolve")
	}
}

func TestResolveAlias_NoMatch(t *testing.T) {
	root := t.TempDir()
	writeFile(t, filepath.Join(root, "tsconfig.json"), `{
  "compilerOptions": { "paths": { "@app/*": ["src/app/*"] } }
}`)
	cfg, _ := repository.FindTSConfig(root)

	if _, ok := cfg.ResolveAlias("react"); ok {
		t.Error("expected an unmatched bare specifier with no baseUrl to remain unresolved")
	}
}
