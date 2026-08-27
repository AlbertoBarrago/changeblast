// Package risk computes a deterministic, explainable risk score for a
// change, from signals produced by the impact engine, Git analyzer, and
// CI analyzer. Every point on the score maps to a documented rule in the
// breakdown — there is no hidden or unexplained contribution.
package risk

import "fmt"

// Score thresholds, in points out of 100. Documented here rather than
// left as unstated magic numbers.
const (
	ThresholdHigh   = 60
	ThresholdMedium = 30
)

// Level is a coarse risk classification derived from Total.
type Level string

const (
	LevelLow    Level = "LOW"
	LevelMedium Level = "MEDIUM"
	LevelHigh   Level = "HIGH"
)

// Weights used by Compute. Named and documented so the scoring model is
// auditable, not tuned via inline literals.
const (
	// WeightPerDownstreamModule is awarded per direct+indirect impacted
	// module, capped at WeightDownstreamCap.
	WeightPerDownstreamModule = 2
	WeightDownstreamCap       = 28

	// WeightCriticalPath is a flat bonus when the target path matches a
	// critical-path keyword (see criticalpath.go).
	WeightCriticalPath = 20

	// WeightChurn tiers are applied based on churn count within the
	// history window; only the highest matching tier applies.
	WeightChurnHigh      = 14 // churn >= ChurnHighThreshold
	WeightChurnMedium    = 7  // churn >= ChurnMediumThreshold
	WeightChurnLow       = 3  // churn >= 1
	ChurnHighThreshold   = 7
	ChurnMediumThreshold = 3

	// WeightCoChange is awarded when at least CoChangeThreshold files are
	// frequently co-changed with the target.
	WeightCoChange    = 12
	CoChangeThreshold = 2

	// WeightCIImpact is awarded when at least one CI workflow is
	// relevant to the target.
	WeightCIImpact = 8
)

// Entry is a single, explained contribution to the total score.
type Entry struct {
	Points int    `json:"points"`
	Reason string `json:"reason"`
}

// Score is the full explainable risk result for a target.
type Score struct {
	Total     int     `json:"total"`
	Level     Level   `json:"level"`
	Breakdown []Entry `json:"breakdown"`
}

// Input is the signal set Compute scores. All fields are optional zero
// values (e.g. Input{} yields a LOW score with an empty breakdown).
type Input struct {
	// TargetPath is used only for critical-path keyword matching.
	TargetPath string
	// DownstreamCount is len(direct impact) + len(indirect impact).
	DownstreamCount int
	// ChurnCount is the number of historically significant changes to
	// the target within the history window (see internal/git).
	ChurnCount int
	// FrequentCoChangeCount is the number of files that co-changed with
	// the target at least CoChangeThreshold times within the window.
	FrequentCoChangeCount int
	// RelevantWorkflows are the names of CI workflows relevant to the
	// target (see internal/ci).
	RelevantWorkflows []string
	// CriticalPathKeywords is the keyword list MatchCriticalPath checks
	// TargetPath against. Callers pass DefaultCriticalPathKeywords, or a
	// repository's .blast.yml `criticalPaths` override
	// (internal/config). A nil/empty value disables critical-path
	// matching entirely rather than falling back silently.
	CriticalPathKeywords []string
}

// Compute produces a deterministic, explained Score from input. The same
// Input always yields the same Score.
func Compute(input Input) Score {
	var entries []Entry
	total := 0

	if input.DownstreamCount > 0 {
		points := input.DownstreamCount * WeightPerDownstreamModule
		if points > WeightDownstreamCap {
			points = WeightDownstreamCap
		}
		total += points
		entries = append(entries, Entry{
			Points: points,
			Reason: fmt.Sprintf("%d downstream modules", input.DownstreamCount),
		})
	}

	if keyword, matched := MatchCriticalPath(input.TargetPath, input.CriticalPathKeywords); matched {
		total += WeightCriticalPath
		entries = append(entries, Entry{
			Points: WeightCriticalPath,
			Reason: fmt.Sprintf("critical path (matched %q in %s)", keyword, input.TargetPath),
		})
	}

	switch {
	case input.ChurnCount >= ChurnHighThreshold:
		total += WeightChurnHigh
		entries = append(entries, Entry{
			Points: WeightChurnHigh,
			Reason: fmt.Sprintf("high historical churn (%d changes)", input.ChurnCount),
		})
	case input.ChurnCount >= ChurnMediumThreshold:
		total += WeightChurnMedium
		entries = append(entries, Entry{
			Points: WeightChurnMedium,
			Reason: fmt.Sprintf("moderate historical churn (%d changes)", input.ChurnCount),
		})
	case input.ChurnCount >= 1:
		total += WeightChurnLow
		entries = append(entries, Entry{
			Points: WeightChurnLow,
			Reason: fmt.Sprintf("low historical churn (%d changes)", input.ChurnCount),
		})
	}

	if input.FrequentCoChangeCount >= CoChangeThreshold {
		total += WeightCoChange
		entries = append(entries, Entry{
			Points: WeightCoChange,
			Reason: fmt.Sprintf("%d frequently co-changed modules", input.FrequentCoChangeCount),
		})
	}

	if len(input.RelevantWorkflows) > 0 {
		total += WeightCIImpact
		noun := "workflow"
		if len(input.RelevantWorkflows) != 1 {
			noun = "workflows"
		}
		entries = append(entries, Entry{
			Points: WeightCIImpact,
			Reason: fmt.Sprintf("%d CI %s affected", len(input.RelevantWorkflows), noun),
		})
	}

	if total > 100 {
		total = 100
	}

	return Score{
		Total:     total,
		Level:     levelFor(total),
		Breakdown: entries,
	}
}

func levelFor(total int) Level {
	switch {
	case total >= ThresholdHigh:
		return LevelHigh
	case total >= ThresholdMedium:
		return LevelMedium
	default:
		return LevelLow
	}
}
