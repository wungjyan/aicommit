package cmd

import (
	"context"
	"fmt"

	"github.com/spf13/cobra"
	"github.com/wungjyan/aicommit/internal/config"
)

// newConfigSetCommand provides a non-interactive, field-level config editor for
// scripts. At least one field is required. Existing fields are preserved unless
// explicitly overwritten. With --check, the resulting config is verified before
// saving; a failed verification never overwrites the existing config.
func newConfigSetCommand(deps Dependencies) *cobra.Command {
	var (
		apiKey   string
		baseURL  string
		model    string
		backend  string
		language string
		check    bool
	)

	cmd := &cobra.Command{
		Use:   "set",
		Short: "Update one or more configuration fields non-interactively",
		Args:  noArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			flags := cmd.Flags()
			changed := flags.Changed("api-key") || flags.Changed("base-url") ||
				flags.Changed("model") || flags.Changed("backend") || flags.Changed("language")
			if !changed {
				return usageErrorf("specify at least one field to set (--api-key, --base-url, --model, --backend, --language)")
			}

			if flags.Changed("backend") && !config.KnownBackend(backend) {
				return usageErrorf("invalid backend %q; expected one of: openai, codex, claude", backend)
			}

			cfg, err := deps.Config.Load()
			if err != nil {
				return err
			}

			// Apply only the explicitly-provided fields, preserving the rest.
			if flags.Changed("api-key") {
				cfg.APIKey = apiKey
			}
			if flags.Changed("base-url") {
				cfg.BaseURL = baseURL
			}
			if flags.Changed("model") {
				cfg.Model = model
			}
			if flags.Changed("backend") {
				cfg.Backend = backend
			}
			if flags.Changed("language") {
				cfg.Language = language
			}

			if check {
				ctx, cancel := context.WithTimeout(cmd.Context(), configCheckTimeout)
				defer cancel()
				var checkErr error
				_ = deps.UI.Spinner(backendCheckLabel(cfg), func() error {
					checkErr = deps.Backend.Check(ctx, cfg)
					return checkErr
				})
				if checkErr != nil {
					// Do not overwrite the existing config on a failed check.
					return fmt.Errorf("configuration check failed, config not saved: %w", checkErr)
				}
				if !config.Resolve(cfg).IsOpenAI() {
					deps.UI.Info(backendCheckSuccess(cfg))
				}
			}

			if err := deps.Config.Save(cfg); err != nil {
				return err
			}
			deps.UI.Success("Configuration saved.")
			return nil
		},
	}

	f := cmd.Flags()
	f.StringVar(&apiKey, "api-key", "", "OpenAI-compatible API key")
	f.StringVar(&baseURL, "base-url", "", "OpenAI-compatible base URL (ending in /v1)")
	f.StringVar(&model, "model", "", "model name")
	// Retain the flag for existing automation and future CLI-backend work, but do
	// not expose it until those backends meet the interactive latency target.
	f.StringVar(&backend, "backend", "", "AI backend")
	_ = f.MarkHidden("backend")
	f.StringVar(&language, "language", "", "commit message language")
	f.BoolVar(&check, "check", false, "verify API connectivity before saving")

	return cmd
}
