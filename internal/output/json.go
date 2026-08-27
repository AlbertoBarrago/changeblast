package output

import (
	"path/filepath"

	"github.com/AlbertoBarrago/blast/internal/impact"
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
	rel := func(p string) string {
		if r, err := filepath.Rel(root, p); err == nil {
			return r
		}
		return p
	}

	direct := make([]string, len(result.Direct))
	for i, d := range result.Direct {
		direct[i] = rel(d)
	}
	indirect := make([]string, len(result.Indirect))
	for i, d := range result.Indirect {
		indirect[i] = rel(d)
	}

	return InspectJSON{
		Target:   rel(result.Target),
		Direct:   direct,
		Indirect: indirect,
	}
}
