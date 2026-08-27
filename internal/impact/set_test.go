package impact_test

import (
	"reflect"
	"testing"

	"github.com/AlbertoBarrago/serval/internal/graph"
	"github.com/AlbertoBarrago/serval/internal/impact"
)

// Graph fixture: a and b are the change set; user imports a; user also
// imports b (shared dependent); svc imports b; api imports user
// (transitive via the union); a imports b (internal edge).
//
//	user -> a, user -> b, svc -> b, api -> user, a -> b
func setFixture() *graph.Graph {
	g := graph.New()
	g.AddEdge("user", "a")
	g.AddEdge("user", "b")
	g.AddEdge("svc", "b")
	g.AddEdge("api", "user")
	g.AddEdge("a", "b")
	g.AddNode("unrelated")
	return g
}

func TestComputeSet_DeduplicatesDirectUnion(t *testing.T) {
	result := impact.ComputeSet(setFixture(), []string{"a", "b"})

	// user imports both a and b; the union counts it once.
	want := []string{"svc", "user"}
	if !reflect.DeepEqual(result.Direct, want) {
		t.Errorf("Direct = %v, want %v", result.Direct, want)
	}
}

func TestComputeSet_TransitiveBeyondUnion(t *testing.T) {
	result := impact.ComputeSet(setFixture(), []string{"a", "b"})

	want := []string{"api"}
	if !reflect.DeepEqual(result.Indirect, want) {
		t.Errorf("Indirect = %v, want %v", result.Indirect, want)
	}
}

func TestComputeSet_TwoTargetsNoOverlap(t *testing.T) {
	g := graph.New()
	g.AddEdge("x", "a")
	g.AddEdge("y", "b")

	result := impact.ComputeSet(g, []string{"a", "b"})

	if !reflect.DeepEqual(result.Direct, []string{"x", "y"}) {
		t.Errorf("Direct = %v, want [x y]", result.Direct)
	}
	if len(result.Indirect) != 0 {
		t.Errorf("Indirect = %v, want empty", result.Indirect)
	}
	if len(result.SharedDependents) != 0 {
		t.Errorf("SharedDependents = %v, want empty", result.SharedDependents)
	}
}

func TestComputeSet_InternalEdges(t *testing.T) {
	result := impact.ComputeSet(setFixture(), []string{"a", "b"})

	// a imports b and both are in the set: an interaction, not an impact.
	want := []impact.Edge{{Importer: "a", Imported: "b"}}
	if !reflect.DeepEqual(result.InternalEdges, want) {
		t.Errorf("InternalEdges = %v, want %v", result.InternalEdges, want)
	}
	// b does not appear in Direct/Indirect even though a imports it.
	for _, p := range append(append([]string{}, result.Direct...), result.Indirect...) {
		if p == "a" || p == "b" {
			t.Errorf("target %q leaked into impact lists (Direct=%v Indirect=%v)", p, result.Direct, result.Indirect)
		}
	}
}

func TestComputeSet_SharedDependents(t *testing.T) {
	result := impact.ComputeSet(setFixture(), []string{"a", "b"})

	want := []string{"user"}
	if !reflect.DeepEqual(result.SharedDependents, want) {
		t.Errorf("SharedDependents = %v, want %v", result.SharedDependents, want)
	}
}

func TestComputeSet_SingleTargetMatchesCompute(t *testing.T) {
	g := setFixture()

	set := impact.ComputeSet(g, []string{"b"})
	single := impact.Compute(g, "b")

	if !reflect.DeepEqual(set.Direct, single.Direct) {
		t.Errorf("Direct = %v, Compute says %v", set.Direct, single.Direct)
	}
	if !reflect.DeepEqual(set.Indirect, single.Indirect) {
		t.Errorf("Indirect = %v, Compute says %v", set.Indirect, single.Indirect)
	}
	if len(set.InternalEdges) != 0 {
		t.Errorf("InternalEdges = %v, want empty for a single target", set.InternalEdges)
	}
}

func TestComputeSet_NoTargets(t *testing.T) {
	result := impact.ComputeSet(setFixture(), nil)

	if len(result.Direct) != 0 || len(result.Indirect) != 0 ||
		len(result.InternalEdges) != 0 || len(result.SharedDependents) != 0 {
		t.Errorf("ComputeSet(nil) = %+v, want all empty", result)
	}
}

func TestComputeSet_DeterministicOrdering(t *testing.T) {
	g := setFixture()

	first := impact.ComputeSet(g, []string{"a", "b"})
	second := impact.ComputeSet(g, []string{"b", "a"})

	if !reflect.DeepEqual(first.Direct, second.Direct) ||
		!reflect.DeepEqual(first.Indirect, second.Indirect) ||
		!reflect.DeepEqual(first.InternalEdges, second.InternalEdges) ||
		!reflect.DeepEqual(first.SharedDependents, second.SharedDependents) {
		t.Errorf("results differ across runs: %+v vs %+v", first, second)
	}
}
