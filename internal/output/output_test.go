package output_test

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"

	"github.com/AlbertoBarrago/serval/internal/ci"
	"github.com/AlbertoBarrago/serval/internal/git"
	"github.com/AlbertoBarrago/serval/internal/impact"
	"github.com/AlbertoBarrago/serval/internal/output"
	"github.com/AlbertoBarrago/serval/internal/risk"
)

func TestRenderInspectText(t *testing.T) {
	var buf bytes.Buffer
	result := impact.Result{
		Target:   "/repo/src/auth/token.ts",
		Direct:   []string{"/repo/src/auth/middleware.ts"},
		Indirect: []string{"/repo/src/api/client.ts"},
	}

	output.RenderInspectText(&buf, "/repo", result)
	got := buf.String()

	for _, want := range []string{"src/auth/token.ts", "src/auth/middleware.ts", "src/api/client.ts", "Direct impact", "Indirect impact"} {
		if !strings.Contains(got, want) {
			t.Errorf("expected output to contain %q, got:\n%s", want, got)
		}
	}
}

func TestRenderInspectText_EmptyImpact(t *testing.T) {
	var buf bytes.Buffer
	output.RenderInspectText(&buf, "/repo", impact.Result{Target: "/repo/x.ts"})

	got := buf.String()
	if strings.Count(got, "(none found)") != 2 {
		t.Errorf("expected two '(none found)' markers for empty direct/indirect, got:\n%s", got)
	}
}

func TestRenderInspectFull_NoColorWhenNotATTY(t *testing.T) {
	var buf bytes.Buffer
	r := output.InspectResult{
		Impact: impact.Result{Target: "/repo/x.ts"},
		Risk:   risk.Compute(risk.Input{}),
	}

	output.RenderInspectFull(&buf, "/repo", r)

	if strings.Contains(buf.String(), "\x1b[") {
		t.Error("expected no ANSI escape codes when writing to a non-TTY buffer")
	}
}

func TestToInspectFullJSON_RoundTrips(t *testing.T) {
	r := output.InspectResult{
		Impact: impact.Result{
			Target:   "/repo/x.ts",
			Direct:   []string{"/repo/y.ts"},
			Indirect: nil,
		},
		History: git.FileHistory{
			Path:    "/repo/x.ts",
			Window:  git.DefaultWindow,
			Changes: 3,
		},
		RelevantWorkflows: []ci.Workflow{{Path: ".github/workflows/ci.yml"}},
		Risk:              risk.Compute(risk.Input{DownstreamCount: 1}),
	}

	j := output.ToInspectFullJSON("/repo", r)

	raw, err := json.Marshal(j)
	if err != nil {
		t.Fatalf("Marshal: %v", err)
	}

	var decoded map[string]interface{}
	if err := json.Unmarshal(raw, &decoded); err != nil {
		t.Fatalf("Unmarshal: %v", err)
	}

	if decoded["target"] != "x.ts" {
		t.Errorf("target = %v, want x.ts", decoded["target"])
	}
	if decoded["changes"].(float64) != 3 {
		t.Errorf("changes = %v, want 3", decoded["changes"])
	}
	workflows := decoded["relevantWorkflows"].([]interface{})
	if len(workflows) != 1 || workflows[0] != ".github/workflows/ci.yml" {
		t.Errorf("relevantWorkflows = %v", workflows)
	}
}

func TestRenderHistoryText_FrequentCoChangeThreshold(t *testing.T) {
	var buf bytes.Buffer
	h := git.FileHistory{
		Path:   "/repo/x.ts",
		Window: git.DefaultWindow,
		CoChanged: []git.CoChange{
			{Path: "/repo/y.ts", Count: 2},
			{Path: "/repo/z.ts", Count: 1},
		},
	}

	output.RenderHistoryText(&buf, "/repo", h)
	got := buf.String()

	if !strings.Contains(got, "1 frequently co-changed modules") {
		t.Errorf("expected exactly 1 frequent co-change (count>=2 only), got:\n%s", got)
	}
	if !strings.Contains(got, "y.ts (2 times)") {
		t.Errorf("expected y.ts listed as frequent, got:\n%s", got)
	}
	if strings.Contains(got, "z.ts") {
		t.Errorf("expected z.ts (count=1) to be excluded from frequent list, got:\n%s", got)
	}
}

func TestRenderSummary_SortedByRiskDescending(t *testing.T) {
	var buf bytes.Buffer
	results := []output.InspectResult{
		{Impact: impact.Result{Target: "/repo/low.ts"}, Risk: risk.Score{Total: 10, Level: risk.LevelLow}},
		{Impact: impact.Result{Target: "/repo/high.ts"}, Risk: risk.Score{Total: 90, Level: risk.LevelHigh}},
		{Impact: impact.Result{Target: "/repo/medium.ts"}, Risk: risk.Score{Total: 40, Level: risk.LevelMedium}},
	}

	output.RenderSummary(&buf, "/repo", "Analyzed .", results)
	got := buf.String()

	iHigh := strings.Index(got, "high.ts")
	iMedium := strings.Index(got, "medium.ts")
	iLow := strings.Index(got, "low.ts")
	if !(iHigh < iMedium && iMedium < iLow) {
		t.Errorf("expected results sorted HIGH > MEDIUM > LOW by position, got:\n%s", got)
	}

	if !strings.Contains(got, "1 HIGH, 1 MEDIUM, 1 LOW") {
		t.Errorf("expected level count footer, got:\n%s", got)
	}
}

func TestRenderSummary_Empty(t *testing.T) {
	var buf bytes.Buffer
	output.RenderSummary(&buf, "/repo", "Analyzed .", nil)

	if !strings.Contains(buf.String(), "no recognized modules found") {
		t.Errorf("expected empty-result message, got:\n%s", buf.String())
	}
}

func TestRenderSummary_ColumnAlignmentUnaffectedByColor(t *testing.T) {
	// RenderSummary always writes to a bytes.Buffer here, which is never
	// a TTY, so colorEnabled is false and this asserts the plain-text
	// column widths stay fixed regardless of level length ("LOW" vs
	// "MEDIUM").
	var buf bytes.Buffer
	results := []output.InspectResult{
		{Impact: impact.Result{Target: "/repo/a.ts"}, Risk: risk.Score{Total: 90, Level: risk.LevelHigh}},
		{Impact: impact.Result{Target: "/repo/b.ts"}, Risk: risk.Score{Total: 10, Level: risk.LevelLow}},
	}

	output.RenderSummary(&buf, "/repo", "Analyzed .", results)
	lines := strings.Split(strings.TrimSpace(buf.String()), "\n")

	var dataLines []string
	for _, l := range lines {
		if strings.Contains(l, "/100") {
			dataLines = append(dataLines, l)
		}
	}
	if len(dataLines) != 2 {
		t.Fatalf("expected 2 data lines, got %d: %v", len(dataLines), dataLines)
	}
	idxA := strings.Index(dataLines[0], "/100")
	idxB := strings.Index(dataLines[1], "/100")
	if idxA != idxB {
		t.Errorf("expected aligned '/100' column across rows, got offsets %d and %d", idxA, idxB)
	}
}
