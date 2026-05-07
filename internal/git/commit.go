package git

import (
	"fmt"
	"os/exec"
)

// Commit executes git commit with the given message.
func Commit(message string) error {
	cmd := exec.Command("git", "commit", "-m", message)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("git commit failed: %s\n%w", string(output), err)
	}

	fmt.Print(string(output))
	return nil
}
