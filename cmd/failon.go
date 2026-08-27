package cmd

import (
	"fmt"
	"strings"

	"github.com/AlbertoBarrago/serval/internal/risk"
)

// validateFailOn checks a --fail-on value up front so an invalid value
// fails fast instead of after a full analysis run.
func validateFailOn(failOn string) error {
	if failOn == "" {
		return nil
	}
	if _, ok := riskLevelRank[risk.Level(normalizeLevel(failOn))]; !ok {
		return fmt.Errorf("invalid --fail-on value %q: must be one of low, medium, high", failOn)
	}
	return nil
}

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
	switch strings.ToLower(strings.TrimSpace(s)) {
	case "low":
		return string(risk.LevelLow)
	case "medium":
		return string(risk.LevelMedium)
	case "high":
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
