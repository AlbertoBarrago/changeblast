package risk

import "testing"

func TestComputeSet_EmptyInput(t *testing.T) {
	score := ComputeSet(SetInput{})

	if score.Total != 0 || score.Level != LevelLow {
		t.Errorf("ComputeSet({}) = %+v, want 0/LOW", score)
	}
	if len(score.Breakdown) != 0 {
		t.Errorf("breakdown = %v, want empty", score.Breakdown)
	}
}

func TestComputeSet_DownstreamUnion(t *testing.T) {
	score := ComputeSet(SetInput{DownstreamCount: 5})

	if score.Total != 5*WeightPerDownstreamModule {
		t.Errorf("total = %d, want %d", score.Total, 5*WeightPerDownstreamModule)
	}
}

func TestComputeSet_DownstreamCapped(t *testing.T) {
	score := ComputeSet(SetInput{DownstreamCount: 100})

	if score.Breakdown[0].Points != WeightDownstreamCap {
		t.Errorf("downstream points = %d, want cap %d", score.Breakdown[0].Points, WeightDownstreamCap)
	}
}

func TestComputeSet_HighRiskPathForcesHighFloor(t *testing.T) {
	score := ComputeSet(SetInput{
		TargetPaths:   []string{"src/api/client.ts", "db/migrations/2024_init.sql"},
		HighRiskPaths: DefaultHighRiskPaths,
	})

	if score.Level != LevelHigh {
		t.Errorf("expected forced HIGH level, got %s", score.Level)
	}
	if !score.Forced {
		t.Error("expected Forced=true")
	}
	if score.Total != 0 {
		t.Errorf("expected Total to stay at the computed value (0), got %d", score.Total)
	}
}

func TestComputeSet_HighRiskPathNoMatchLeavesScoreUnforced(t *testing.T) {
	score := ComputeSet(SetInput{
		TargetPaths:   []string{"src/api/client.ts"},
		HighRiskPaths: DefaultHighRiskPaths,
	})

	if score.Forced {
		t.Error("expected Forced=false when no target matches")
	}
}

func TestComputeSet_InternalEdges(t *testing.T) {
	score := ComputeSet(SetInput{InternalEdgeCount: 2})

	if score.Total != 2*WeightSetInteraction {
		t.Errorf("total = %d, want %d", score.Total, 2*WeightSetInteraction)
	}
}

func TestComputeSet_InternalEdgesCapped(t *testing.T) {
	score := ComputeSet(SetInput{InternalEdgeCount: 10})

	if score.Breakdown[0].Points != WeightSetInteractionCap {
		t.Errorf("interaction points = %d, want cap %d", score.Breakdown[0].Points, WeightSetInteractionCap)
	}
}

func TestComputeSet_SharedDependents(t *testing.T) {
	score := ComputeSet(SetInput{SharedDependentCount: 3})

	if score.Total != WeightSharedDependents {
		t.Errorf("total = %d, want %d (flat bonus, not per module)", score.Total, WeightSharedDependents)
	}
}

func TestComputeSet_CriticalPathAnyTarget(t *testing.T) {
	score := ComputeSet(SetInput{
		TargetPaths:          []string{"src/util/helper.ts", "src/auth/login.ts"},
		CriticalPathKeywords: []string{"auth"},
	})

	if score.Total != WeightCriticalPath {
		t.Errorf("total = %d, want %d (awarded once, not per match)", score.Total, WeightCriticalPath)
	}
}

func TestComputeSet_CriticalPathDisabledWithoutKeywords(t *testing.T) {
	score := ComputeSet(SetInput{
		TargetPaths:          []string{"src/auth/login.ts"},
		CriticalPathKeywords: nil,
	})

	if score.Total != 0 {
		t.Errorf("total = %d, want 0 (nil keywords disable matching, as with Compute)", score.Total)
	}
}

func TestComputeSet_ChurnTiersOnMax(t *testing.T) {
	for _, tc := range []struct {
		churn int
		want  int
	}{
		{0, 0},
		{1, WeightChurnLow},
		{ChurnMediumThreshold, WeightChurnMedium},
		{ChurnHighThreshold, WeightChurnHigh},
	} {
		score := ComputeSet(SetInput{ChurnCount: tc.churn})
		if score.Total != tc.want {
			t.Errorf("churn %d: total = %d, want %d", tc.churn, score.Total, tc.want)
		}
	}
}

func TestComputeSet_CIUnion(t *testing.T) {
	score := ComputeSet(SetInput{RelevantWorkflows: []string{"ci.yml", "release.yml"}})

	if score.Total != WeightCIImpact {
		t.Errorf("total = %d, want %d", score.Total, WeightCIImpact)
	}
}

func TestComputeSet_MaxScoreConsistent(t *testing.T) {
	// Every weight maxed out: the breakdown must still sum to the total.
	score := ComputeSet(SetInput{
		DownstreamCount:      100,
		InternalEdgeCount:    10,
		SharedDependentCount: 5,
		ChurnCount:           ChurnHighThreshold,
		RelevantWorkflows:    []string{"ci.yml"},
		TargetPaths:          []string{"src/auth/login.ts"},
		CriticalPathKeywords: []string{"auth"},
	})

	if score.Level != LevelHigh {
		t.Errorf("level = %s, want HIGH", score.Level)
	}
	sum := 0
	for _, e := range score.Breakdown {
		sum += e.Points
		if e.Points <= 0 {
			t.Errorf("breakdown entry with non-positive points: %+v", e)
		}
	}
	if sum != score.Total {
		t.Errorf("breakdown sums to %d, want total %d", sum, score.Total)
	}
}

func TestComputeSet_Deterministic(t *testing.T) {
	input := SetInput{
		DownstreamCount:      7,
		InternalEdgeCount:    2,
		SharedDependentCount: 1,
		ChurnCount:           3,
		RelevantWorkflows:    []string{"ci.yml"},
		TargetPaths:          []string{"src/auth/login.ts", "src/api/routes.ts"},
		CriticalPathKeywords: []string{"auth"},
	}

	a, b := ComputeSet(input), ComputeSet(input)
	if a.Total != b.Total || len(a.Breakdown) != len(b.Breakdown) {
		t.Errorf("ComputeSet not deterministic: %+v vs %+v", a, b)
	}
}
