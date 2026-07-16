package ai

import (
	"context"
	"errors"

	"github.com/wungjyan/aicommit/internal/config"
)

// CLIStatus is a credential-free snapshot of a local CLI backend's state, for
// display by `aicommit config`. It never contains tokens or account data.
type CLIStatus struct {
	Installed bool
	Path      string // executable path when installed
	// Auth is the CLI's informational login report. It is one of
	// "authenticated", "unauthenticated", "unknown" (probe error or timeout),
	// or "" when the backend is not a local CLI. It is never a configuration
	// gate because custom providers may use separate credentials.
	Auth string
}

const (
	AuthAuthenticated   = "authenticated"
	AuthUnauthenticated = "unauthenticated"
	AuthUnknown         = "unknown"
)

// localCLI is the shared behavior of the Codex and Claude providers.
type localCLI interface {
	Installed() (string, bool)
	CheckInstalled() error
	CheckAuth(ctx context.Context) error
}

// Check verifies the prerequisites owned by aicommit without mutating
// configuration. OpenAI performs a Chat Completions ping because aicommit owns
// that API configuration. Codex and Claude only require their CLI to be
// installed; authentication and provider configuration belong to those CLIs
// and are verified by the first real generation call.
func Check(ctx context.Context, cfg config.Config) error {
	eff := config.Resolve(cfg)
	switch eff.Backend.Value {
	case config.BackendOpenAI:
		p, err := newOpenAIFromEffective(eff)
		if err != nil {
			return err
		}
		return p.Ping(ctx)
	case config.BackendCodex:
		return NewCodexProvider(eff.Language.Value).CheckInstalled()
	case config.BackendClaude:
		return NewClaudeProvider(eff.Language.Value).CheckInstalled()
	default:
		return errUnknownBackend(eff.Backend.Value)
	}
}

// Status returns a non-failing, credential-free status snapshot for display.
// Only Codex and Claude populate install/auth fields; OpenAI returns an empty
// status because its credentials live in the effective config, not a CLI.
func Status(cfg config.Config) CLIStatus {
	eff := config.Resolve(cfg)

	var cli localCLI
	switch eff.Backend.Value {
	case config.BackendCodex:
		cli = NewCodexProvider(eff.Language.Value)
	case config.BackendClaude:
		cli = NewClaudeProvider(eff.Language.Value)
	default:
		return CLIStatus{}
	}

	path, installed := cli.Installed()
	if !installed {
		return CLIStatus{Installed: false}
	}

	// Probe auth with a bounded timeout; any error or timeout degrades to
	// "unknown" rather than failing the whole display command.
	ctx, cancel := context.WithTimeout(context.Background(), cliAuthTimeout)
	defer cancel()

	auth := AuthUnknown
	switch err := cli.CheckAuth(ctx); {
	case err == nil:
		auth = AuthAuthenticated
	case errors.Is(err, ErrCLINotAuthenticated):
		auth = AuthUnauthenticated
	}
	return CLIStatus{Installed: true, Path: path, Auth: auth}
}

// errUnknownBackend wraps ErrUnknownBackend with the offending value.
func errUnknownBackend(name string) error {
	return &backendError{name: name}
}

type backendError struct{ name string }

func (e *backendError) Error() string { return "unknown AI backend: " + e.name }
func (e *backendError) Unwrap() error { return ErrUnknownBackend }
