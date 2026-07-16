package cmd

import (
	"errors"
	"fmt"
	"testing"

	"github.com/wungjyan/aicommit/internal/ai"
)

func TestExitCode(t *testing.T) {
	tests := []struct {
		name string
		err  error
		want int
	}{
		{name: "success", want: ExitSuccess},
		{name: "usage", err: usageErrorf("bad flag"), want: ExitUsageError},
		{name: "unknown command", err: errors.New("unknown command \"bad\" for \"aicommit\""), want: ExitUsageError},
		{name: "invalid message", err: invalidMessageError(errors.New("invalid type")), want: ExitValidationError},
		{name: "not configured", err: fmt.Errorf("provider: %w", ai.ErrNotConfigured), want: ExitAIError},
		{name: "invalid key", err: ai.ErrAPIKeyInvalid, want: ExitAIError},
		{name: "rate limited", err: ai.ErrRateLimited, want: ExitAIError},
		{name: "request failed", err: ai.ErrRequestFailed, want: ExitAIError},
		{name: "CLI missing", err: ai.ErrCLINotInstalled, want: ExitAIError},
		{name: "CLI unauthenticated", err: ai.ErrCLINotAuthenticated, want: ExitAIError},
		{name: "command timeout", err: ai.ErrCommandTimeout, want: ExitAIError},
		{name: "general", err: errBoom, want: ExitGeneralError},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := ExitCode(tt.err); got != tt.want {
				t.Errorf("ExitCode(%v) = %d, want %d", tt.err, got, tt.want)
			}
		})
	}
}

func TestNoArgsReturnsUsageError(t *testing.T) {
	deps, _, _ := testDeps()
	err := execute(NewRootCommand(deps), "extra")
	if !errors.Is(err, ErrUsage) {
		t.Fatalf("error = %v, want ErrUsage", err)
	}
	if got := ExitCode(err); got != ExitUsageError {
		t.Errorf("ExitCode = %d, want %d", got, ExitUsageError)
	}
}
