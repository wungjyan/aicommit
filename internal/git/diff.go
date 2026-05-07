package git

import (
	"fmt"
	"os/exec"
	"strings"
)

// GetStagedDiff returns the staged changes (git diff --cached).
// Returns empty string and nil error if there are no staged changes.
func GetStagedDiff() (string, error) {
	cmd := exec.Command("git", "diff", "--cached")
	output, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("failed to get staged diff: %w", err)
	}

	diff := strings.TrimSpace(string(output))
	if diff == "" {
		return "", nil
	}

	return diff, nil
}
