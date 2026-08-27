package output

import (
	"fmt"
	"io"
	"path/filepath"

	"github.com/AlbertoBarrago/serval/internal/git"
)

// FrequentCoChangeThreshold is the minimum co-change count for a file to
// be counted as "frequently co-changed" in the human-readable summary
// and in risk scoring.
const FrequentCoChangeThreshold = 2

// FrequentCoChangeCount returns how many files in h.CoChanged meet
// FrequentCoChangeThreshold.
func FrequentCoChangeCount(h git.FileHistory) int {
	n := 0
	for _, c := range h.CoChanged {
		if c.Count >= FrequentCoChangeThreshold {
			n++
		}
	}
	return n
}

// RenderHistoryText writes a human-readable rendering of a file's Git
// history signals to w.
func RenderHistoryText(w io.Writer, root string, h git.FileHistory) {
	rel := func(p string) string {
		if r, err := filepath.Rel(root, p); err == nil {
			return r
		}
		return p
	}

	fmt.Fprintln(w, "Target")
	fmt.Fprintf(w, "  %s\n\n", rel(h.Path))

	fmt.Fprintln(w, "Git history")
	fmt.Fprintf(w, "  %d significant changes (last %d days)\n", h.Changes, h.Window.Days)

	frequent := FrequentCoChangeCount(h)
	fmt.Fprintf(w, "  %d frequently co-changed modules\n", frequent)

	if frequent > 0 {
		fmt.Fprintln(w)
		fmt.Fprintln(w, "Frequently co-changed")
		for _, c := range h.CoChanged {
			if c.Count < FrequentCoChangeThreshold {
				continue
			}
			fmt.Fprintf(w, "  %s (%d times)\n", rel(c.Path), c.Count)
		}
	}
}

// HistoryJSON is the JSON-serializable shape of a history result.
type HistoryJSON struct {
	Target        string         `json:"target"`
	HistoryWindow git.Window     `json:"historyWindow"`
	Changes       int            `json:"changes"`
	CoChanged     []git.CoChange `json:"coChanged"`
}

// ToHistoryJSON converts a git.FileHistory to its JSON representation,
// rendering paths relative to root.
func ToHistoryJSON(root string, h git.FileHistory) HistoryJSON {
	rel := func(p string) string {
		if r, err := filepath.Rel(root, p); err == nil {
			return r
		}
		return p
	}

	coChanged := make([]git.CoChange, len(h.CoChanged))
	for i, c := range h.CoChanged {
		coChanged[i] = git.CoChange{Path: rel(c.Path), Count: c.Count}
	}

	return HistoryJSON{
		Target:        rel(h.Path),
		HistoryWindow: h.Window,
		Changes:       h.Changes,
		CoChanged:     coChanged,
	}
}
