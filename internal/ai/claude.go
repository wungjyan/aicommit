package ai

import (
	"context"
	"encoding/json"
	"fmt"
	"os/exec"
	"time"
)

// claudeBin is the Claude Code CLI executable name.
const claudeBin = "claude"

// ClaudeProvider generates commit messages through the user's authenticated
// Claude Code CLI. It never reads or persists Claude credentials; it inherits
// the CLI's own authentication and model configuration.
type ClaudeProvider struct {
	runner   CommandRunner
	language string

	// lookPath resolves the executable; injectable for tests.
	lookPath func(string) (string, error)
	// genTimeout bounds generation; zero uses cliGenerateTimeout.
	genTimeout time.Duration
}

// NewClaudeProvider builds a Claude provider backed by the real command runner.
func NewClaudeProvider(language string) *ClaudeProvider {
	return &ClaudeProvider{
		runner:   NewCommandRunner(),
		language: language,
		lookPath: exec.LookPath,
	}
}

// resolveBin returns the Claude executable path or ErrCLINotInstalled.
func (p *ClaudeProvider) resolveBin() (string, error) {
	lookPath := p.lookPath
	if lookPath == nil {
		lookPath = exec.LookPath
	}
	path, err := lookPath(claudeBin)
	if err != nil {
		return "", fmt.Errorf("%w: install the Claude Code CLI and run `claude` to log in", ErrCLINotInstalled)
	}
	return path, nil
}

// Generate runs Claude Code non-interactively with all tools disabled and no
// session persistence. The generation rules are passed as the positional
// prompt; the diff is piped via stdin so no diff content is ever interpreted as
// a shell argument. The clean final message is read from stdout.
func (p *ClaudeProvider) Generate(ctx context.Context, diff string) (string, error) {
	bin, err := p.resolveBin()
	if err != nil {
		return "", err
	}

	timeout := p.genTimeout
	if timeout <= 0 {
		timeout = cliGenerateTimeout
	}
	runCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	spec := CommandSpec{
		Name: bin,
		Args: []string{
			"-p",
			"--tools", "",
			"--no-session-persistence",
			BuildPrompt(p.language),
		},
		Stdin: diff,
	}

	res, runErr := p.runner.Run(runCtx, spec)
	if runErr != nil {
		return "", fmt.Errorf("claude generation failed: %w", runErr)
	}

	msg := trimMessage(res.Stdout)
	if msg == "" {
		return "", ErrEmptyMessage
	}
	return msg, nil
}

// claudeAuthStatus is the subset of `claude auth status` JSON we consume. Only
// the boolean is read; no token or account credential is parsed or stored.
type claudeAuthStatus struct {
	LoggedIn bool `json:"loggedIn"`
}

// CheckAuth verifies the Claude CLI is installed and logged in via
// `claude auth status`, parsing the loggedIn boolean from its JSON output.
func (p *ClaudeProvider) CheckAuth(ctx context.Context) error {
	bin, err := p.resolveBin()
	if err != nil {
		return err
	}

	runCtx, cancel := context.WithTimeout(ctx, cliAuthTimeout)
	defer cancel()

	res, runErr := p.runner.Run(runCtx, CommandSpec{
		Name: bin,
		Args: []string{"auth", "status"},
	})
	if runErr == nil && res.ExitCode == 0 {
		var status claudeAuthStatus
		if json.Unmarshal([]byte(res.Stdout), &status) == nil && status.LoggedIn {
			return nil
		}
	}
	return fmt.Errorf("%w: run `claude` to log in", ErrCLINotAuthenticated)
}

// Installed reports whether the Claude CLI is available, returning its path.
func (p *ClaudeProvider) Installed() (string, bool) {
	path, err := p.resolveBin()
	if err != nil {
		return "", false
	}
	return path, true
}
