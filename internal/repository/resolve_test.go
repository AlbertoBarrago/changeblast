package repository_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/AlbertoBarrago/impactline/internal/repository"
)

func TestResolver_RelativeExtensionResolution(t *testing.T) {
	root := t.TempDir()
	writeFile(t, filepath.Join(root, "token.ts"), "export {}")
	from := filepath.Join(root, "middleware.ts")
	writeFile(t, from, "")

	r, err := repository.NewResolver(root)
	if err != nil {
		t.Fatalf("NewResolver: %v", err)
	}

	resolved, ok := r.Resolve(from, "./token")
	if !ok {
		t.Fatal("expected ./token to resolve to token.ts")
	}
	if resolved != filepath.Join(root, "token.ts") {
		t.Errorf("resolved = %q, want token.ts", resolved)
	}
}

func TestResolver_IndexResolution(t *testing.T) {
	root := t.TempDir()
	dir := filepath.Join(root, "utils")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	writeFile(t, filepath.Join(dir, "index.ts"), "export {}")
	from := filepath.Join(root, "app.ts")
	writeFile(t, from, "")

	r, err := repository.NewResolver(root)
	if err != nil {
		t.Fatalf("NewResolver: %v", err)
	}

	resolved, ok := r.Resolve(from, "./utils")
	if !ok {
		t.Fatal("expected ./utils to resolve to utils/index.ts")
	}
	if resolved != filepath.Join(dir, "index.ts") {
		t.Errorf("resolved = %q, want utils/index.ts", resolved)
	}
}

func TestResolver_BareSpecifierUnresolvedWithoutTSConfig(t *testing.T) {
	root := t.TempDir()
	from := filepath.Join(root, "app.ts")
	writeFile(t, from, "")

	r, err := repository.NewResolver(root)
	if err != nil {
		t.Fatalf("NewResolver: %v", err)
	}

	if _, ok := r.Resolve(from, "react"); ok {
		t.Error("expected bare specifier with no tsconfig to remain unresolved (external)")
	}
}

func TestResolver_MissingFileUnresolved(t *testing.T) {
	root := t.TempDir()
	from := filepath.Join(root, "app.ts")
	writeFile(t, from, "")

	r, err := repository.NewResolver(root)
	if err != nil {
		t.Fatalf("NewResolver: %v", err)
	}

	if _, ok := r.Resolve(from, "./does-not-exist"); ok {
		t.Error("expected a nonexistent relative import to remain unresolved")
	}
}
