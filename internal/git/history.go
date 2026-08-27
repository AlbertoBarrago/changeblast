// Package git extracts historical signals (churn, co-change frequency)
// for a file from the repository's Git history, by shelling out to the
// git binary. It performs no analysis beyond raw signal extraction —
// scoring those signals is the risk engine's job.
package git

import (
	"bytes"
	"fmt"
	"os/exec"
	"path/filepath"
	"strings"
)

// HistoryWindowDays and HistoryWindowMaxCommits define the bounded window
// over which all historical signals (churn, co-change frequency) are
// computed: the last HistoryWindowDays days, or the last
// HistoryWindowMaxCommits commits touching the file, whichever is
// smaller. This bounds cost on large/old repositories and must stay a
// named, documented constant rather than an unstated magic number — see
// docs/architecture.md. Overridable per-repository via .serval.yml
// (internal/config); see AnalyzeWithWindow.
const (
	HistoryWindowDays       = 90
	HistoryWindowMaxCommits = 200
)

// Window describes the history window signals were computed over, for
// inclusion in --json output and human-readable messaging.
type Window struct {
	Days       int `json:"days"`
	MaxCommits int `json:"maxCommits"`
}

// DefaultWindow is the Window value corresponding to the package
// constants above.
var DefaultWindow = Window{Days: HistoryWindowDays, MaxCommits: HistoryWindowMaxCommits}

// CoChange records how often another file changed in the same commit as
// the analyzed file, within the history window.
type CoChange struct {
	Path  string `json:"path"`
	Count int    `json:"count"`
}

// FileHistory is the historical signal set for a single file.
type FileHistory struct {
	Path      string     `json:"path"`
	Window    Window     `json:"historyWindow"`
	Changes   int        `json:"changes"`
	CoChanged []CoChange `json:"coChanged"`
}

// Analyze computes churn and co-change signals for path (an absolute
// path) within repoRoot, over DefaultWindow.
func Analyze(repoRoot, path string) (FileHistory, error) {
	return AnalyzeWithWindow(repoRoot, path, DefaultWindow)
}

// AnalyzeWithWindow computes churn and co-change signals for path (an
// absolute path) within repoRoot, over the given window. This is the
// entry point for a repository's .serval.yml `historyWindow`
// override (internal/config); Analyze is a thin wrapper over this with
// DefaultWindow.
func AnalyzeWithWindow(repoRoot, path string, window Window) (FileHistory, error) {
	rel, err := filepath.Rel(repoRoot, path)
	if err != nil {
		return FileHistory{}, err
	}
	rel = filepath.ToSlash(rel)

	commits, err := commitsTouching(repoRoot, rel, window)
	if err != nil {
		return FileHistory{}, err
	}

	coChanged, err := coChangedFiles(repoRoot, commits, rel)
	if err != nil {
		return FileHistory{}, err
	}

	return FileHistory{
		Path:      path,
		Window:    window,
		Changes:   len(commits),
		CoChanged: coChanged,
	}, nil
}

// commitsTouching returns commit hashes touching relPath, most recent
// first, bounded by window.
func commitsTouching(repoRoot, relPath string, window Window) ([]string, error) {
	since := fmt.Sprintf("%d.days.ago", window.Days)
	args := []string{
		"log",
		fmt.Sprintf("--since=%s", since),
		fmt.Sprintf("--max-count=%d", window.MaxCommits),
		"--format=%H",
		"--follow",
		"--",
		relPath,
	}

	out, err := runGit(repoRoot, args...)
	if err != nil {
		return nil, err
	}

	var hashes []string
	for _, line := range strings.Split(strings.TrimSpace(out), "\n") {
		if line == "" {
			continue
		}
		hashes = append(hashes, line)
	}
	return hashes, nil
}

// coChangedFiles tallies, across commits, which other files changed
// alongside relPath, and returns them sorted by descending frequency.
func coChangedFiles(repoRoot string, commits []string, relPath string) ([]CoChange, error) {
	counts := make(map[string]int)

	for _, hash := range commits {
		out, err := runGit(repoRoot, "show", "--name-only", "--format=", hash)
		if err != nil {
			return nil, err
		}

		for _, line := range strings.Split(strings.TrimSpace(out), "\n") {
			line = strings.TrimSpace(line)
			if line == "" || line == relPath {
				continue
			}
			counts[line]++
		}
	}

	result := make([]CoChange, 0, len(counts))
	for path, count := range counts {
		result = append(result, CoChange{Path: filepath.Join(repoRoot, path), Count: count})
	}
	sortCoChangesDesc(result)

	return result, nil
}

func sortCoChangesDesc(list []CoChange) {
	for i := 1; i < len(list); i++ {
		for j := i; j > 0 && (list[j].Count > list[j-1].Count ||
			(list[j].Count == list[j-1].Count && list[j].Path < list[j-1].Path)); j-- {
			list[j], list[j-1] = list[j-1], list[j]
		}
	}
}

func runGit(repoRoot string, args ...string) (string, error) {
	cmd := exec.Command("git", args...)
	cmd.Dir = repoRoot

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		return "", fmt.Errorf("git %s: %w: %s", strings.Join(args, " "), err, stderr.String())
	}
	return stdout.String(), nil
}
