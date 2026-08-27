package repository

import (
	"os"
	"path/filepath"
	"testing"
)

func TestMatchPathPattern_ShortSpecifierNoPanic(t *testing.T) {
	// "@x/*/y" with a shorter specifier previously sliced out of range
	// and panicked; it must simply not match.
	matched, _ := matchPathPattern("@x/*/y", "@x/z")
	if matched {
		t.Error("pattern with longer fixed parts should not match a shorter specifier")
	}
}

func TestResolveAlias_DeterministicOverlappingPatterns(t *testing.T) {
	dir := t.TempDir()
	// The more specific pattern "@app/core/*" must win over "@app/*"
	// regardless of map iteration order.
	cfg := &TSConfig{
		Paths: map[string][]string{
			"@app/*":      {"./src/*"},
			"@app/core/*": {"./src/core/*"},
		},
		dir: dir,
	}

	for i := 0; i < 50; i++ {
		got, ok := cfg.ResolveAlias("@app/core/util")
		if !ok {
			t.Fatalf("iteration %d: expected alias resolution", i)
		}
		want := filepath.Join(dir, "src", "core", "util")
		if got != want {
			t.Fatalf("iteration %d: got %q, want %q", i, got, want)
		}
	}
}

func TestResolveAlias_TriesTargetsInOrder(t *testing.T) {
	dir := t.TempDir()
	if err := os.MkdirAll(filepath.Join(dir, "src", "b"), 0o755); err != nil {
		t.Fatal(err)
	}
	// First target does not exist on disk; resolver must fall back to
	// the second one, like TypeScript does.
	cfg := &TSConfig{
		Paths: map[string][]string{
			"@x/*": {"./src/a/*", "./src/b/*"},
		},
		dir: dir,
	}

	got, ok := cfg.ResolveAlias("@x/mod")
	if !ok {
		t.Fatal("expected alias resolution")
	}
	want := filepath.Join(dir, "src", "b", "mod")
	if got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}

func TestParseModulePath_TrailingComment(t *testing.T) {
	mod, err := parseModulePath([]byte("module example.com/m // my module\n"))
	if err != nil {
		t.Fatalf("parseModulePath: %v", err)
	}
	if mod != "example.com/m" {
		t.Errorf("got %q, want %q", mod, "example.com/m")
	}
}

func TestParseModulePath_PlainAndIndented(t *testing.T) {
	cases := map[string]string{
		"module example.com/a\n":                 "example.com/a",
		"\n// comment\n  module example.com/b\n": "example.com/b",
	}
	for in, want := range cases {
		got, err := parseModulePath([]byte(in))
		if err != nil {
			t.Fatalf("parseModulePath(%q): %v", in, err)
		}
		if got != want {
			t.Errorf("parseModulePath(%q) = %q, want %q", in, got, want)
		}
	}
}
