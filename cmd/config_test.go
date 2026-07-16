package cmd

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"

	"github.com/wungjyan/aicommit/internal/ai"
	"github.com/wungjyan/aicommit/internal/config"
)

// runConfig builds the command tree and runs `config <args...>`.
func runConfig(t *testing.T, deps Dependencies, args ...string) error {
	t.Helper()
	cmd := NewRootCommand(deps)
	return execute(cmd, append([]string{"config"}, args...)...)
}

func TestConfigDisplayOpenAIMasksKey(t *testing.T) {
	deps, out, _ := testDeps()
	deps.Config = &fakeConfig{cfg: config.Config{Backend: "openai", APIKey: "sk-abcdef1234567890", Model: "gpt-4o-mini"}}

	if err := runConfig(t, deps); err != nil {
		t.Fatalf("config: %v", err)
	}
	got := out.String()
	if strings.Contains(got, "sk-abcdef1234567890") {
		t.Errorf("raw API key leaked:\n%s", got)
	}
	if !strings.Contains(got, "sk-a") || !strings.Contains(got, "7890") {
		t.Errorf("expected masked key with prefix/suffix:\n%s", got)
	}
	if !strings.Contains(got, "Backend  : openai") {
		t.Errorf("missing backend line:\n%s", got)
	}
}

// Non-OpenAI backends must not display API Key / Base URL, and must show CLI
// status instead.
func TestConfigDisplayCodexHidesAPIFields(t *testing.T) {
	deps, out, _ := testDeps()
	deps.Config = &fakeConfig{cfg: config.Config{Backend: "codex", APIKey: "sk-stale"}}
	deps.Backend = fakeBackend{status: ai.CLIStatus{Installed: true, Path: "/usr/bin/codex", Auth: ai.AuthAuthenticated}}

	if err := runConfig(t, deps); err != nil {
		t.Fatalf("config: %v", err)
	}
	got := out.String()
	if strings.Contains(got, "API Key") || strings.Contains(got, "sk-stale") {
		t.Errorf("codex display leaked API fields:\n%s", got)
	}
	if !strings.Contains(got, "CLI      : /usr/bin/codex") {
		t.Errorf("missing CLI path:\n%s", got)
	}
	if !strings.Contains(got, "Auth     : available (reported by CLI)") {
		t.Errorf("missing auth status:\n%s", got)
	}
}

func TestConfigDisplayCodexLoginNotDetectedIsInformational(t *testing.T) {
	deps, out, _ := testDeps()
	deps.Config = &fakeConfig{cfg: config.Config{Backend: "codex"}}
	deps.Backend = fakeBackend{status: ai.CLIStatus{Installed: true, Path: "/usr/bin/codex", Auth: ai.AuthUnauthenticated}}

	if err := runConfig(t, deps); err != nil {
		t.Fatalf("config: %v", err)
	}
	if got := out.String(); !strings.Contains(got, "not detected (custom provider may still work)") {
		t.Errorf("expected friendly custom-provider hint:\n%s", got)
	}
}

// Degradation: a missing CLI must render informative strings, not fail.
func TestConfigDisplayCodexNotInstalled(t *testing.T) {
	deps, out, _ := testDeps()
	deps.Config = &fakeConfig{cfg: config.Config{Backend: "codex"}}
	deps.Backend = fakeBackend{status: ai.CLIStatus{Installed: false}}

	if err := runConfig(t, deps); err != nil {
		t.Fatalf("config must not fail on missing CLI: %v", err)
	}
	if !strings.Contains(out.String(), "not installed") {
		t.Errorf("expected 'not installed':\n%s", out.String())
	}
}

func TestConfigJSONMasksKey(t *testing.T) {
	deps, out, _ := testDeps()
	deps.Config = &fakeConfig{cfg: config.Config{Backend: "openai", APIKey: "sk-abcdef1234567890"}}

	if err := runConfig(t, deps, "--json"); err != nil {
		t.Fatalf("config --json: %v", err)
	}
	raw := out.String()
	if strings.Contains(raw, "sk-abcdef1234567890") {
		t.Errorf("raw key leaked in JSON:\n%s", raw)
	}

	var payload configJSON
	if err := json.Unmarshal([]byte(raw), &payload); err != nil {
		t.Fatalf("invalid JSON: %v\n%s", err, raw)
	}
	if payload.APIKey == nil || payload.APIKey.Source == "" {
		t.Errorf("expected api_key with source, got %+v", payload.APIKey)
	}
	if payload.Backend.Value != "openai" {
		t.Errorf("backend = %q", payload.Backend.Value)
	}
}

func TestConfigPathWritesToStdout(t *testing.T) {
	deps, out, _ := testDeps()
	deps.Config = &fakeConfig{path: "/home/x/.aicommit/config.json"}

	if err := runConfig(t, deps, "path"); err != nil {
		t.Fatalf("config path: %v", err)
	}
	if got := strings.TrimSpace(out.String()); got != "/home/x/.aicommit/config.json" {
		t.Errorf("path = %q", got)
	}
}

func TestConfigSetRequiresAField(t *testing.T) {
	deps, _, _ := testDeps()
	cmd := NewRootCommand(deps)
	cmd.SetErr(new(bytes.Buffer))
	err := execute(cmd, "config", "set")
	if err == nil {
		t.Fatal("expected usage error with no fields")
	}
	if !isUsageError(err) {
		t.Errorf("expected usage error, got %v", err)
	}
}

func TestConfigSetPartialUpdatePreservesFields(t *testing.T) {
	deps, _, _ := testDeps()
	fc := &fakeConfig{cfg: config.Config{Backend: "openai", APIKey: "sk-old", BaseURL: "https://old/v1", Model: "old"}}
	deps.Config = fc

	cmd := NewRootCommand(deps)
	if err := execute(cmd, "config", "set", "--model", "new-model"); err != nil {
		t.Fatalf("config set: %v", err)
	}
	if len(fc.saved) != 1 {
		t.Fatalf("expected one save, got %d", len(fc.saved))
	}
	saved := fc.saved[0]
	if saved.Model != "new-model" {
		t.Errorf("model not updated: %q", saved.Model)
	}
	// Other fields preserved.
	if saved.APIKey != "sk-old" || saved.BaseURL != "https://old/v1" {
		t.Errorf("set clobbered other fields: %+v", saved)
	}
}

func TestConfigSetRejectsInvalidBackend(t *testing.T) {
	deps, _, _ := testDeps()
	fc := &fakeConfig{}
	deps.Config = fc

	cmd := NewRootCommand(deps)
	cmd.SetErr(new(bytes.Buffer))
	err := execute(cmd, "config", "set", "--backend", "gemini")
	if !isUsageError(err) {
		t.Errorf("expected usage error for invalid backend, got %v", err)
	}
	if len(fc.saved) != 0 {
		t.Error("invalid backend must not save")
	}
}

// --check failure must not overwrite the existing config.
func TestConfigSetCheckFailureDoesNotSave(t *testing.T) {
	deps, _, _ := testDeps()
	fc := &fakeConfig{cfg: config.Config{Backend: "openai", APIKey: "sk-old"}}
	deps.Config = fc
	deps.Backend = fakeBackend{checkErr: errBoom}

	cmd := NewRootCommand(deps)
	err := execute(cmd, "config", "set", "--api-key", "sk-new", "--check")
	if err == nil {
		t.Fatal("expected error when check fails")
	}
	if len(fc.saved) != 0 {
		t.Errorf("config must not be saved when --check fails, saved=%v", fc.saved)
	}
}

func TestConfigCheckSuccess(t *testing.T) {
	deps, _, _ := testDeps()
	deps.Config = &fakeConfig{cfg: config.Config{Backend: "openai", APIKey: "sk"}}
	deps.Backend = fakeBackend{checkErr: nil}
	ui := &recordingUI{}
	deps.UI = ui

	cmd := NewRootCommand(deps)
	if err := execute(cmd, "config", "check"); err != nil {
		t.Fatalf("config check: %v", err)
	}
	if len(ui.success) == 0 {
		t.Error("expected a success message")
	}
}

func TestConfigCheckCLISuccessExplainsDeferredProviderCheck(t *testing.T) {
	deps, _, _ := testDeps()
	deps.Config = &fakeConfig{cfg: config.Config{Backend: "codex"}}
	deps.Backend = fakeBackend{checkErr: nil}
	ui := &recordingUI{}
	deps.UI = ui

	cmd := NewRootCommand(deps)
	if err := execute(cmd, "config", "check"); err != nil {
		t.Fatalf("config check: %v", err)
	}
	if len(ui.success) != 1 || !strings.Contains(ui.success[0], "will be checked when generating") {
		t.Errorf("unexpected success message: %v", ui.success)
	}
}

func TestConfigCheckFailure(t *testing.T) {
	deps, _, _ := testDeps()
	deps.Config = &fakeConfig{cfg: config.Config{Backend: "openai", APIKey: "sk"}}
	deps.Backend = fakeBackend{checkErr: errBoom}

	cmd := NewRootCommand(deps)
	cmd.SetErr(new(bytes.Buffer))
	if err := execute(cmd, "config", "check"); err == nil {
		t.Fatal("expected error when check fails")
	}
}

// The legacy `ai` command must no longer exist.
func TestLegacyAICommandRemoved(t *testing.T) {
	deps, _, _ := testDeps()
	cmd := NewRootCommand(deps)
	cmd.SetOut(new(bytes.Buffer))
	cmd.SetErr(new(bytes.Buffer))
	err := execute(cmd, "ai")
	if err == nil || !strings.Contains(err.Error(), "unknown command") {
		t.Errorf("expected `ai` to be unknown, got %v", err)
	}
}

// isUsageError reports whether err is (or wraps) a usage error.
func isUsageError(err error) bool {
	return err != nil && strings.Contains(err.Error(), "usage error")
}
