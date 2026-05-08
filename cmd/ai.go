package cmd

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/spf13/cobra"
	"github.com/wungjyan/aicommit/internal/ai"
	"github.com/wungjyan/aicommit/internal/config"
	"github.com/wungjyan/aicommit/internal/ui"
)

type preset struct {
	name    string
	baseURL string
	model   string
}

var presets = []preset{
	{name: "OpenAI", baseURL: "https://api.openai.com/v1", model: "gpt-4o-mini"},
	{name: "DeepSeek", baseURL: "https://api.deepseek.com/v1", model: "deepseek-chat"},
	{name: "OpenRouter", baseURL: "https://openrouter.ai/api/v1", model: "openai/gpt-4o-mini"},
	{name: "Bailian", baseURL: "https://dashscope.aliyuncs.com/compatible-mode/v1", model: "qwen3.5-plus"},
	{name: "Custom (enter manually)", baseURL: "", model: ""},
}

var aiCmd = &cobra.Command{
	Use:   "ai",
	Short: "Configure AI provider settings",
	Long: `Configure your AI provider for commit message generation.

Without arguments, shows the current configuration.
Use --setup to run the interactive setup wizard.`,
	RunE: runAI,
}

var setupMode bool

func init() {
	aiCmd.Flags().BoolVar(&setupMode, "setup", false, "run interactive setup wizard")
	rootCmd.AddCommand(aiCmd)
}

func runAI(cmd *cobra.Command, args []string) error {
	cfg, _ := config.LoadConfig()

	if !setupMode {
		showConfig(cfg)
		fmt.Println()
		ui.Info("Run `aicommit ai --setup` to change these settings.")
		return nil
	}

	return interactiveSetup(cfg)
}

func showConfig(cfg config.Config) {
	masked := maskKey(cfg.APIKey)
	if masked == "" {
		masked = ui.Dim("(not set)")
	}
	baseURL := cfg.BaseURL
	if baseURL == "" {
		baseURL = ui.Dim("(default: https://api.openai.com/v1)")
	}
	model := cfg.Model
	if model == "" {
		model = ui.Dim("(default: gpt-4o-mini)")
	}
	fmt.Println(ui.Bold("Current AI configuration:"))
	fmt.Printf("  API Key  : %s\n", masked)
	fmt.Printf("  Base URL : %s\n", baseURL)
	fmt.Printf("  Model    : %s\n", model)
}

func interactiveSetup(cfg config.Config) error {
	reader := bufio.NewReader(os.Stdin)

	fmt.Println(ui.Bold("=== aicommit AI Setup ==="))
	fmt.Println()

	// Step 1: choose provider
	fmt.Println("Select AI provider:")
	for i, p := range presets {
		fmt.Printf("  %s) %s\n", ui.Highlight(fmt.Sprintf("%d", i+1)), p.name)
	}
	fmt.Printf("Choice %s: ", ui.Dim("[1]"))
	choiceStr, _ := reader.ReadString('\n')
	choiceStr = strings.TrimSpace(choiceStr)
	if choiceStr == "" {
		choiceStr = "1"
	}

	choice := 0
	for i := range presets {
		if choiceStr == fmt.Sprintf("%d", i+1) {
			choice = i
			break
		}
	}

	selected := presets[choice]
	fmt.Println()

	// Step 2: API key
	fmt.Printf("API Key %s: ", ui.Dim("["+maskOr(cfg.APIKey, "required")+"]"))
	apiKey, _ := reader.ReadString('\n')
	apiKey = strings.TrimSpace(apiKey)
	if apiKey == "" {
		apiKey = cfg.APIKey
	}

	// Step 3: base URL
	var defaultBase string
	if selected.baseURL != "" {
		// Preset: use the preset's URL.
		defaultBase = selected.baseURL
	} else {
		// Custom: always show a placeholder so the user knows the expected format.
		defaultBase = "https://your-provider.com/v1"
	}
	hint := defaultBase
	if selected.baseURL == "" && cfg.BaseURL != "" {
		// Show current saved value as extra context, but placeholder is still the default.
		hint = defaultBase + ui.Dim("  (current: "+cfg.BaseURL+")")
	}
	fmt.Printf("Base URL %s: ", ui.Dim("["+hint+"]"))
	baseURL, _ := reader.ReadString('\n')
	baseURL = strings.TrimSpace(baseURL)
	if baseURL == "" {
		if selected.baseURL != "" {
			baseURL = selected.baseURL
		} else if cfg.BaseURL != "" {
			baseURL = cfg.BaseURL
		} else {
			baseURL = defaultBase
		}
	}

	// Step 4: model
	defaultModel := selected.model
	if defaultModel == "" {
		if cfg.Model != "" {
			defaultModel = cfg.Model
		} else {
			defaultModel = "gpt-4o-mini"
		}
	}
	fmt.Printf("Model %s: ", ui.Dim("["+defaultModel+"]"))
	model, _ := reader.ReadString('\n')
	model = strings.TrimSpace(model)
	if model == "" {
		model = defaultModel
	}

	newCfg := config.Config{
		APIKey:  apiKey,
		BaseURL: baseURL,
		Model:   model,
	}

	// Step 5: verify before saving
	fmt.Println()
	var verifyErr error
	spinErr := ui.Spinner("Verifying connection", func() error {
		verifyErr = verifyConfig(newCfg)
		return verifyErr
	})
	_ = spinErr

	if verifyErr != nil {
		fmt.Println()
		ui.Error(verifyErr.Error())
		fmt.Println()
		ui.Warn("Configuration was NOT saved. Please check your settings and try again.")
		return nil
	}

	if err := config.SaveConfig(newCfg); err != nil {
		return fmt.Errorf("failed to save config: %w", err)
	}

	ui.Success("Configuration saved to ~/.aicommit/config.json")
	return nil
}

func verifyConfig(cfg config.Config) error {
	provider, err := ai.NewOpenAIProvider(cfg)
	if err != nil {
		return err
	}
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	return provider.Ping(ctx)
}

func maskKey(key string) string {
	if key == "" {
		return ""
	}
	if len(key) <= 8 {
		return strings.Repeat("*", len(key))
	}
	return key[:4] + strings.Repeat("*", len(key)-8) + key[len(key)-4:]
}

func maskOr(val, fallback string) string {
	if val == "" {
		return fallback
	}
	return maskKey(val)
}
