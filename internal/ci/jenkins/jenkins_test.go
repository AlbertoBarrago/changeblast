package jenkins_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/AlbertoBarrago/serval/internal/ci/jenkins"
)

func TestDiscover(t *testing.T) {
	root := t.TempDir()
	pipeline := `
// pipeline definition
pipeline {
    agent any
    stages {
        stage('Auth Tests') {
            when {
                changeset "src/auth/**"
            }
            steps {
                sh 'run auth tests'
            }
        }
        stage('Lint') {
            steps {
                sh 'lint'
            }
        }
    }
}
`
	if err := os.WriteFile(filepath.Join(root, "Jenkinsfile"), []byte(pipeline), 0o644); err != nil {
		t.Fatal(err)
	}

	p := jenkins.New()
	workflows, err := p.Discover(root)
	if err != nil {
		t.Fatalf("Discover: %v", err)
	}
	if len(workflows) != 2 {
		t.Fatalf("expected 2 stages, got %d: %+v", len(workflows), workflows)
	}

	var authTests, lint bool
	for _, wf := range workflows {
		switch wf.Name {
		case "Auth Tests":
			authTests = true
			if len(wf.PathFilters) != 1 || wf.PathFilters[0] != "src/auth/**" {
				t.Errorf("unexpected path filters for Auth Tests: %+v", wf.PathFilters)
			}
		case "Lint":
			lint = true
			if len(wf.PathFilters) != 0 {
				t.Errorf("expected Lint to have no path filters, got %+v", wf.PathFilters)
			}
		}
	}
	if !authTests || !lint {
		t.Errorf("expected both stages to be discovered, got %+v", workflows)
	}
}

func TestDiscover_NoJenkinsfile(t *testing.T) {
	root := t.TempDir()
	p := jenkins.New()
	workflows, err := p.Discover(root)
	if err != nil {
		t.Fatalf("Discover: %v", err)
	}
	if workflows != nil {
		t.Errorf("expected nil workflows, got %+v", workflows)
	}
}

func TestDiscover_ScriptedPipelineHasNoStages(t *testing.T) {
	root := t.TempDir()
	pipeline := `
node {
    sh 'echo hello'
}
`
	if err := os.WriteFile(filepath.Join(root, "Jenkinsfile"), []byte(pipeline), 0o644); err != nil {
		t.Fatal(err)
	}

	p := jenkins.New()
	workflows, err := p.Discover(root)
	if err != nil {
		t.Fatalf("Discover: %v", err)
	}
	if workflows != nil {
		t.Errorf("expected nil workflows for scripted pipeline, got %+v", workflows)
	}
}
