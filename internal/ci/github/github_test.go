package github_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/AlbertoBarrago/serval/internal/ci/github"
)

func TestDiscover(t *testing.T) {
	root := t.TempDir()
	dir := filepath.Join(root, ".github", "workflows")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}

	filtered := `
name: integration-auth
on:
  push:
    paths:
      - 'src/auth/**'
`
	unfiltered := `
name: lint
on: [push, pull_request]
`
	if err := os.WriteFile(filepath.Join(dir, "auth.yml"), []byte(filtered), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "lint.yml"), []byte(unfiltered), 0o644); err != nil {
		t.Fatal(err)
	}

	p := github.New()
	workflows, err := p.Discover(root)
	if err != nil {
		t.Fatalf("Discover: %v", err)
	}
	if len(workflows) != 2 {
		t.Fatalf("expected 2 workflows, got %d: %+v", len(workflows), workflows)
	}

	var auth, lint bool
	for _, wf := range workflows {
		switch wf.Name {
		case "integration-auth":
			auth = true
			if len(wf.PathFilters) != 1 || wf.PathFilters[0] != "src/auth/**" {
				t.Errorf("unexpected path filters for auth workflow: %+v", wf.PathFilters)
			}
		case "lint":
			lint = true
			if len(wf.PathFilters) != 0 {
				t.Errorf("expected lint workflow to have no path filters, got %+v", wf.PathFilters)
			}
		}
	}
	if !auth || !lint {
		t.Errorf("expected both workflows to be discovered, got %+v", workflows)
	}
}

func TestDiscover_NoWorkflowsDir(t *testing.T) {
	root := t.TempDir()
	p := github.New()
	workflows, err := p.Discover(root)
	if err != nil {
		t.Fatalf("Discover: %v", err)
	}
	if workflows != nil {
		t.Errorf("expected nil workflows, got %+v", workflows)
	}
}
