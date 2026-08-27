package azure_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/AlbertoBarrago/blast/internal/ci/azure"
)

func TestDiscover_FilteredTriggerAndPR(t *testing.T) {
	root := t.TempDir()
	config := `
name: auth-pipeline
trigger:
  branches:
    include:
      - main
  paths:
    include:
      - src/auth/*
pr:
  paths:
    include:
      - src/auth/*
`
	if err := os.WriteFile(filepath.Join(root, "azure-pipelines.yml"), []byte(config), 0o644); err != nil {
		t.Fatal(err)
	}

	p := azure.New()
	workflows, err := p.Discover(root)
	if err != nil {
		t.Fatalf("Discover: %v", err)
	}
	if len(workflows) != 1 {
		t.Fatalf("expected 1 workflow, got %d: %+v", len(workflows), workflows)
	}
	wf := workflows[0]
	if wf.Name != "auth-pipeline" {
		t.Errorf("Name = %q, want auth-pipeline", wf.Name)
	}
	if len(wf.PathFilters) != 2 || wf.PathFilters[0] != "src/auth/*" {
		t.Errorf("unexpected path filters: %+v", wf.PathFilters)
	}
}

func TestDiscover_NoTriggerIsUnfiltered(t *testing.T) {
	root := t.TempDir()
	config := `
steps:
  - script: echo hello
`
	if err := os.WriteFile(filepath.Join(root, "azure-pipelines.yml"), []byte(config), 0o644); err != nil {
		t.Fatal(err)
	}

	p := azure.New()
	workflows, err := p.Discover(root)
	if err != nil {
		t.Fatalf("Discover: %v", err)
	}
	if len(workflows) != 1 {
		t.Fatalf("expected 1 workflow, got %d: %+v", len(workflows), workflows)
	}
	if len(workflows[0].PathFilters) != 0 {
		t.Errorf("expected unfiltered (no trigger declared), got %+v", workflows[0].PathFilters)
	}
}

func TestDiscover_PRNoneWithFilteredTrigger(t *testing.T) {
	root := t.TempDir()
	config := `
trigger:
  paths:
    include:
      - src/**
pr: none
`
	if err := os.WriteFile(filepath.Join(root, "azure-pipelines.yml"), []byte(config), 0o644); err != nil {
		t.Fatal(err)
	}

	p := azure.New()
	workflows, err := p.Discover(root)
	if err != nil {
		t.Fatalf("Discover: %v", err)
	}
	if len(workflows) != 1 || len(workflows[0].PathFilters) != 1 || workflows[0].PathFilters[0] != "src/**" {
		t.Errorf("expected pr:none to be ignored and trigger filter to stand, got %+v", workflows)
	}
}

func TestDiscover_NoConfigFile(t *testing.T) {
	root := t.TempDir()
	p := azure.New()
	workflows, err := p.Discover(root)
	if err != nil {
		t.Fatalf("Discover: %v", err)
	}
	if workflows != nil {
		t.Errorf("expected nil workflows, got %+v", workflows)
	}
}
