package cmd

import (
	"bytes"
	"strings"
	"testing"

	"github.com/wungjyan/aicommit/internal/config"
)

// runSetup drives `config setup` with the given stdin script.
func runSetup(t *testing.T, deps Dependencies, stdin string) error {
	t.Helper()
	deps.In = strings.NewReader(stdin)
	cmd := NewRootCommand(deps)
	return execute(cmd, "config", "setup")
}

// Entry point 2: language only — must preserve the existing backend config.
func TestSetupLanguageOnlyPreservesBackend(t *testing.T) {
	deps, _, _ := testDeps()
	fc := &fakeConfig{cfg: config.Config{Backend: "openai", APIKey: "sk-keep", Model: "m"}}
	deps.Config = fc

	// "2" = language group, then "2" = 中文.
	if err := runSetup(t, deps, "2\n2\n"); err != nil {
		t.Fatalf("setup: %v", err)
	}
	if len(fc.saved) != 1 {
		t.Fatalf("expected one save, got %d", len(fc.saved))
	}
	saved := fc.saved[0]
	if saved.Language != "中文" {
		t.Errorf("language = %q, want 中文", saved.Language)
	}
	// Backend config preserved.
	if saved.APIKey != "sk-keep" || saved.Backend != "openai" || saved.Model != "m" {
		t.Errorf("language-only setup clobbered backend config: %+v", saved)
	}
}

// Entry point 1: backend only configures the supported OpenAI-compatible API
// and must preserve the existing Language.
func TestSetupBackendOnlyPreservesLanguage(t *testing.T) {
	deps, _, _ := testDeps()
	fc := &fakeConfig{cfg: config.Config{Backend: "openai", Language: "中文"}}
	deps.Config = fc
	deps.Backend = fakeBackend{checkErr: nil}

	// Backend group, OpenAI preset, API key, then accept the preset defaults.
	if err := runSetup(t, deps, "1\n1\nsk-new\n\n\n"); err != nil {
		t.Fatalf("setup: %v", err)
	}
	if len(fc.saved) != 1 {
		t.Fatalf("expected one save, got %d", len(fc.saved))
	}
	saved := fc.saved[0]
	if saved.Backend != "openai" {
		t.Errorf("backend = %q, want openai", saved.Backend)
	}
	if saved.Language != "中文" {
		t.Errorf("backend-only setup dropped language: %q", saved.Language)
	}
}

// A failed verification must not save or corrupt the existing config.
func TestSetupVerificationFailureDoesNotSave(t *testing.T) {
	deps, _, _ := testDeps()
	fc := &fakeConfig{cfg: config.Config{Backend: "openai", APIKey: "sk-old"}}
	deps.Config = fc
	deps.Backend = fakeBackend{checkErr: errBoom}

	// Backend group -> OpenAI preset; verification fails.
	err := runSetup(t, deps, "1\n1\nsk-new\n\n\n")
	if err == nil {
		t.Fatal("expected error on failed verification")
	}
	if len(fc.saved) != 0 {
		t.Errorf("must not save on failed verification, saved=%v", fc.saved)
	}
}

// CLI backends must not appear in the interactive setup flow.
func TestSetupHidesCLIBackendChoices(t *testing.T) {
	var errOut bytes.Buffer
	deps, _, _ := testDeps()
	deps.ErrOut = &errOut
	deps.Config = &fakeConfig{cfg: config.Config{Backend: "openai"}}
	deps.Backend = fakeBackend{checkErr: nil}

	if err := runSetup(t, deps, "1\n1\nsk-new\n\n\n"); err != nil {
		t.Fatalf("setup: %v", err)
	}
	got := errOut.String()
	if !strings.Contains(got, "only currently available backend") {
		t.Errorf("missing API-only notice:\n%s", got)
	}
	if strings.Contains(got, "Codex CLI") || strings.Contains(got, "Claude Code CLI") {
		t.Errorf("CLI backend appeared in setup flow:\n%s", got)
	}
}

// Entry point 3: all settings — configures backend then language, saving once.
func TestSetupAllSettings(t *testing.T) {
	deps, _, _ := testDeps()
	fc := &fakeConfig{cfg: config.Config{Backend: "openai"}}
	deps.Config = fc
	deps.Backend = fakeBackend{checkErr: nil}

	// All settings, OpenAI preset, API key/defaults, then English language.
	if err := runSetup(t, deps, "3\n1\nsk-new\n\n\n1\n"); err != nil {
		t.Fatalf("setup: %v", err)
	}
	if len(fc.saved) != 1 {
		t.Fatalf("expected exactly one save, got %d", len(fc.saved))
	}
	saved := fc.saved[0]
	if saved.Backend != "openai" || saved.Language != "English" {
		t.Errorf("all-settings result unexpected: %+v", saved)
	}
}

func TestConfigSetHidesBackendFlag(t *testing.T) {
	deps, _, _ := testDeps()
	cmd := newConfigSetCommand(deps)
	if flag := cmd.Flags().Lookup("backend"); flag == nil || !flag.Hidden {
		t.Fatal("--backend must be hidden until CLI backend latency is acceptable")
	}
}

// Prompts must go to stderr, never stdout (stdout stays script-clean).
func TestSetupPromptsGoToStderr(t *testing.T) {
	var out, errOut bytes.Buffer
	deps, _, _ := testDeps()
	deps.Out = &out
	deps.ErrOut = &errOut
	deps.In = strings.NewReader("2\n1\n") // language -> English
	deps.Config = &fakeConfig{cfg: config.Config{Backend: "openai"}}

	cmd := NewRootCommand(deps)
	if err := execute(cmd, "config", "setup"); err != nil {
		t.Fatalf("setup: %v", err)
	}
	if strings.Contains(out.String(), "Select") {
		t.Errorf("wizard prompts leaked into stdout:\n%s", out.String())
	}
	if !strings.Contains(errOut.String(), "Select") {
		t.Errorf("expected wizard prompts on stderr:\n%s", errOut.String())
	}
}
