// Package graph provides a directed dependency graph over repository files.
// Edges point from an importer file to the file it imports.
package graph

// Graph is a directed multigraph of file dependencies.
type Graph struct {
	// forward maps an importer path to the paths it imports.
	forward map[string]map[string]struct{}
	// reverse maps an imported path to the paths that import it.
	reverse map[string]map[string]struct{}
	// nodes tracks every file known to the graph, even if it has no edges.
	nodes map[string]struct{}
}

// New returns an empty dependency graph.
func New() *Graph {
	return &Graph{
		forward: make(map[string]map[string]struct{}),
		reverse: make(map[string]map[string]struct{}),
		nodes:   make(map[string]struct{}),
	}
}

// AddNode registers a file in the graph even if it has no known edges yet.
func (g *Graph) AddNode(path string) {
	g.nodes[path] = struct{}{}
}

// AddEdge records that importer depends on imported.
func (g *Graph) AddEdge(importer, imported string) {
	g.AddNode(importer)
	g.AddNode(imported)

	if g.forward[importer] == nil {
		g.forward[importer] = make(map[string]struct{})
	}
	g.forward[importer][imported] = struct{}{}

	if g.reverse[imported] == nil {
		g.reverse[imported] = make(map[string]struct{})
	}
	g.reverse[imported][importer] = struct{}{}
}

// Dependents returns the files that directly import the given path.
func (g *Graph) Dependents(path string) []string {
	return keys(g.reverse[path])
}

// Dependencies returns the files the given path directly imports.
func (g *Graph) Dependencies(path string) []string {
	return keys(g.forward[path])
}

// HasNode reports whether path is known to the graph.
func (g *Graph) HasNode(path string) bool {
	_, ok := g.nodes[path]
	return ok
}

// Nodes returns every file known to the graph.
func (g *Graph) Nodes() []string {
	return keys(g.nodes)
}

func keys(m map[string]struct{}) []string {
	out := make([]string, 0, len(m))
	for k := range m {
		out = append(out, k)
	}
	return out
}
