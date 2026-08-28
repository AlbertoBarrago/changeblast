package risk

import (
	"fmt"
	"strings"
)

// Set-level weights, same auditability contract as the per-file weights
// in risk.go: named, documented, every point traceable to a rule.
const (
	// WeightSetInteraction is awarded per dependency edge between two
	// changed files (the files' risks interact rather than being
	// independent), capped at WeightSetInteractionCap.
	WeightSetInteraction    = 6
	WeightSetInteractionCap = 18

	// WeightSharedDependents is a flat bonus when at least one unchanged
	// module directly imports two or more changed files.
	WeightSharedDependents = 8
)

// SetInput is the signal set ComputeSet scores for a whole change set
// (e.g. every changed file in a diff). It deliberately mirrors Input but
// differs in three documented ways that prevent double-counting:
//
//   - DownstreamCount is the size of the deduplicated direct+indirect
//     union across targets — a module reachable from three changed files
//     counts once (see internal/impact.ComputeSet).
//   - ChurnCount is the maximum per-target churn, not the sum: churning
//     files changed together does not multiply their history signal.
//   - A critical-path keyword match on any single target is enough; the
//     bonus is not awarded per match.
type SetInput struct {
	// TargetPaths are the change-set files, used for critical-path
	// keyword matching.
	TargetPaths []string
	// DownstreamCount is len(direct union) + len(indirect union).
	DownstreamCount int
	// InternalEdgeCount is the number of dependency edges whose endpoints
	// are both in the change set.
	InternalEdgeCount int
	// SharedDependentCount is the number of unchanged modules directly
	// importing at least two changed files.
	SharedDependentCount int
	// ChurnCount is the maximum churn across the targets.
	ChurnCount int
	// RelevantWorkflows is the deduplicated union of workflow names
	// relevant to any target.
	RelevantWorkflows []string
	// CriticalPathKeywords behaves as in Input.
	CriticalPathKeywords []string
	// HighRiskPaths behaves as in Input: a match on any single target
	// is enough to floor the resulting Level at LevelHigh.
	HighRiskPaths []string
}

// ComputeSet produces a deterministic, explained Score for a change set.
// The same SetInput always yields the same Score.
func ComputeSet(input SetInput) Score {
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
			Reason: fmt.Sprintf("%d downstream modules (deduplicated across the change set)", input.DownstreamCount),
		})
	}

	if input.InternalEdgeCount > 0 {
		points := input.InternalEdgeCount * WeightSetInteraction
		if points > WeightSetInteractionCap {
			points = WeightSetInteractionCap
		}
		total += points
		entries = append(entries, Entry{
			Points: points,
			Reason: fmt.Sprintf("%d dependency edge(s) between changed files", input.InternalEdgeCount),
		})
	}

	if input.SharedDependentCount > 0 {
		total += WeightSharedDependents
		verb := "imports"
		noun := "module"
		if input.SharedDependentCount != 1 {
			verb = "import"
			noun = "modules"
		}
		entries = append(entries, Entry{
			Points: WeightSharedDependents,
			Reason: fmt.Sprintf("%d unchanged %s %s multiple changed files", input.SharedDependentCount, noun, verb),
		})
	}

	if len(input.TargetPaths) > 0 {
		var matches []string
		for _, target := range input.TargetPaths {
			if keyword, matched := MatchCriticalPath(target, input.CriticalPathKeywords); matched {
				matches = append(matches, fmt.Sprintf("%q in %s", keyword, target))
			}
		}
		if len(matches) > 0 {
			total += WeightCriticalPath
			entries = append(entries, Entry{
				Points: WeightCriticalPath,
				Reason: fmt.Sprintf("critical path (%s)", strings.Join(matches, ", ")),
			})
		}
	}

	switch {
	case input.ChurnCount >= ChurnHighThreshold:
		total += WeightChurnHigh
		entries = append(entries, Entry{
			Points: WeightChurnHigh,
			Reason: fmt.Sprintf("high historical churn (%d changes, max across the set)", input.ChurnCount),
		})
	case input.ChurnCount >= ChurnMediumThreshold:
		total += WeightChurnMedium
		entries = append(entries, Entry{
			Points: WeightChurnMedium,
			Reason: fmt.Sprintf("moderate historical churn (%d changes, max across the set)", input.ChurnCount),
		})
	case input.ChurnCount >= 1:
		total += WeightChurnLow
		entries = append(entries, Entry{
			Points: WeightChurnLow,
			Reason: fmt.Sprintf("low historical churn (%d changes, max across the set)", input.ChurnCount),
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
			Reason: fmt.Sprintf("%d CI %s affected (union across the set)", len(input.RelevantWorkflows), noun),
		})
	}

	if total > 100 {
		// Unreachable with the current weights (their maximum is below
		// 100), but kept as the same defensive guarantee Compute gives:
		// if weights ever change to allow overflow, the breakdown still
		// sums to the clamped total.
		overflow := total - 100
		total = 100
		for i := len(entries) - 1; i >= 0 && overflow > 0; i-- {
			deduct := min(entries[i].Points, overflow)
			entries[i].Points -= deduct
			overflow -= deduct
		}
		filtered := entries[:0]
		for _, e := range entries {
			if e.Points > 0 {
				filtered = append(filtered, e)
			}
		}
		entries = filtered
	}

	score := Score{
		Total:     total,
		Level:     levelFor(total),
		Breakdown: entries,
	}

	var forcedMatches []string
	for _, target := range input.TargetPaths {
		if pattern, matched := MatchHighRiskPath(target, input.HighRiskPaths); matched {
			forcedMatches = append(forcedMatches, fmt.Sprintf("%q in %s", pattern, target))
		}
	}
	if len(forcedMatches) > 0 {
		score.Forced = true
		score.ForcedReason = fmt.Sprintf("high-risk path (%s)", strings.Join(forcedMatches, ", "))
		score.Level = LevelHigh
	}

	return score
}
