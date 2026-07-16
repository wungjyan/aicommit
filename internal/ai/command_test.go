package ai

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"strings"
	"testing"
	"time"
)

// This test file drives the production execRunner against a fake executable:
// the test binary re-executes itself with GO_HELPER_MODE set, so no real CLI is
// ever spawned. See TestMain and helperCommand.

func TestMain(m *testing.M) {
	if mode := os.Getenv("GO_HELPER_MODE"); mode != "" {
		runHelper(mode)
		return
	}
	os.Exit(m.Run())
}

// runHelper implements the fake executable behaviors, selected by GO_HELPER_MODE.
func runHelper(mode string) {
	switch mode {
	case "echo-stdin":
		// Copy stdin to stdout — used to prove the diff arrives via stdin intact.
		data, _ := io.ReadAll(os.Stdin)
		fmt.Fprint(os.Stdout, string(data))
	case "echo-args":
		// Print each arg on its own line — proves argv is passed unaltered.
		for _, a := range os.Args[1:] {
			fmt.Fprintln(os.Stdout, a)
		}
	case "split-streams":
		fmt.Fprint(os.Stdout, "this is stdout")
		fmt.Fprint(os.Stderr, "this is stderr")
	case "fail":
		fmt.Fprint(os.Stderr, "boom on stderr")
		os.Exit(7)
	case "flood-stdout":
		// Emit far more than the output cap to exercise bounded buffering.
		chunk := strings.Repeat("x", 4096)
		for i := 0; i < 1024; i++ {
			fmt.Fprint(os.Stdout, chunk)
		}
	case "sleep":
		// Sleep long enough that timeout/cancel fires first.
		time.Sleep(30 * time.Second)
	case "spawn-child-sleep":
		// Spawn a grandchild that sleeps, then sleep ourselves. Used to verify
		// the whole process tree is terminated. The grandchild writes its PID to
		// the file named by GO_HELPER_PIDFILE before sleeping.
		pidFile := os.Getenv("GO_HELPER_PIDFILE")
		child := exec.Command(os.Args[0])
		child.Env = append(os.Environ(), "GO_HELPER_MODE=write-pid-then-sleep")
		_ = child.Start()
		if child.Process != nil && pidFile != "" {
			_ = os.WriteFile(pidFile, []byte(fmt.Sprintf("%d", child.Process.Pid)), 0600)
		}
		time.Sleep(30 * time.Second)
	case "write-pid-then-sleep":
		time.Sleep(30 * time.Second)
	}
	os.Exit(0)
}

// helperSpec builds a CommandSpec that re-executes the test binary in the given
// helper mode.
func helperSpec(mode string, args ...string) CommandSpec {
	return CommandSpec{
		Name: os.Args[0],
		Args: args,
		Env:  append(os.Environ(), "GO_HELPER_MODE="+mode),
	}
}

func TestRunPassesStdinIntact(t *testing.T) {
	// A diff full of shell metacharacters, quotes and newlines must survive
	// unchanged — proving it is not routed through a shell.
	diff := "diff --git a/x b/x\n+ rm -rf / ; echo \"$(whoami)\" `id` 'quoted'\n+ | & > <"
	spec := helperSpec("echo-stdin")
	spec.Stdin = diff

	res, err := NewCommandRunner().Run(context.Background(), spec)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if res.Stdout != diff {
		t.Errorf("stdin round-trip mismatch:\n got: %q\nwant: %q", res.Stdout, diff)
	}
}

func TestRunPassesArgsWithoutShell(t *testing.T) {
	// Metacharacters in args must be delivered literally, not interpreted.
	args := []string{"--flag", "a b", "$(echo hi)", "a;b|c"}
	res, err := NewCommandRunner().Run(context.Background(), helperSpec("echo-args", args...))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	got := strings.Split(strings.TrimRight(res.Stdout, "\n"), "\n")
	if len(got) != len(args) {
		t.Fatalf("got %d args back, want %d: %q", len(got), len(args), got)
	}
	for i := range args {
		if got[i] != args[i] {
			t.Errorf("arg %d = %q, want %q", i, got[i], args[i])
		}
	}
}

func TestRunSeparatesStdoutAndStderr(t *testing.T) {
	res, err := NewCommandRunner().Run(context.Background(), helperSpec("split-streams"))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if res.Stdout != "this is stdout" {
		t.Errorf("stdout = %q", res.Stdout)
	}
	if res.Stderr != "this is stderr" {
		t.Errorf("stderr = %q", res.Stderr)
	}
}

func TestRunNonZeroExit(t *testing.T) {
	res, err := NewCommandRunner().Run(context.Background(), helperSpec("fail"))
	if !errors.Is(err, ErrCommandFailed) {
		t.Fatalf("expected ErrCommandFailed, got %v", err)
	}
	if res.ExitCode != 7 {
		t.Errorf("exit code = %d, want 7", res.ExitCode)
	}
	// stderr is still captured for diagnostics.
	if !strings.Contains(res.Stderr, "boom on stderr") {
		t.Errorf("stderr = %q, want it to contain the failure output", res.Stderr)
	}
}

func TestRunMissingExecutable(t *testing.T) {
	spec := CommandSpec{Name: "definitely-not-a-real-binary-xyz"}
	_, err := NewCommandRunner().Run(context.Background(), spec)
	if !errors.Is(err, ErrCommandNotFound) {
		t.Errorf("expected ErrCommandNotFound, got %v", err)
	}
}

func TestRunBoundedOutput(t *testing.T) {
	spec := helperSpec("flood-stdout")
	spec.MaxOutputBytes = 64 * 1024
	res, err := NewCommandRunner().Run(context.Background(), spec)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !res.StdoutTruncated {
		t.Error("expected StdoutTruncated = true")
	}
	if len(res.Stdout) > spec.MaxOutputBytes {
		t.Errorf("captured %d bytes, exceeds cap %d", len(res.Stdout), spec.MaxOutputBytes)
	}
}

func TestRunTimeout(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()

	start := time.Now()
	_, err := NewCommandRunner().Run(ctx, helperSpec("sleep"))
	if !errors.Is(err, ErrCommandTimeout) {
		t.Fatalf("expected ErrCommandTimeout, got %v", err)
	}
	if elapsed := time.Since(start); elapsed > 10*time.Second {
		t.Errorf("timeout took too long: %s", elapsed)
	}
}

func TestRunCancel(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	go func() {
		time.Sleep(100 * time.Millisecond)
		cancel()
	}()

	_, err := NewCommandRunner().Run(ctx, helperSpec("sleep"))
	if !errors.Is(err, ErrCommandCanceled) {
		t.Fatalf("expected ErrCommandCanceled, got %v", err)
	}
}
