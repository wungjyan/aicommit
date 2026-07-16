//go:build windows

package ai

import (
	"os/exec"
	"strconv"
)

// configureProcessGroup is a no-op on Windows; process-tree termination is done
// via taskkill in terminateProcessTree.
func configureProcessGroup(cmd *exec.Cmd) {}

// terminateProcessTree kills the child and all of its descendants using
// taskkill /T, which walks the process tree rooted at the child PID.
func terminateProcessTree(cmd *exec.Cmd) error {
	if cmd.Process == nil {
		return nil
	}
	kill := exec.Command("taskkill", "/F", "/T", "/PID", strconv.Itoa(cmd.Process.Pid))
	if err := kill.Run(); err != nil {
		// Fall back to killing just the direct child.
		return cmd.Process.Kill()
	}
	return nil
}
