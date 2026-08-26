package output_test

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"

	"github.com/AlbertoBarrago/changeblast/internal/ci"
	"github.com/AlbertoBarrago/changeblast/internal/git"
	"github.com/AlbertoBarrago/changeblast/internal/impact"
	"github.com/AlbertoBarrago/changeblast/internal/output"
	"github.com/AlbertoBarrago/changeblast/internal/risk"
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
