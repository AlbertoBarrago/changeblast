package cmd

import (
	"fmt"

	"github.com/AlbertoBarrago/serval/internal/risk"
)

// riskLevelRank orders risk.Level values for --fail-on threshold
// comparison. Shared by inspect and diff, since both gate on risk level.
var riskLevelRank = map[risk.Level]int{
	risk.LevelLow:    1,
	risk.LevelMedium: 2,
	risk.LevelHigh:   3,
}

// applyFailOn returns a failOnError (mapped to exit code 2 in Execute)
// when failOn is set and level meets or exceeds it.
func applyFailOn(failOn string, level risk.Level) error {
	if failOn == "" {
		return nil
	}

	threshold, ok := riskLevelRank[risk.Level(normalizeLevel(failOn))]
	if !ok {
		return fmt.Errorf("invalid --fail-on value %q: must be one of low, medium, high", failOn)
	}

	if riskLevelRank[level] >= threshold {
		return failOnError{level: level}
	}
	return nil
}

func normalizeLevel(s string) string {
	switch s {
	case "low", "LOW":
		return string(risk.LevelLow)
	case "medium", "MEDIUM":
		return string(risk.LevelMedium)
	case "high", "HIGH":
		return string(risk.LevelHigh)
	default:
		return s
	}
}

// failOnError signals that risk threshold gating (--fail-on) tripped.
// Execute() in root.go maps this to exit code 2, per the documented exit
// code contract.
type failOnError struct {
	level risk.Level
}

func (e failOnError) Error() string {
	return fmt.Sprintf("risk level %s meets or exceeds --fail-on threshold", e.level)
}
