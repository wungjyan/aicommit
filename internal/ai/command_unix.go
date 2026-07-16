//go:build unix

package ai

import (
	"os/exec"
	"syscall"
)

// configureProcessGroup starts the child in its own process group so signals can
// be delivered to the whole tree via the negative PID.
func configureProcessGroup(cmd *exec.Cmd) {
	if cmd.SysProcAttr == nil {
		cmd.SysProcAttr = &syscall.SysProcAttr{}
	}
	cmd.SysProcAttr.Setpgid = true
}

// terminateProcessTree kills the child's entire process group. Sending the
// signal to -pgid reaches the child and every descendant it spawned.
func terminateProcessTree(cmd *exec.Cmd) error {
	if cmd.Process == nil {
		return nil
	}
	pgid, err := syscall.Getpgid(cmd.Process.Pid)
	if err != nil {
		// Fall back to killing just the direct child.
		return cmd.Process.Kill()
	}
	return syscall.Kill(-pgid, syscall.SIGKILL)
}
