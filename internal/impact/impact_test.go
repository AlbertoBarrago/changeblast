package impact_test

import (
	"reflect"
	"sort"
	"testing"
	"time"

	"github.com/AlbertoBarrago/impactline/internal/graph"
	"github.com/AlbertoBarrago/impactline/internal/impact"
)

func TestCompute_DirectAndIndirect(t *testing.T) {
	// token <- middleware <- client (chain), plus unrelated node.
	g := graph.New()
	g.AddEdge("middleware", "token")
	g.AddEdge("client", "middleware")
	g.AddNode("unrelated")

	result := impact.Compute(g, "token")

	if !reflect.DeepEqual(result.Direct, []string{"middleware"}) {
		t.Errorf("Direct = %v, want [middleware]", result.Direct)
	}
	if !reflect.DeepEqual(result.Indirect, []string{"client"}) {
		t.Errorf("Indirect = %v, want [client]", result.Indirect)
	}
}

func TestCompute_NoDependents(t *testing.T) {
	g := graph.New()
	g.AddNode("isolated")

	result := impact.Compute(g, "isolated")

	if len(result.Direct) != 0 {
		t.Errorf("expected no direct impact, got %v", result.Direct)
	}
	if len(result.Indirect) != 0 {
		t.Errorf("expected no indirect impact, got %v", result.Indirect)
	}
}

func TestCompute_CycleDoesNotInfiniteLoop(t *testing.T) {
	g := graph.New()
	g.AddEdge("a", "b")
	g.AddEdge("b", "a") // cycle

	done := make(chan impact.Result, 1)
	go func() { done <- impact.Compute(g, "b") }()

	select {
	case result := <-done:
		if !reflect.DeepEqual(result.Direct, []string{"a"}) {
			t.Errorf("Direct = %v, want [a]", result.Direct)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("Compute did not terminate on a cyclic graph")
	}
}

func TestCompute_DiamondDependencyDeduped(t *testing.T) {
	// target <- left, right <- top (diamond); top must appear once in indirect.
	g := graph.New()
	g.AddEdge("left", "target")
	g.AddEdge("right", "target")
	g.AddEdge("top", "left")
	g.AddEdge("top", "right")

	result := impact.Compute(g, "target")

	direct := sorted(result.Direct)
	if !reflect.DeepEqual(direct, []string{"left", "right"}) {
		t.Errorf("Direct = %v, want [left right]", direct)
	}
	if !reflect.DeepEqual(result.Indirect, []string{"top"}) {
		t.Errorf("Indirect = %v, want [top] (deduped)", result.Indirect)
	}
}

func sorted(s []string) []string {
	out := append([]string{}, s...)
	sort.Strings(out)
	return out
}
