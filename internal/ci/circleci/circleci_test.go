package circleci_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/AlbertoBarrago/serval/internal/ci/circleci"
)

func TestDiscover(t *testing.T) {
	root := t.TempDir()

	config := `
version: 2.1

jobs:
  build:
    docker:
      - image: cimg/base:stable
    steps:
      - checkout
  test:
    docker:
      - image: cimg/base:stable
    steps:
      - checkout

workflows:
  version: 2
  build-and-test:
    jobs:
      - build
      - test
  nightly:
    jobs:
      - test
`
	dir := filepath.Join(root, ".circleci")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "config.yml"), []byte(config), 0o644); err != nil {
		t.Fatal(err)
	}

	p := circleci.New()
	workflows, err := p.Discover(root)
	if err != nil {
		t.Fatalf("Discover: %v", err)
	}
	if len(workflows) != 2 {
		t.Fatalf("expected 2 workflows, got %d: %+v", len(workflows), workflows)
	}

	var sawBuildAndTest, sawNightly bool
	for _, wf := range workflows {
		if wf.Path != ".circleci/config.yml" {
			t.Errorf("unexpected path %q", wf.Path)
		}
		if len(wf.PathFilters) != 0 {
			t.Errorf("expected no path filters, got %v", wf.PathFilters)
		}
		switch wf.Name {
		case "build-and-test":
			sawBuildAndTest = true
		case "nightly":
			sawNightly = true
		}
	}
	if !sawBuildAndTest || !sawNightly {
		t.Fatalf("missing expected workflows: %+v", workflows)
	}
}

func TestDiscover_NoWorkflowsFallsBackToDefault(t *testing.T) {
	root := t.TempDir()

	config := `
version: 2
jobs:
  build:
    docker:
      - image: cimg/base:stable
    steps:
      - checkout
`
	dir := filepath.Join(root, ".circleci")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "config.yml"), []byte(config), 0o644); err != nil {
		t.Fatal(err)
	}

	p := circleci.New()
	workflows, err := p.Discover(root)
	if err != nil {
		t.Fatalf("Discover: %v", err)
	}
	if len(workflows) != 1 || workflows[0].Name != "default" {
		t.Fatalf("expected single \"default\" workflow, got %+v", workflows)
	}
}

func TestDiscover_NoConfigFile(t *testing.T) {
	root := t.TempDir()

	p := circleci.New()
	workflows, err := p.Discover(root)
	if err != nil {
		t.Fatalf("Discover: %v", err)
	}
	if workflows != nil {
		t.Fatalf("expected no workflows, got %+v", workflows)
	}
}

func TestDiscover_MalformedConfig(t *testing.T) {
	root := t.TempDir()

	dir := filepath.Join(root, ".circleci")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "config.yml"), []byte("not: valid: yaml: ["), 0o644); err != nil {
		t.Fatal(err)
	}

	p := circleci.New()
	workflows, err := p.Discover(root)
	if err != nil {
		t.Fatalf("Discover should not fail the scan on malformed config, got err: %v", err)
	}
	if workflows != nil {
		t.Fatalf("expected no workflows for malformed config, got %+v", workflows)
	}
}
