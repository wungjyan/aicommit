package cmd

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/spf13/cobra"
	"github.com/wungjyan/aicommit/internal/config"
	"golang.org/x/term"
)

// preset is an OpenAI-compatible provider preset that pre-fills base URL/model.
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

const setupVerifyTimeout = 15 * time.Second

// newConfigSetupCommand runs the interactive setup wizard, organized into two
// bounded groups: AI backend settings and commit output language.
func newConfigSetupCommand(deps Dependencies) *cobra.Command {
	return &cobra.Command{
		Use:   "setup",
		Short: "Run the interactive configuration wizard",
		Args:  noArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runConfigSetup(cmd, deps)
		},
	}
}

func runConfigSetup(cmd *cobra.Command, deps Dependencies) error {
	w := newWizard(deps)

	cfg, err := deps.Config.Load()
	if err != nil {
		return err
	}

	w.line("=== aicommit setup ===")
	w.line("")
	w.line("Select settings to configure:")
	w.line("  1) AI backend settings")
	w.line("  2) Commit output language")
	w.line("  3) All settings")

	choice := w.prompt("Choice [1]: ", "1")
	switch choice {
	case "1":
		return setupBackendOnly(cmd, deps, w, cfg)
	case "2":
		return setupLanguageOnly(deps, w, cfg)
	case "3":
		return setupAll(cmd, deps, w, cfg)
	default:
		return usageErrorf("invalid choice %q", choice)
	}
}

// setupBackendOnly configures and verifies the AI backend, preserving Language.
func setupBackendOnly(cmd *cobra.Command, deps Dependencies, w *wizard, cfg config.Config) error {
	newCfg, err := configureBackend(cmd, deps, w, cfg)
	if err != nil {
		return err
	}
	newCfg.Language = cfg.Language
	if err := deps.Config.Save(newCfg); err != nil {
		return err
	}
	deps.UI.Success("AI backend settings saved.")
	return nil
}

// setupLanguageOnly changes only the Language, with no connection test.
func setupLanguageOnly(deps Dependencies, w *wizard, cfg config.Config) error {
	language := chooseLanguage(w, cfg.Language)
	cfg.Language = language
	if err := deps.Config.Save(cfg); err != nil {
		return err
	}
	deps.UI.Success("Commit output language saved.")
	return nil
}

// setupAll configures the backend and language, verifies the backend, and only
// persists everything once both steps succeed.
func setupAll(cmd *cobra.Command, deps Dependencies, w *wizard, cfg config.Config) error {
	newCfg, err := configureBackend(cmd, deps, w, cfg)
	if err != nil {
		return err
	}
	newCfg.Language = chooseLanguage(w, cfg.Language)
	if err := deps.Config.Save(newCfg); err != nil {
		return err
	}
	deps.UI.Success("Configuration saved.")
	return nil
}

// configureBackend selects a backend, gathers its settings and verifies them.
// It returns the new config (Language unset) and never persists on its own.
func configureBackend(cmd *cobra.Command, deps Dependencies, w *wizard, cfg config.Config) (config.Config, error) {
	w.line("")
	w.line("Select AI backend:")
	w.line("  1) OpenAI-compatible API")
	w.line("  2) Codex CLI")
	w.line("  3) Claude Code CLI")

	choice := w.prompt("Choice [1]: ", "1")
	switch choice {
	case "1":
		return configureOpenAI(cmd, deps, w, cfg)
	case "2":
		return configureCLIBackend(cmd, deps, w, cfg, config.BackendCodex)
	case "3":
		return configureCLIBackend(cmd, deps, w, cfg, config.BackendClaude)
	default:
		return config.Config{}, usageErrorf("invalid backend choice %q", choice)
	}
}

func configureOpenAI(cmd *cobra.Command, deps Dependencies, w *wizard, cfg config.Config) (config.Config, error) {
	w.line("")
	w.line("Select provider preset:")
	for i, p := range presets {
		w.line(fmt.Sprintf("  %d) %s", i+1, p.name))
	}
	sel := w.chooseIndex("Choice [1]: ", len(presets), 0)
	selected := presets[sel]

	apiKey := w.promptSecret("API Key: ", cfg.APIKey)

	defaultBase := selected.baseURL
	if defaultBase == "" {
		defaultBase = cfg.BaseURL
	}
	baseURL := w.prompt(fmt.Sprintf("Base URL [%s]: ", orPlaceholder(defaultBase, "https://your-provider.com/v1")), defaultBase)

	defaultModel := selected.model
	if defaultModel == "" {
		defaultModel = cfg.Model
	}
	if defaultModel == "" {
		defaultModel = config.DefaultModel
	}
	model := w.prompt(fmt.Sprintf("Model [%s]: ", defaultModel), defaultModel)

	newCfg := cfg
	newCfg.Backend = config.BackendOpenAI
	newCfg.APIKey = apiKey
	newCfg.BaseURL = baseURL
	newCfg.Model = model

	if err := verifyBackend(cmd, deps, newCfg); err != nil {
		return config.Config{}, err
	}
	return newCfg, nil
}

// configureCLIBackend selects a local CLI backend and verifies its auth. It
// keeps existing OpenAI fields for later reuse but they will not be shown or run.
func configureCLIBackend(cmd *cobra.Command, deps Dependencies, w *wizard, cfg config.Config, backend string) (config.Config, error) {
	newCfg := cfg
	newCfg.Backend = backend

	w.line("")
	w.line(fmt.Sprintf("Verifying %s CLI authentication...", backend))
	if err := verifyBackend(cmd, deps, config.Config{Backend: backend, Language: cfg.Language}); err != nil {
		return config.Config{}, err
	}
	return newCfg, nil
}

// verifyBackend runs the backend check with a bounded timeout, reporting via UI.
func verifyBackend(cmd *cobra.Command, deps Dependencies, cfg config.Config) error {
	ctx, cancel := context.WithTimeout(cmd.Context(), setupVerifyTimeout)
	defer cancel()

	var checkErr error
	_ = deps.UI.Spinner("Verifying", func() error {
		checkErr = deps.Backend.Check(ctx, cfg)
		return checkErr
	})
	if checkErr != nil {
		deps.UI.Warn("Configuration was NOT saved.")
		return fmt.Errorf("verification failed: %w", checkErr)
	}
	return nil
}

// chooseLanguage prompts for the commit output language.
func chooseLanguage(w *wizard, current string) string {
	w.line("")
	w.line("Select commit message language:")
	w.line("  1) English")
	w.line("  2) 中文")
	w.line("  3) Custom (enter manually)")

	choice := w.prompt("Choice [1]: ", "1")
	switch choice {
	case "1":
		return "English"
	case "2":
		return "中文"
	case "3":
		return w.prompt("Language: ", current)
	default:
		// Treat any other non-empty input as the language literal.
		if choice != "" {
			return choice
		}
		return "English"
	}
}

func orPlaceholder(v, placeholder string) string {
	if strings.TrimSpace(v) == "" {
		return placeholder
	}
	return v
}

// wizard reads user input from deps.In and writes prompts to deps.ErrOut, so
// stdout is never polluted by interactive text.
type wizard struct {
	deps   Dependencies
	reader *bufio.Reader
}

func newWizard(deps Dependencies) *wizard {
	return &wizard{deps: deps, reader: bufio.NewReader(deps.In)}
}

// line writes an informational line to the diagnostic stream.
func (w *wizard) line(s string) {
	fmt.Fprintln(w.deps.ErrOut, s)
}

// prompt writes label to stderr and reads one line; empty input yields def.
func (w *wizard) prompt(label, def string) string {
	fmt.Fprint(w.deps.ErrOut, label)
	text, _ := w.reader.ReadString('\n')
	text = strings.TrimSpace(text)
	if text == "" {
		return def
	}
	return text
}

// chooseIndex reads a 1-based menu choice, looping until it is in range; empty
// input selects the zero-based defIdx.
func (w *wizard) chooseIndex(label string, n, defIdx int) int {
	for {
		s := w.prompt(label, fmt.Sprintf("%d", defIdx+1))
		var i int
		if _, err := fmt.Sscanf(s, "%d", &i); err == nil && i >= 1 && i <= n {
			return i - 1
		}
		w.line(fmt.Sprintf("Please enter a number between 1 and %d.", n))
	}
}

// promptSecret reads a secret without echoing when In is a real terminal.
// When In is not a terminal (tests, pipes), it falls back to a plain read.
func (w *wizard) promptSecret(label, def string) string {
	fmt.Fprint(w.deps.ErrOut, label)
	if f, ok := w.deps.In.(*os.File); ok && term.IsTerminal(int(f.Fd())) {
		b, err := term.ReadPassword(int(f.Fd()))
		fmt.Fprintln(w.deps.ErrOut)
		if err == nil {
			s := strings.TrimSpace(string(b))
			if s == "" {
				return def
			}
			return s
		}
	}
	text, _ := w.reader.ReadString('\n')
	text = strings.TrimSpace(text)
	if text == "" {
		return def
	}
	return text
}
