package cmd

import (
	"context"
	"io"

	"github.com/wungjyan/aicommit/internal/config"
)

// Dependencies holds the external collaborators the command tree needs.
//
// Production wiring is assembled in Execute; tests inject fakes so no command
// touches the real Git repo, config file, AI backend, or terminal. Every field
// is an interface or io stream so the command layer never reaches for package
// globals or os.Std* directly.
type Dependencies struct {
	In     io.Reader
	Out    io.Writer
	ErrOut io.Writer

	Git      GitService
	Config   ConfigStore
	Provider ProviderFactory
	UI       UI
	Confirm  Confirmer

	Version VersionInfo
}

// VersionInfo carries the build metadata injected via ldflags.
type VersionInfo struct {
	Version string
	Commit  string
	Date    string
}

// GitService abstracts the git operations used by the command layer.
type GitService interface {
	IsGitRepo() error
	GetStagedDiff() (string, error)
	Commit(message string) error
}

// ConfigStore abstracts reading the persisted configuration.
type ConfigStore interface {
	Load() (config.Config, error)
}

// Provider generates a commit message from a staged diff.
type Provider interface {
	Generate(ctx context.Context, diff string) (string, error)
}

// ProviderFactory builds a Provider from the effective configuration.
type ProviderFactory interface {
	New(cfg config.Config) (Provider, error)
}

// Confirmer runs one round of the interactive confirmation prompt and reports
// the chosen action ("commit", "edit", "regenerate", "quit").
type Confirmer interface {
	Confirm(message string, valid bool) (action string, edited string, err error)
}

// UI abstracts user-facing status output and the spinner.
type UI interface {
	Success(msg string)
	Error(msg string)
	Warn(msg string)
	Info(msg string)
	Spinner(label string, fn func() error) error
}
