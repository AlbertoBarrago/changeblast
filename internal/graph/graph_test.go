package graph_test

import (
	"reflect"
	"sort"
	"testing"

	"github.com/AlbertoBarrago/serval/internal/graph"
)

func TestAddEdge_DependentsAndDependencies(t *testing.T) {
	g := graph.New()
	g.AddEdge("a", "b")
	g.AddEdge("c", "b")

	deps := sorted(g.Dependents("b"))
	if !reflect.DeepEqual(deps, []string{"a", "c"}) {
		t.Errorf("Dependents(b) = %v, want [a c]", deps)
	}

	if !reflect.DeepEqual(g.Dependencies("a"), []string{"b"}) {
		t.Errorf("Dependencies(a) = %v, want [b]", g.Dependencies("a"))
	}
}

func TestAddNode_IsolatedNodeHasNoEdges(t *testing.T) {
	g := graph.New()
	g.AddNode("standalone.ts")

	if !g.HasNode("standalone.ts") {
		t.Fatal("expected standalone.ts to be a known node")
	}
	if len(g.Dependents("standalone.ts")) != 0 {
		t.Errorf("expected no dependents, got %v", g.Dependents("standalone.ts"))
	}
	if len(g.Dependencies("standalone.ts")) != 0 {
		t.Errorf("expected no dependencies, got %v", g.Dependencies("standalone.ts"))
	}
}

func TestHasNode_Unknown(t *testing.T) {
	g := graph.New()
	if g.HasNode("nope") {
		t.Error("expected unknown node to report false")
	}
}

func TestDependents_ReturnedInSortedOrder(t *testing.T) {
	g := graph.New()
	// Insert edges out of alphabetical order; map iteration order is
	// randomized per Go process, so a regression here would only show up
	// intermittently without this explicit ordering assertion.
	for _, importer := range []string{"z.ts", "m.ts", "a.ts", "q.ts", "b.ts"} {
		g.AddEdge(importer, "target.ts")
	}

	want := []string{"a.ts", "b.ts", "m.ts", "q.ts", "z.ts"}
	if got := g.Dependents("target.ts"); !reflect.DeepEqual(got, want) {
		t.Errorf("Dependents(target.ts) = %v, want %v", got, want)
	}
}

func TestNodes_ReturnedInSortedOrder(t *testing.T) {
	g := graph.New()
	for _, importer := range []string{"z.ts", "m.ts", "a.ts"} {
		g.AddNode(importer)
	}

	want := []string{"a.ts", "m.ts", "z.ts"}
	if got := g.Nodes(); !reflect.DeepEqual(got, want) {
		t.Errorf("Nodes() = %v, want %v", got, want)
	}
}

func TestAddEdge_AutoRegistersNodes(t *testing.T) {
	g := graph.New()
	g.AddEdge("a", "b")

	if !g.HasNode("a") || !g.HasNode("b") {
		t.Error("expected both endpoints of an edge to be registered as nodes")
	}
}

func sorted(s []string) []string {
	out := append([]string{}, s...)
	sort.Strings(out)
	return out
}
