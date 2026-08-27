package gitlab_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/AlbertoBarrago/changeblast/internal/ci/gitlab"
)

func TestDiscover(t *testing.T) {
	root := t.TempDir()

	config := `
stages:
  - test

variables:
  FOO: bar

.template:
  script:
    - echo hidden

auth-test:
  stage: test
  rules:
    - changes:
        - src/auth/**
  script:
    - echo auth

lint:
  stage: test
  script:
    - echo lint

api-test:
  stage: test
  only:
    changes:
      - src/api/*.ts
  script:
    - echo api
`
	if err := os.WriteFile(filepath.Join(root, ".gitlab-ci.yml"), []byte(config), 0o644); err != nil {
		t.Fatal(err)
	}

	p := gitlab.New()
	workflows, err := p.Discover(root)
	if err != nil {
		t.Fatalf("Discover: %v", err)
	}
	if len(workflows) != 3 {
		t.Fatalf("expected 3 workflows (template excluded), got %d: %+v", len(workflows), workflows)
	}

	var authTest, lint, apiTest bool
	for _, wf := range workflows {
		switch wf.Name {
		case "auth-test":
			authTest = true
			if len(wf.PathFilters) != 1 || wf.PathFilters[0] != "src/auth/**" {
				t.Errorf("unexpected path filters for auth-test: %+v", wf.PathFilters)
			}
		case "lint":
			lint = true
			if len(wf.PathFilters) != 0 {
				t.Errorf("expected lint to have no path filters (no rules/only), got %+v", wf.PathFilters)
			}
		case "api-test":
			apiTest = true
			if len(wf.PathFilters) != 1 || wf.PathFilters[0] != "src/api/*.ts" {
				t.Errorf("unexpected path filters for api-test: %+v", wf.PathFilters)
			}
		case "template":
			t.Errorf("expected hidden .template job to be excluded")
		}
	}
	if !authTest || !lint || !apiTest {
		t.Errorf("expected all 3 real jobs to be discovered, got %+v", workflows)
	}
}

func TestDiscover_RuleWithoutChangesIsUnfiltered(t *testing.T) {
	root := t.TempDir()

	config := `
deploy:
  rules:
    - if: '$CI_COMMIT_BRANCH == "main"'
    - changes:
        - src/**
  script:
    - echo deploy
`
	if err := os.WriteFile(filepath.Join(root, ".gitlab-ci.yml"), []byte(config), 0o644); err != nil {
		t.Fatal(err)
	}

	p := gitlab.New()
	workflows, err := p.Discover(root)
	if err != nil {
		t.Fatalf("Discover: %v", err)
	}
	if len(workflows) != 1 {
		t.Fatalf("expected 1 workflow, got %d: %+v", len(workflows), workflows)
	}
	if len(workflows[0].PathFilters) != 0 {
		t.Errorf("expected unfiltered (one rule has no changes:), got %+v", workflows[0].PathFilters)
	}
}

func TestDiscover_NoConfigFile(t *testing.T) {
	root := t.TempDir()
	p := gitlab.New()
	workflows, err := p.Discover(root)
	if err != nil {
		t.Fatalf("Discover: %v", err)
	}
	if workflows != nil {
		t.Errorf("expected nil workflows, got %+v", workflows)
	}
}
