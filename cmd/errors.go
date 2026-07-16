package cmd

import (
	"errors"
	"fmt"
	"strings"

	"github.com/spf13/cobra"
	"github.com/wungjyan/aicommit/internal/ai"
)

var (
	// ErrUsage marks a command-usage error (bad flags, invalid values, or
	// unexpected positional arguments).
	ErrUsage = errors.New("usage error")

	// ErrInvalidMessage marks a generated message that failed Conventional
	// Commit validation in a non-interactive flow.
	ErrInvalidMessage = errors.New("invalid generated commit message")
)

const (
	ExitSuccess         = 0
	ExitGeneralError    = 1
	ExitUsageError      = 2
	ExitAIError         = 3
	ExitValidationError = 4
)

// usageErrorf builds a usage error wrapping ErrUsage.
func usageErrorf(format string, args ...any) error {
	return fmt.Errorf("%s: %w", fmt.Sprintf(format, args...), ErrUsage)
}

// noArgs is Cobra's standard positional-argument validator with the CLI's
// usage-error category attached for stable process exit codes.
func noArgs(cmd *cobra.Command, args []string) error {
	if err := cobra.NoArgs(cmd, args); err != nil {
		return fmt.Errorf("%w: %w", ErrUsage, err)
	}
	return nil
}

func invalidMessageError(err error) error {
	return fmt.Errorf("%w: %w", ErrInvalidMessage, err)
}

// ExitCode maps command errors to the documented process exit-code contract.
// It intentionally checks sentinel errors with errors.Is so wrapped provider
// diagnostics retain their original context while remaining classifiable.
func ExitCode(err error) int {
	if err == nil {
		return ExitSuccess
	}
	if errors.Is(err, ErrUsage) || isCobraUsageError(err) {
		return ExitUsageError
	}
	if errors.Is(err, ErrInvalidMessage) {
		return ExitValidationError
	}
	if isAIError(err) {
		return ExitAIError
	}
	return ExitGeneralError
}

func isAIError(err error) bool {
	for _, target := range []error{
		ai.ErrNotConfigured,
		ai.ErrAPIKeyInvalid,
		ai.ErrRateLimited,
		ai.ErrRequestFailed,
		ai.ErrUnknownBackend,
		ai.ErrCLINotInstalled,
		ai.ErrCLINotAuthenticated,
		ai.ErrEmptyMessage,
		ai.ErrCommandNotFound,
		ai.ErrCommandTimeout,
		ai.ErrCommandCanceled,
		ai.ErrCommandFailed,
	} {
		if errors.Is(err, target) {
			return true
		}
	}
	return false
}

// Cobra does not expose a sentinel for unknown-command parse errors. The
// command-local validators and flag error handler cover ordinary usage errors;
// these remaining parser messages are still user input mistakes and exit 2.
func isCobraUsageError(err error) bool {
	message := err.Error()
	return strings.HasPrefix(message, "unknown command ") || strings.Contains(message, "unknown flag:")
}
