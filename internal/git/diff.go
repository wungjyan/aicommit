package git

import (
	"errors"
	"fmt"
	"os/exec"
	"strings"
)

var ErrNotGitRepo = errors.New("not a git repository (or any of the parent directories)")
var ErrNoStagedChanges = errors.New("no staged changes found; use `git add` to stage your changes first")

// maxDiffBytes is the default max size for a diff (80KB, roughly 20K tokens).
const maxDiffBytes = 80 * 1024

// IsGitRepo checks whether the current directory is inside a git work tree.
func IsGitRepo() error {
	cmd := exec.Command("git", "rev-parse", "--is-inside-work-tree")
	if err := cmd.Run(); err != nil {
		return ErrNotGitRepo
	}
	return nil
}

// GetStagedDiff returns the staged changes (git diff --cached).
// Returns ErrNoStagedChanges if there are no staged changes.
func GetStagedDiff() (string, error) {
	cmd := exec.Command("git", "diff", "--cached")
	output, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("failed to get staged diff: %w", err)
	}

	diff := strings.TrimSpace(string(output))
	if diff == "" {
		return "", ErrNoStagedChanges
	}

	return diff, nil
}

// TruncateDiff truncates a diff to fit within maxBytes.
// It keeps the beginning of the diff and appends a notice.
// If the diff is already within the limit, it is returned as-is.
func TruncateDiff(diff string, maxBytes int) string {
	if maxBytes <= 0 {
		maxBytes = maxDiffBytes
	}

	if len(diff) <= maxBytes {
		return diff
	}

	// We want to leave room for the truncation notice.
	notice := "\n\n... [diff truncated — too many changes to display in full] ..."
	cutoff := maxBytes - len(notice)
	if cutoff < 0 {
		return notice
	}

	// Try to cut at a newline boundary so we don't split a line.
	truncated := diff[:cutoff]
	if idx := strings.LastIndex(truncated, "\n"); idx > cutoff/2 {
		truncated = truncated[:idx]
	}

	return truncated + notice
}
