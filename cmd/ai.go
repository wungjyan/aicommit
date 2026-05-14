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
	language := cfg.Language
	if language == "" || language == "English" {
		language = ui.Dim("English (default)")
	}
	fmt.Println(ui.Bold("Current AI configuration:"))
	fmt.Printf("  API Key  : %s\n", masked)
	fmt.Printf("  Base URL : %s\n", baseURL)
	fmt.Printf("  Model    : %s\n", model)
	fmt.Printf("  Language : %s\n", language)
}

func interactiveSetup(cfg config.Config) error {
	reader := bufio.NewReader(os.Stdin)

	fmt.Println(ui.Bold("=== aicommit AI Setup ==="))
	fmt.Println()

	fmt.Println("Select settings to configure:")
	fmt.Printf("  %s) %s %s\n", ui.Highlight("1"), "Model/API settings", ui.Dim("["+connectionSummary(cfg)+"]"))
	fmt.Printf("  %s) %s %s\n", ui.Highlight("2"), "Commit output language", ui.Dim("["+languageSummary(cfg)+"]"))
	fmt.Printf("Choice %s: ", ui.Dim("[1]"))
	choiceStr, _ := reader.ReadString('\n')
	choiceStr = strings.TrimSpace(choiceStr)
	if choiceStr == "" {
		choiceStr = "1"
	}

	fmt.Println()

	switch choiceStr {
	case "2":
		return setupLanguage(reader, cfg)
	default:
		return setupConnection(reader, cfg)
	}
}

func setupConnection(reader *bufio.Reader, cfg config.Config) error {
	defaultProviderChoice := detectProviderChoice(cfg)

	fmt.Println("Select AI provider:")
	for i, p := range presets {
		fmt.Printf("  %s) %s\n", ui.Highlight(fmt.Sprintf("%d", i+1)), p.name)
	}
	fmt.Printf("Choice %s: ", ui.Dim("["+defaultProviderChoice+"]"))
	choiceStr, _ := reader.ReadString('\n')
	choiceStr = strings.TrimSpace(choiceStr)
	if choiceStr == "" {
		choiceStr = defaultProviderChoice
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

	// Step 1: API key
	fmt.Printf("API Key %s: ", ui.Dim("["+maskOr(cfg.APIKey, "required")+"]"))
	apiKey, _ := reader.ReadString('\n')
	apiKey = strings.TrimSpace(apiKey)
	if apiKey == "" {
		apiKey = cfg.APIKey
	}

	// Step 2: base URL
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

	// Step 3: model
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
		APIKey:   apiKey,
		BaseURL:  baseURL,
		Model:    model,
		Language: cfg.Language,
	}

	// Step 4: verify before saving
	fmt.Println()
	fmt.Println(ui.Bold("Will save:"))
	fmt.Printf("  Provider : %s\n", providerName(selected, baseURL))
	fmt.Printf("  Base URL : %s\n", baseURL)
	fmt.Printf("  Model    : %s\n", model)
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
	fmt.Printf("Current model/API settings: %s\n", connectionSummary(newCfg))
	return nil
}

func setupLanguage(reader *bufio.Reader, cfg config.Config) error {
	languageOptions := []string{"English", "中文"}
	defaultLanguage := cfg.Language
	if defaultLanguage == "" {
		defaultLanguage = "English"
	}
	defaultLanguageChoice := "1"
	switch defaultLanguage {
	case "English":
		defaultLanguageChoice = "1"
	case "中文":
		defaultLanguageChoice = "2"
	default:
		defaultLanguageChoice = fmt.Sprintf("%d", len(languageOptions)+1)
	}

	fmt.Println("Select commit message language:")
	for i, lang := range languageOptions {
		fmt.Printf("  %s) %s\n", ui.Highlight(fmt.Sprintf("%d", i+1)), lang)
	}
	fmt.Printf("  %s) %s\n", ui.Highlight(fmt.Sprintf("%d", len(languageOptions)+1)), "Custom (enter manually)")
	fmt.Printf("Choice %s: ", ui.Dim("["+defaultLanguageChoice+"]"))
	langStr, _ := reader.ReadString('\n')
	langStr = strings.TrimSpace(langStr)
	if langStr == "" {
		langStr = defaultLanguageChoice
	}

	var language string
	switch langStr {
	case "1":
		language = "English"
	case "2":
		language = "中文"
	default:
		if langStr == fmt.Sprintf("%d", len(languageOptions)+1) {
			fmt.Printf("Language %s: ", ui.Dim("["+defaultLanguage+"]"))
			language, _ = reader.ReadString('\n')
			language = strings.TrimSpace(language)
		} else {
			language = langStr
		}
	}
	if language == "" {
		language = defaultLanguage
	}

	if language == cfg.Language {
		fmt.Println("Language unchanged — nothing to save.")
		return nil
	}

	newCfg := config.Config{
		APIKey:   cfg.APIKey,
		BaseURL:  cfg.BaseURL,
		Model:    cfg.Model,
		Language: language,
	}

	if err := config.SaveConfig(newCfg); err != nil {
		return fmt.Errorf("failed to save config: %w", err)
	}

	ui.Success("Commit output language saved to ~/.aicommit/config.json")
	fmt.Printf("Current output language: %s\n", languageSummary(newCfg))
	return nil
}

func connectionSummary(cfg config.Config) string {
	model := cfg.Model
	if model == "" {
		model = "gpt-4o-mini"
	}

	baseURL := cfg.BaseURL
	if baseURL == "" {
		baseURL = "default base URL"
	}

	return fmt.Sprintf("%s, %s", model, baseURL)
}

func languageSummary(cfg config.Config) string {
	if cfg.Language == "" {
		return "English"
	}
	return cfg.Language
}

func detectProviderChoice(cfg config.Config) string {
	baseURL := strings.TrimRight(cfg.BaseURL, "/")
	if baseURL == "" {
		return "1"
	}

	for i, p := range presets {
		if p.baseURL != "" && strings.TrimRight(p.baseURL, "/") == baseURL {
			return fmt.Sprintf("%d", i+1)
		}
	}

	return fmt.Sprintf("%d", len(presets))
}

func providerName(selected preset, baseURL string) string {
	if selected.baseURL == "" {
		if strings.TrimSpace(baseURL) == "" {
			return "Custom"
		}
		for _, p := range presets {
			if p.baseURL != "" && strings.TrimRight(p.baseURL, "/") == strings.TrimRight(baseURL, "/") {
				return p.name
			}
		}
		return "Custom"
	}

	return selected.name
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
