package ai

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"os/exec"
	"strings"
	"time"
)

var (
	// ErrCommandNotFound indicates the executable could not be started (missing
	// binary or not on PATH).
	ErrCommandNotFound = errors.New("command not found")

	// ErrCommandTimeout indicates the command exceeded its deadline.
	ErrCommandTimeout = errors.New("command timed out")

	// ErrCommandCanceled indicates the command was canceled via context.
	ErrCommandCanceled = errors.New("command canceled")

	// ErrCommandFailed indicates the command ran but exited non-zero.
	ErrCommandFailed = errors.New("command failed")
)

// defaultMaxOutputBytes caps each captured stream so a misbehaving subprocess
// cannot exhaust memory with unbounded output.
const defaultMaxOutputBytes = 1 << 20 // 1 MiB

// killGraceDelay is how long the runner waits after a cancel signal before
// os/exec force-kills the direct child and closes its pipes.
const killGraceDelay = 3 * time.Second

// CommandSpec describes a subprocess invocation. The instruction/prompt is
// passed via Args (never through a shell) and untrusted input such as a diff is
// passed via Stdin, so no caller can inject shell syntax or hit argv length
// limits.
type CommandSpec struct {
	Name string   // executable name or path
	Args []string // arguments passed directly to the executable (no shell)

	Stdin string // written to the child's stdin, then stdin is closed

	Env []string // full environment; nil inherits the parent process environment

	// MaxOutputBytes caps each captured stream. Zero uses defaultMaxOutputBytes.
	MaxOutputBytes int
}

// CommandResult holds the captured output of a finished command.
type CommandResult struct {
	Stdout          string
	Stderr          string
	ExitCode        int
	StdoutTruncated bool
	StderrTruncated bool
}

// CommandRunner runs a subprocess to completion. It is injected into the local
// CLI providers so tests can substitute a fake and never spawn a real CLI.
type CommandRunner interface {
	Run(ctx context.Context, spec CommandSpec) (CommandResult, error)
}

// execRunner is the production CommandRunner backed by os/exec.
//
// It runs the child in its own process group (Unix) or job-controlled tree
// (Windows) so that a timeout or cancellation terminates the entire tree, not
// just the direct child. Output is captured into bounded buffers.
type execRunner struct{}

// NewCommandRunner returns the production CommandRunner.
func NewCommandRunner() CommandRunner { return execRunner{} }

func (execRunner) Run(ctx context.Context, spec CommandSpec) (CommandResult, error) {
	limit := spec.MaxOutputBytes
	if limit <= 0 {
		limit = defaultMaxOutputBytes
	}

	cmd := exec.CommandContext(ctx, spec.Name, spec.Args...)
	if spec.Env != nil {
		cmd.Env = spec.Env
	}
	if spec.Stdin != "" {
		cmd.Stdin = strings.NewReader(spec.Stdin)
	}

	stdout := &boundedBuffer{limit: limit}
	stderr := &boundedBuffer{limit: limit}
	cmd.Stdout = stdout
	cmd.Stderr = stderr

	// Run the child in its own process group / job so the whole tree can be
	// terminated. configureProcessGroup is defined per platform.
	configureProcessGroup(cmd)

	// On context cancellation, terminate the entire process tree rather than the
	// default single-process kill. WaitDelay forces a hard kill + pipe close if
	// the tree does not exit promptly.
	cmd.Cancel = func() error { return terminateProcessTree(cmd) }
	cmd.WaitDelay = killGraceDelay

	if err := cmd.Start(); err != nil {
		return CommandResult{}, fmt.Errorf("%w: %s: %v", ErrCommandNotFound, spec.Name, err)
	}

	waitErr := cmd.Wait()

	result := CommandResult{
		Stdout:          stdout.String(),
		Stderr:          stderr.String(),
		ExitCode:        cmd.ProcessState.ExitCode(),
		StdoutTruncated: stdout.truncated,
		StderrTruncated: stderr.truncated,
	}

	// Prefer context errors so timeout and cancellation are reported precisely,
	// even though Wait returns a generic kill/exit error in those cases.
	switch ctx.Err() {
	case context.DeadlineExceeded:
		return result, fmt.Errorf("%w after deadline: %s", ErrCommandTimeout, spec.Name)
	case context.Canceled:
		return result, fmt.Errorf("%w: %s", ErrCommandCanceled, spec.Name)
	}

	if waitErr != nil {
		var exitErr *exec.ExitError
		if errors.As(waitErr, &exitErr) {
			preview := outputPreview(result.Stderr)
			return result, fmt.Errorf("%w: %s exited with code %d: %s", ErrCommandFailed, spec.Name, result.ExitCode, preview)
		}
		// Non-exit errors (e.g. I/O) surface as a start/run failure.
		return result, fmt.Errorf("%w: %s: %v", ErrCommandNotFound, spec.Name, waitErr)
	}

	return result, nil
}

// outputPreview trims captured output to a short, single-line diagnostic.
func outputPreview(s string) string {
	s = strings.TrimSpace(s)
	s = strings.ReplaceAll(s, "\n", " ")
	const max = 200
	if len(s) > max {
		return s[:max] + "..."
	}
	return s
}

// boundedBuffer is an io.Writer that stops storing bytes once it reaches limit,
// recording that truncation occurred. It keeps consuming writes so the child is
// never blocked on a full pipe.
type boundedBuffer struct {
	buf       bytes.Buffer
	limit     int
	truncated bool
}

func (b *boundedBuffer) Write(p []byte) (int, error) {
	remaining := b.limit - b.buf.Len()
	if remaining <= 0 {
		b.truncated = true
		return len(p), nil
	}
	if len(p) > remaining {
		b.buf.Write(p[:remaining])
		b.truncated = true
		return len(p), nil
	}
	return b.buf.Write(p)
}

func (b *boundedBuffer) String() string { return b.buf.String() }

// ensure boundedBuffer satisfies io.Writer.
var _ io.Writer = (*boundedBuffer)(nil)
