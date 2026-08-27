package risk_test

import (
	"testing"

	"github.com/AlbertoBarrago/impactline/internal/risk"
)

func TestCompute_Deterministic(t *testing.T) {
	input := risk.Input{
		TargetPath:            "src/auth/token.ts",
		DownstreamCount:       14,
		ChurnCount:            7,
		FrequentCoChangeCount: 3,
		RelevantWorkflows:     []string{"integration-auth.yml"},
		CriticalPathKeywords:  risk.DefaultCriticalPathKeywords,
	}

	first := risk.Compute(input)
	second := risk.Compute(input)

	if first.Total != second.Total {
		t.Fatalf("expected deterministic score, got %d then %d", first.Total, second.Total)
	}

	want := risk.WeightDownstreamCap + risk.WeightCriticalPath + risk.WeightChurnHigh + risk.WeightCoChange + risk.WeightCIImpact
	if first.Total != want {
		t.Errorf("expected total %d, got %d (breakdown: %+v)", want, first.Total, first.Breakdown)
	}
	if first.Level != risk.LevelHigh {
		t.Errorf("expected HIGH level, got %s", first.Level)
	}
	if len(first.Breakdown) != 5 {
		t.Errorf("expected 5 breakdown entries, got %d: %+v", len(first.Breakdown), first.Breakdown)
	}
}

func TestCompute_ZeroInputIsLow(t *testing.T) {
	score := risk.Compute(risk.Input{})
	if score.Total != 0 {
		t.Errorf("expected 0, got %d", score.Total)
	}
	if score.Level != risk.LevelLow {
		t.Errorf("expected LOW, got %s", score.Level)
	}
	if len(score.Breakdown) != 0 {
		t.Errorf("expected empty breakdown, got %+v", score.Breakdown)
	}
}

func TestMatchCriticalPath(t *testing.T) {
	cases := []struct {
		path    string
		matched bool
	}{
		{"src/auth/token.ts", true},
		{"src/payment/checkout.ts", true},
		{"src/api/client.ts", false},
		{"src/security/scan.ts", true},
	}

	for _, c := range cases {
		_, matched := risk.MatchCriticalPath(c.path, risk.DefaultCriticalPathKeywords)
		if matched != c.matched {
			t.Errorf("MatchCriticalPath(%q) = %v, want %v", c.path, matched, c.matched)
		}
	}
}
