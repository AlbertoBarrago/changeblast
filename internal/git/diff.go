package git

import (
	"path/filepath"
	"strings"
)

// ChangedFiles returns the absolute paths of files that differ between
// ref and the current working tree, including uncommitted changes to
// tracked files (`git diff --name-only <ref>`) and new untracked files
// (`git ls-files --others --exclude-standard`, which `git diff` does not
// report). Deleted files are included; callers that need an existing
// file on disk should filter separately.
func ChangedFiles(repoRoot, ref string) ([]string, error) {
	tracked, err := runGit(repoRoot, "diff", "--name-only", ref, "--")
	if err != nil {
		return nil, err
	}

	untracked, err := runGit(repoRoot, "ls-files", "--others", "--exclude-standard")
	if err != nil {
		return nil, err
	}

	seen := make(map[string]bool)
	var files []string
	for _, line := range append(splitNonEmptyLines(tracked), splitNonEmptyLines(untracked)...) {
		if seen[line] {
			continue
		}
		seen[line] = true
		files = append(files, filepath.Join(repoRoot, line))
	}
	return files, nil
}

func splitNonEmptyLines(s string) []string {
	trimmed := strings.TrimSpace(s)
	if trimmed == "" {
		return nil
	}
	return strings.Split(trimmed, "\n")
}
