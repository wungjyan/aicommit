package cmd

import (
	"context"

	"github.com/wungjyan/aicommit/internal/ai"
	"github.com/wungjyan/aicommit/internal/config"
	"github.com/wungjyan/aicommit/internal/git"
	"github.com/wungjyan/aicommit/internal/prompt"
	"github.com/wungjyan/aicommit/internal/ui"
)

// productionDeps wires the command tree to the real Git repo, config file,
// AI backend, prompt and UI. It preserves the behavior that existed before the
// command layer became injectable.
func productionDeps(v VersionInfo) Dependencies {
	return Dependencies{
		Git:      gitAdapter{},
		Config:   configAdapter{},
		Provider: providerFactory{},
		Backend:  backendAdapter{},
		UI:       uiAdapter{},
		Confirm:  confirmAdapter{},
		Editor:   editorAdapter{},
		IsTTY:    isTerminal,
		Version:  v,
	}
}

type gitAdapter struct{}

func (gitAdapter) IsGitRepo() error               { return git.IsGitRepo() }
func (gitAdapter) GetStagedDiff() (string, error) { return git.GetStagedDiff() }
func (gitAdapter) Commit(message string) error    { return git.Commit(message) }

type configAdapter struct{}

func (configAdapter) Load() (config.Config, error) { return config.LoadConfig() }
func (configAdapter) Save(cfg config.Config) error { return config.SaveConfig(cfg) }
func (configAdapter) Path() (string, error)        { return config.ConfigPath() }

// backendAdapter routes backend checks and status queries to the AI package,
// keeping CLI details out of the command layer.
type backendAdapter struct{}

func (backendAdapter) Check(ctx context.Context, cfg config.Config) error {
	return ai.Check(ctx, cfg)
}
func (backendAdapter) Status(cfg config.Config) ai.CLIStatus {
	return ai.Status(cfg)
}

// providerFactory builds the backend-appropriate provider via the AI factory,
// which dispatches on the effective backend and never falls back to OpenAI.
type providerFactory struct{}

func (providerFactory) New(cfg config.Config) (Provider, error) {
	return ai.NewProvider(cfg)
}

type uiAdapter struct{}

func (uiAdapter) Success(msg string)                          { ui.Success(msg) }
func (uiAdapter) Error(msg string)                            { ui.Error(msg) }
func (uiAdapter) Warn(msg string)                             { ui.Warn(msg) }
func (uiAdapter) Info(msg string)                             { ui.Info(msg) }
func (uiAdapter) Spinner(label string, fn func() error) error { return ui.Spinner(label, fn) }
func (uiAdapter) DisableColor()                               { ui.DisableColor() }

type confirmAdapter struct{}

func (confirmAdapter) Confirm(message string, valid bool) (string, string, error) {
	return prompt.Confirm(message, valid)
}

type editorAdapter struct{}

func (editorAdapter) Edit(message string) (string, error) {
	return prompt.EditMessage(message)
}

// compile-time assertion that the OpenAI provider satisfies the local Provider
// interface, so the factory adapter stays honest.
var _ Provider = (*ai.OpenAIProvider)(nil)
