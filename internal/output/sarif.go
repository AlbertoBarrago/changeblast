package output

import (
	"fmt"
	"path/filepath"
	"strings"

	"github.com/AlbertoBarrago/serval/internal/risk"
)

// SARIF (Static Analysis Results Interchange Format) 2.1.0 log,
// consuming the same InspectResult every other renderer consumes — no
// new computation, pure formatting, same as RenderInspectFull/
// ToInspectFullJSON. Intended for CI code-scanning integrations (e.g.
// github/codeql-action/upload-sarif) that expect this shape rather than
// serval's own JSON.
//
// A single fixed rule (sarifRuleID) is used for every result: a
// distinct rule per risk-breakdown reason is a reasonable future
// extension, not attempted here.
const sarifRuleID = "serval/blast-radius"

type sarifLog struct {
	Schema  string     `json:"$schema"`
	Version string     `json:"version"`
	Runs    []sarifRun `json:"runs"`
}

type sarifRun struct {
	Tool    sarifTool     `json:"tool"`
	Results []sarifResult `json:"results"`
}

type sarifTool struct {
	Driver sarifDriver `json:"driver"`
}

type sarifDriver struct {
	Name  string      `json:"name"`
	Rules []sarifRule `json:"rules"`
}

type sarifRule struct {
	ID               string    `json:"id"`
	ShortDescription sarifText `json:"shortDescription"`
}

type sarifText struct {
	Text string `json:"text"`
}

type sarifResult struct {
	RuleID    string          `json:"ruleId"`
	Level     string          `json:"level"`
	Message   sarifText       `json:"message"`
	Locations []sarifLocation `json:"locations"`
}

type sarifLocation struct {
	PhysicalLocation sarifPhysicalLocation `json:"physicalLocation"`
}

type sarifPhysicalLocation struct {
	ArtifactLocation sarifArtifactLocation `json:"artifactLocation"`
}

type sarifArtifactLocation struct {
	URI string `json:"uri"`
}

// ToSARIF converts results into a SARIF 2.1.0 log with one result per
// analyzed target.
func ToSARIF(root string, results []InspectResult) sarifLog {
	sarifResults := make([]sarifResult, len(results))
	for i, r := range results {
		sarifResults[i] = sarifResult{
			RuleID:  sarifRuleID,
			Level:   sarifLevel(r.Risk.Level),
			Message: sarifText{Text: sarifMessage(r)},
			Locations: []sarifLocation{{
				PhysicalLocation: sarifPhysicalLocation{
					ArtifactLocation: sarifArtifactLocation{
						URI: filepath.ToSlash(relPath(root, r.Impact.Target)),
					},
				},
			}},
		}
	}

	return sarifLog{
		Schema:  "https://raw.githubusercontent.com/oasis-tcs/sarif-spec/master/Schemata/sarif-schema-2.1.0.json",
		Version: "2.1.0",
		Runs: []sarifRun{{
			Tool: sarifTool{Driver: sarifDriver{
				Name: "serval",
				Rules: []sarifRule{{
					ID:               sarifRuleID,
					ShortDescription: sarifText{Text: "Deterministic blast-radius risk score"},
				}},
			}},
			Results: sarifResults,
		}},
	}
}

// sarifLevel maps serval's risk level to a SARIF result level.
func sarifLevel(l risk.Level) string {
	switch l {
	case risk.LevelHigh:
		return "error"
	case risk.LevelMedium:
		return "warning"
	default:
		return "note"
	}
}

// sarifMessage renders the same risk total and breakdown shown in the
// text/JSON output as a single message string.
func sarifMessage(r InspectResult) string {
	var b strings.Builder
	fmt.Fprintf(&b, "Risk %s (%d/100)", r.Risk.Level, r.Risk.Total)
	for _, e := range r.Risk.Breakdown {
		fmt.Fprintf(&b, "; +%d %s", e.Points, e.Reason)
	}
	return b.String()
}
