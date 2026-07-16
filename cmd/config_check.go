package cmd

import (
	"context"
	"fmt"
	"time"

	"github.com/spf13/cobra"
)

// configCheckTimeout bounds the whole verification.
const configCheckTimeout = 30 * time.Second

// newConfigCheckCommand verifies the effective AI configuration and connection
// without mutating the config. Human-readable status goes to stderr; the exit
// code reflects the result.
func newConfigCheckCommand(deps Dependencies) *cobra.Command {
	return &cobra.Command{
		Use:   "check",
		Short: "Verify the current AI configuration and connection",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg, err := deps.Config.Load()
			if err != nil {
				return err
			}

			ctx, cancel := context.WithTimeout(cmd.Context(), configCheckTimeout)
			defer cancel()

			var checkErr error
			_ = deps.UI.Spinner("Verifying configuration", func() error {
				checkErr = deps.Backend.Check(ctx, cfg)
				return checkErr
			})
			if checkErr != nil {
				return fmt.Errorf("configuration check failed: %w", checkErr)
			}

			deps.UI.Success("Configuration is valid.")
			return nil
		},
	}
}
