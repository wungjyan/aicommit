package cmd

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/spf13/cobra"
	"github.com/wungjyan/aicommit/internal/ai"
	"github.com/wungjyan/aicommit/internal/config"
)

// newConfigCommand builds the `config` command tree: bare display plus the
// setup/set/check/path subcommands.
func newConfigCommand(deps Dependencies) *cobra.Command {
	var jsonOut bool

	configCmd := &cobra.Command{
		Use:   "config",
		Short: "Show or manage aicommit configuration",
		Long: `Show the effective configuration and its sources, or manage it with the
setup, set, check and path subcommands.`,
		Args: noArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runConfigDisplay(cmd, deps, jsonOut)
		},
	}
	configCmd.Flags().BoolVar(&jsonOut, "json", false, "output the configuration as JSON")

	configCmd.AddCommand(newConfigSetupCommand(deps))
	configCmd.AddCommand(newConfigSetCommand(deps))
	configCmd.AddCommand(newConfigCheckCommand(deps))
	configCmd.AddCommand(newConfigPathCommand(deps))

	return configCmd
}

// runConfigDisplay renders the effective configuration and its per-value
// sources. It never fails on CLI probe errors: install/auth degrade to
// informative strings while the command still exits 0.
func runConfigDisplay(cmd *cobra.Command, deps Dependencies, jsonOut bool) error {
	cfg, err := deps.Config.Load()
	if err != nil {
		return err
	}
	eff := config.Resolve(cfg)

	if jsonOut {
		return writeConfigJSON(cmd, deps, eff)
	}
	writeConfigText(cmd, deps, eff)
	return nil
}

func writeConfigText(cmd *cobra.Command, deps Dependencies, eff config.Effective) {
	out := cmd.OutOrStdout()
	fmt.Fprintf(out, "Backend  : %s  (%s)\n", eff.Backend.Value, eff.Backend.Source)

	if eff.IsOpenAI() {
		fmt.Fprintf(out, "API Key  : %s  (%s)\n", maskKey(eff.APIKey.Value), eff.APIKey.Source)
		fmt.Fprintf(out, "Base URL : %s  (%s)\n", eff.BaseURL.Value, eff.BaseURL.Source)
		fmt.Fprintf(out, "Model    : %s  (%s)\n", eff.Model.Value, eff.Model.Source)
	} else {
		st := deps.Backend.Status(config.Config{Backend: eff.Backend.Value})
		fmt.Fprintf(out, "CLI      : %s\n", cliPathText(st))
		fmt.Fprintf(out, "Auth     : %s\n", cliAuthText(st))
		fmt.Fprintf(out, "Model    : managed by %s CLI  (%s)\n", eff.Backend.Value, config.SourceCLI)
	}

	fmt.Fprintf(out, "Language : %s  (%s)\n", eff.Language.Value, eff.Language.Source)
}

// configJSON is the machine-readable shape of `config --json`. It never
// contains a raw API key (only the masked form) or any CLI token.
type configJSON struct {
	Backend  valueJSON  `json:"backend"`
	Language valueJSON  `json:"language"`
	APIKey   *valueJSON `json:"api_key,omitempty"`
	BaseURL  *valueJSON `json:"base_url,omitempty"`
	Model    *valueJSON `json:"model,omitempty"`
	CLI      *cliJSON   `json:"cli,omitempty"`
}

type valueJSON struct {
	Value  string `json:"value"`
	Source string `json:"source"`
}

type cliJSON struct {
	Installed bool   `json:"installed"`
	Path      string `json:"path,omitempty"`
	Auth      string `json:"auth,omitempty"`
}

func writeConfigJSON(cmd *cobra.Command, deps Dependencies, eff config.Effective) error {
	payload := configJSON{
		Backend:  valueJSON{eff.Backend.Value, string(eff.Backend.Source)},
		Language: valueJSON{eff.Language.Value, string(eff.Language.Source)},
	}

	if eff.IsOpenAI() {
		payload.APIKey = &valueJSON{maskKey(eff.APIKey.Value), string(eff.APIKey.Source)}
		payload.BaseURL = &valueJSON{eff.BaseURL.Value, string(eff.BaseURL.Source)}
		payload.Model = &valueJSON{eff.Model.Value, string(eff.Model.Source)}
	} else {
		st := deps.Backend.Status(config.Config{Backend: eff.Backend.Value})
		payload.CLI = &cliJSON{Installed: st.Installed, Path: st.Path, Auth: st.Auth}
		payload.Model = &valueJSON{"", string(config.SourceCLI)}
	}

	enc := json.NewEncoder(cmd.OutOrStdout())
	enc.SetIndent("", "  ")
	return enc.Encode(payload)
}

func cliPathText(st ai.CLIStatus) string {
	if !st.Installed {
		return "not installed"
	}
	return st.Path
}

func cliAuthText(st ai.CLIStatus) string {
	if !st.Installed {
		return "unavailable (CLI not installed)"
	}
	switch st.Auth {
	case ai.AuthAuthenticated:
		return "available (reported by CLI)"
	case ai.AuthUnauthenticated:
		return "not detected (custom provider may still work)"
	default:
		return "status unavailable (checked when generating)"
	}
}

// maskKey masks an API key, showing only a short prefix and suffix.
func maskKey(key string) string {
	if key == "" {
		return "(not set)"
	}
	if len(key) <= 8 {
		return strings.Repeat("*", len(key))
	}
	return key[:4] + strings.Repeat("*", len(key)-8) + key[len(key)-4:]
}
