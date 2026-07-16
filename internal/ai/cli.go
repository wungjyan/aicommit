package ai

import (
	"errors"
	"strings"
	"time"
)

// Shared sentinel errors for local CLI backends (Codex, Claude).
var (
	// ErrCLINotInstalled indicates the backend's CLI is not on PATH.
	ErrCLINotInstalled = errors.New("AI CLI not installed")

	// ErrCLINotAuthenticated indicates the backend's CLI has no valid login.
	ErrCLINotAuthenticated = errors.New("AI CLI not authenticated")

	// ErrEmptyMessage indicates the CLI produced no usable commit message.
	ErrEmptyMessage = errors.New("no commit message generated")
)

// cliAuthTimeout bounds a CLI authentication-status probe.
const cliAuthTimeout = 15 * time.Second

// trimMessage normalizes a model's raw output into a bare commit message,
// stripping surrounding whitespace and wrapping quotes/backticks.
func trimMessage(s string) string {
	s = strings.TrimSpace(s)
	s = strings.Trim(s, "`\"'")
	return strings.TrimSpace(s)
}
