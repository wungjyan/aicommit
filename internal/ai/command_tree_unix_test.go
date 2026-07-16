//go:build unix

package ai

import (
	"context"
	"fmt"
	"os"
	"syscall"
	"testing"
	"time"
)

// TestRunTerminatesProcessTree verifies that cancelling the parent kills the
// grandchild too, not just the direct child. This relies on POSIX signal-0
// liveness probing, so it is Unix-only; Windows tree termination is covered by
// the taskkill path and CI on that platform.
func TestRunTerminatesProcessTree(t *testing.T) {
	pidFile := t.TempDir() + "/grandchild.pid"

	spec := helperSpec("spawn-child-sleep")
	spec.Env = append(spec.Env, "GO_HELPER_PIDFILE="+pidFile)

	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan struct{})
	go func() {
		_, _ = NewCommandRunner().Run(ctx, spec)
		close(done)
	}()

	grandchildPid := waitForPidFile(t, pidFile)

	cancel()
	<-done

	if pidAliveWithin(grandchildPid, 3*time.Second) {
		t.Errorf("grandchild pid %d still alive after cancel; process tree not terminated", grandchildPid)
	}
}

func waitForPidFile(t *testing.T, path string) int {
	t.Helper()
	deadline := time.Now().Add(5 * time.Second)
	for time.Now().Before(deadline) {
		data, err := os.ReadFile(path)
		if err == nil && len(data) > 0 {
			var pid int
			fmt.Sscanf(string(data), "%d", &pid)
			if pid > 0 {
				return pid
			}
		}
		time.Sleep(20 * time.Millisecond)
	}
	t.Fatal("timed out waiting for grandchild pid file")
	return 0
}

// pidAliveWithin returns false as soon as the process is gone, else true if it
// is still alive when the timeout elapses.
func pidAliveWithin(pid int, timeout time.Duration) bool {
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		if !pidAlive(pid) {
			return false
		}
		time.Sleep(50 * time.Millisecond)
	}
	return pidAlive(pid)
}

// pidAlive reports whether a process with the given pid currently exists.
// Signal 0 performs error checking without actually delivering a signal.
func pidAlive(pid int) bool {
	return syscall.Kill(pid, 0) == nil
}
