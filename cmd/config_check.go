package cmd

import (
	"context"
	"fmt"
	"time"

	"github.com/spf13/cobra"
	"github.com/wungjyan/aicommit/internal/config"
)

// configCheckTimeout bounds the whole verification.
const configCheckTimeout = 30 * time.Second

// newConfigCheckCommand verifies the effective backend prerequisites without
// mutating config. OpenAI checks connectivity; local CLI backends check only
// installation because their authentication/provider setup is CLI-owned.
func newConfigCheckCommand(deps Dependencies) *cobra.Command {
	return &cobra.Command{
		Use:   "check",
		Short: "Check the current backend prerequisites",
		Args:  noArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg, err := deps.Config.Load()
			if err != nil {
				return err
			}

			ctx, cancel := context.WithTimeout(cmd.Context(), configCheckTimeout)
			defer cancel()

			var checkErr error
			_ = deps.UI.Spinner(backendCheckLabel(cfg), func() error {
				checkErr = deps.Backend.Check(ctx, cfg)
				return checkErr
			})
			if checkErr != nil {
				return fmt.Errorf("configuration check failed: %w", checkErr)
			}

			deps.UI.Success(backendCheckSuccess(cfg))
			return nil
		},
	}
}

func backendCheckLabel(cfg config.Config) string {
	switch config.Resolve(cfg).Backend.Value {
	case config.BackendCodex:
		return "Checking Codex CLI installation"
	case config.BackendClaude:
		return "Checking Claude Code CLI installation"
	default:
		return "Verifying API configuration"
	}
}

func backendCheckSuccess(cfg config.Config) string {
	switch config.Resolve(cfg).Backend.Value {
	case config.BackendCodex:
		return "Codex CLI is installed. Official login and custom providers are both supported; authentication and connectivity will be checked when generating a commit message."
	case config.BackendClaude:
		return "Claude Code CLI is installed. Official login and custom providers are both supported; authentication and connectivity will be checked when generating a commit message."
	default:
		return "API configuration is valid and the connection succeeded."
	}
}

func cliBackendName(backend string) string {
	switch backend {
	case config.BackendCodex:
		return "Codex CLI"
	case config.BackendClaude:
		return "Claude Code CLI"
	default:
		return backend
	}
}
