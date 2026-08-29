package output_test

import (
	"encoding/json"
	"testing"

	"github.com/AlbertoBarrago/serval/internal/impact"
	"github.com/AlbertoBarrago/serval/internal/output"
	"github.com/AlbertoBarrago/serval/internal/risk"
)

func TestToSARIF(t *testing.T) {
	results := []output.InspectResult{
		{
			Impact: impact.Result{Target: "/repo/src/auth/token.ts"},
			Risk: risk.Compute(risk.Input{
				TargetPath:           "src/auth/token.ts",
				DownstreamCount:      20,
				ChurnCount:           10,
				CriticalPathKeywords: risk.DefaultCriticalPathKeywords,
				NoCorrelatedTest:     true,
			}),
		},
		{
			Impact: impact.Result{Target: "/repo/src/util/log.ts"},
			Risk:   risk.Compute(risk.Input{}),
		},
	}

	log := output.ToSARIF("/repo", results)

	encoded, err := json.Marshal(log)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}

	var decoded map[string]interface{}
	if err := json.Unmarshal(encoded, &decoded); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}

	if decoded["version"] != "2.1.0" {
		t.Errorf("expected version 2.1.0, got %v", decoded["version"])
	}
	if decoded["$schema"] == "" || decoded["$schema"] == nil {
		t.Error("expected a non-empty $schema")
	}

	runs, ok := decoded["runs"].([]interface{})
	if !ok || len(runs) != 1 {
		t.Fatalf("expected exactly 1 run, got %+v", decoded["runs"])
	}
	run := runs[0].(map[string]interface{})

	tool := run["tool"].(map[string]interface{})
	driver := tool["driver"].(map[string]interface{})
	if driver["name"] != "serval" {
		t.Errorf("expected driver name \"serval\", got %v", driver["name"])
	}
	rules, ok := driver["rules"].([]interface{})
	if !ok || len(rules) != 1 {
		t.Fatalf("expected exactly 1 rule, got %+v", driver["rules"])
	}

	sarifResults, ok := run["results"].([]interface{})
	if !ok || len(sarifResults) != 2 {
		t.Fatalf("expected 2 results, got %+v", run["results"])
	}

	first := sarifResults[0].(map[string]interface{})
	if first["level"] != "error" {
		t.Errorf("expected level \"error\" for a HIGH-risk result, got %v", first["level"])
	}
	loc := first["locations"].([]interface{})[0].(map[string]interface{})
	physLoc := loc["physicalLocation"].(map[string]interface{})
	artifact := physLoc["artifactLocation"].(map[string]interface{})
	if artifact["uri"] != "src/auth/token.ts" {
		t.Errorf("expected uri \"src/auth/token.ts\", got %v", artifact["uri"])
	}

	second := sarifResults[1].(map[string]interface{})
	if second["level"] != "note" {
		t.Errorf("expected level \"note\" for a LOW-risk result, got %v", second["level"])
	}
}
