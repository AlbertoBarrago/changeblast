package bitbucket_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/AlbertoBarrago/serval/internal/ci/bitbucket"
)

func TestDiscover(t *testing.T) {
	root := t.TempDir()

	config := `
image: atlassian/default-image:4

pipelines:
  default:
    - step:
        script:
          - echo default

  branches:
    main:
      - step:
          script:
            - echo main
    develop:
      - step:
          script:
            - echo develop

  pull-requests:
    '**':
      - step:
          script:
            - echo pr

  custom:
    deploy:
      - step:
          script:
            - echo deploy
`
	if err := os.WriteFile(filepath.Join(root, "bitbucket-pipelines.yml"), []byte(config), 0o644); err != nil {
		t.Fatal(err)
	}

	p := bitbucket.New()
	workflows, err := p.Discover(root)
	if err != nil {
		t.Fatalf("Discover: %v", err)
	}
	if len(workflows) != 5 {
		t.Fatalf("expected 5 workflows, got %d: %+v", len(workflows), workflows)
	}

	want := map[string]bool{
		"default":          false,
		"branches:main":    false,
		"branches:develop": false,
		"pull-requests:**": false,
		"custom:deploy":    false,
	}
	for _, wf := range workflows {
		if wf.Path != "bitbucket-pipelines.yml" {
			t.Errorf("unexpected path %q", wf.Path)
		}
		if len(wf.PathFilters) != 0 {
			t.Errorf("expected no path filters, got %v", wf.PathFilters)
		}
		if _, ok := want[wf.Name]; !ok {
			t.Errorf("unexpected workflow name %q", wf.Name)
		}
		want[wf.Name] = true
	}
	for name, seen := range want {
		if !seen {
			t.Errorf("missing expected workflow %q", name)
		}
	}
}

func TestDiscover_NoConfigFile(t *testing.T) {
	root := t.TempDir()

	p := bitbucket.New()
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

	if err := os.WriteFile(filepath.Join(root, "bitbucket-pipelines.yml"), []byte("not: valid: yaml: ["), 0o644); err != nil {
		t.Fatal(err)
	}

	p := bitbucket.New()
	workflows, err := p.Discover(root)
	if err != nil {
		t.Fatalf("Discover should not fail the scan on malformed config, got err: %v", err)
	}
	if workflows != nil {
		t.Fatalf("expected no workflows for malformed config, got %+v", workflows)
	}
}
