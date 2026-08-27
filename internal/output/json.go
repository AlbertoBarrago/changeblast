package output

import (
	"github.com/AlbertoBarrago/serval/internal/impact"
)

// InspectJSON is the JSON-serializable shape of an inspect result.
// Field names are stable API surface once v0.1 ships.
type InspectJSON struct {
	Target   string   `json:"target"`
	Direct   []string `json:"direct"`
	Indirect []string `json:"indirect"`
}

// ToInspectJSON converts an impact.Result to its JSON representation,
// rendering paths relative to root.
func ToInspectJSON(root string, result impact.Result) InspectJSON {

	direct := make([]string, len(result.Direct))
	for i, d := range result.Direct {
		direct[i] = relPath(root, d)
	}
	indirect := make([]string, len(result.Indirect))
	for i, d := range result.Indirect {
		indirect[i] = relPath(root, d)
	}

	return InspectJSON{
		Target:   relPath(root, result.Target),
		Direct:   direct,
		Indirect: indirect,
	}
}
