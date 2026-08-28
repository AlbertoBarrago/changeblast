package risk_test

import (
	"testing"

	"github.com/AlbertoBarrago/serval/internal/risk"
)

func TestMatchHighRiskPath(t *testing.T) {
	patterns := []string{
		"**/migrations/**",
		"**/*.env*",
		"protocol.ts",
		"**/infra/**",
	}

	cases := []struct {
		name    string
		path    string
		matched bool
	}{
		{"nested under double-star prefix", "db/migrations/2024_init.sql", true},
		{"double-star at root", "migrations/2024_init.sql", true},
		{"double-star suffix glob on filename", "config/.env.local", true},
		{"double-star suffix glob at root", ".env", true},
		{"exact literal match", "protocol.ts", true},
		{"literal does not match nested path", "src/protocol.ts", false},
		{"double-star directory match", "apps/api/infra/terraform.tf", true},
		{"unrelated path does not match", "src/api/client.ts", false},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			_, matched := risk.MatchHighRiskPath(tc.path, patterns)
			if matched != tc.matched {
				t.Errorf("MatchHighRiskPath(%q) = %v, want %v", tc.path, matched, tc.matched)
			}
		})
	}
}

func TestMatchHighRiskPath_NoPatternsDisablesFloor(t *testing.T) {
	if _, matched := risk.MatchHighRiskPath("db/migrations/2024_init.sql", nil); matched {
		t.Error("expected no match with nil patterns")
	}
}

func TestCompute_HighRiskPathForcesHighFloor(t *testing.T) {
	score := risk.Compute(risk.Input{
		TargetPath:    "db/migrations/2024_init.sql",
		HighRiskPaths: risk.DefaultHighRiskPaths,
	})

	if score.Level != risk.LevelHigh {
		t.Errorf("expected forced HIGH level, got %s", score.Level)
	}
	if !score.Forced {
		t.Error("expected Forced=true")
	}
	if score.ForcedReason == "" {
		t.Error("expected a non-empty ForcedReason")
	}
	if score.Total != 0 {
		t.Errorf("expected Total to stay at the computed value (0), got %d", score.Total)
	}
}

func TestCompute_HighRiskPathNoMatchLeavesScoreUnforced(t *testing.T) {
	score := risk.Compute(risk.Input{
		TargetPath:    "src/api/client.ts",
		HighRiskPaths: risk.DefaultHighRiskPaths,
	})

	if score.Forced {
		t.Error("expected Forced=false for a non-matching path")
	}
	if score.Level != risk.LevelLow {
		t.Errorf("expected LOW level, got %s", score.Level)
	}
}

func TestCompute_HighRiskPathAlreadyHighStaysForced(t *testing.T) {
	score := risk.Compute(risk.Input{
		TargetPath:      "db/migrations/2024_init.sql",
		DownstreamCount: 14,
		HighRiskPaths:   risk.DefaultHighRiskPaths,
	})

	if score.Level != risk.LevelHigh {
		t.Errorf("expected HIGH level, got %s", score.Level)
	}
	if !score.Forced {
		t.Error("expected Forced=true even when the computed score already reaches HIGH")
	}
}
