package ci_test

import (
	"testing"

	"github.com/AlbertoBarrago/blast/internal/ci"
)

func TestRelevant(t *testing.T) {
	workflows := []ci.Workflow{
		{Name: "auth", Path: "auth.yml", PathFilters: []string{"src/auth/**"}},
		{Name: "unfiltered", Path: "unfiltered.yml"},
		{Name: "api", Path: "api.yml", PathFilters: []string{"src/api/*.ts"}},
	}

	changed := []string{"src/auth/token.ts"}
	got := ci.Relevant(workflows, changed)

	names := map[string]bool{}
	for _, wf := range got {
		names[wf.Name] = true
	}

	if !names["auth"] {
		t.Error("expected auth workflow to be relevant")
	}
	if !names["unfiltered"] {
		t.Error("expected unfiltered workflow to always be relevant")
	}
	if names["api"] {
		t.Error("expected api workflow to not be relevant")
	}
}
