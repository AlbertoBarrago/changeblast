package output

import (
	"fmt"
	"io"

	"github.com/AlbertoBarrago/changeblast/internal/ci"
	"github.com/AlbertoBarrago/changeblast/internal/git"
	"github.com/AlbertoBarrago/changeblast/internal/impact"
	"github.com/AlbertoBarrago/changeblast/internal/risk"
)

// InspectResult aggregates every signal shown by `blast inspect`: the
// dependency impact, Git history, relevant CI workflows, and the
// resulting risk score. Renderers only format this — they compute
// nothing.
type InspectResult struct {
	Impact            impact.Result
	History           git.FileHistory
	RelevantWorkflows []ci.Workflow
	Risk              risk.Score
}

// RenderInspectFull writes the complete human-readable `blast inspect`
// report to w.
func RenderInspectFull(w io.Writer, root string, r InspectResult) {
	RenderInspectText(w, root, r.Impact)
	fmt.Fprintln(w)

	fmt.Fprintln(w, "CI")
	if len(r.RelevantWorkflows) == 0 {
		fmt.Fprintln(w, "  (none found)")
	}
	for _, wf := range r.RelevantWorkflows {
		fmt.Fprintf(w, "  %s\n", wf.Path)
	}
	fmt.Fprintln(w)

	fmt.Fprintln(w, "Git history")
	fmt.Fprintf(w, "  %d significant changes (last %d days)\n", r.History.Changes, r.History.Window.Days)
	frequent := FrequentCoChangeCount(r.History)
	fmt.Fprintf(w, "  %d frequently co-changed modules\n", frequent)
	fmt.Fprintln(w)

	fmt.Fprintln(w, "Risk")
	fmt.Fprintf(w, "  %s — %d/100\n", colorizeLevel(w, r.Risk.Level), r.Risk.Total)
	for _, e := range r.Risk.Breakdown {
		fmt.Fprintf(w, "  +%-3d %s\n", e.Points, e.Reason)
	}
}

// colorizeLevel renders a risk level with a color matching its severity
// (red HIGH, yellow MEDIUM, green LOW) when w supports it.
func colorizeLevel(w io.Writer, level risk.Level) string {
	enabled := colorEnabled(w)
	switch level {
	case risk.LevelHigh:
		return colorize(enabled, ansiBold+ansiRed, string(level))
	case risk.LevelMedium:
		return colorize(enabled, ansiBold+ansiYellow, string(level))
	default:
		return colorize(enabled, ansiBold+ansiGreen, string(level))
	}
}

// InspectFullJSON is the JSON-serializable shape of a full inspect
// result.
type InspectFullJSON struct {
	Target            string         `json:"target"`
	Direct            []string       `json:"direct"`
	Indirect          []string       `json:"indirect"`
	HistoryWindow     git.Window     `json:"historyWindow"`
	Changes           int            `json:"changes"`
	CoChanged         []git.CoChange `json:"coChanged"`
	RelevantWorkflows []string       `json:"relevantWorkflows"`
	Risk              risk.Score     `json:"risk"`
}

// ToInspectFullJSON converts an InspectResult to its JSON representation.
func ToInspectFullJSON(root string, r InspectResult) InspectFullJSON {
	base := ToInspectJSON(root, r.Impact)
	hist := ToHistoryJSON(root, r.History)

	workflows := make([]string, len(r.RelevantWorkflows))
	for i, wf := range r.RelevantWorkflows {
		workflows[i] = wf.Path
	}

	return InspectFullJSON{
		Target:            base.Target,
		Direct:            base.Direct,
		Indirect:          base.Indirect,
		HistoryWindow:     hist.HistoryWindow,
		Changes:           hist.Changes,
		CoChanged:         hist.CoChanged,
		RelevantWorkflows: workflows,
		Risk:              r.Risk,
	}
}
