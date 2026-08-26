package repository_test

import (
	"path/filepath"
	"testing"

	"github.com/AlbertoBarrago/changeblast/internal/impact"
	"github.com/AlbertoBarrago/changeblast/internal/repository"
)

func TestScanAndComputeImpact_SimpleTS(t *testing.T) {
	root, err := filepath.Abs("../../testdata/fixtures/simple-ts")
	if err != nil {
		t.Fatal(err)
	}

	scanner, err := repository.NewScanner(root)
	if err != nil {
		t.Fatalf("NewScanner: %v", err)
	}

	g, err := scanner.Scan()
	if err != nil {
		t.Fatalf("Scan: %v", err)
	}

	target := filepath.Join(root, "src/auth/token.ts")
	if !g.HasNode(target) {
		t.Fatalf("expected graph to contain target %s", target)
	}

	result := impact.Compute(g, target)

	wantDirect := filepath.Join(root, "src/auth/middleware.ts")
	if !contains(result.Direct, wantDirect) {
		t.Errorf("expected direct impact to contain %s, got %v", wantDirect, result.Direct)
	}

	wantIndirect := filepath.Join(root, "src/api/client.ts")
	if !contains(result.Indirect, wantIndirect) {
		t.Errorf("expected indirect impact to contain %s, got %v", wantIndirect, result.Indirect)
	}

	unrelated := filepath.Join(root, "src/unrelated.ts")
	if contains(result.Direct, unrelated) || contains(result.Indirect, unrelated) {
		t.Errorf("expected unrelated.ts to not be part of impact")
	}
}

func contains(list []string, item string) bool {
	for _, v := range list {
		if v == item {
			return true
		}
	}
	return false
}
