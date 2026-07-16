package ai

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"time"
)

// codexBin is the Codex CLI executable name.
const codexBin = "codex"

// cliGenerateTimeout bounds a single CLI generation call.
const cliGenerateTimeout = 60 * time.Second

// CodexProvider generates commit messages through the user's authenticated
// Codex CLI. It never reads or persists Codex credentials; it inherits the
// CLI's own authentication and model configuration.
type CodexProvider struct {
	runner   CommandRunner
	language string

	// lookPath resolves the executable; injectable for tests.
	lookPath func(string) (string, error)
	// genTimeout bounds generation; zero uses cliGenerateTimeout.
	genTimeout time.Duration
}

// NewCodexProvider builds a Codex provider backed by the real command runner.
func NewCodexProvider(language string) *CodexProvider {
	return &CodexProvider{
		runner:   NewCommandRunner(),
		language: language,
		lookPath: exec.LookPath,
	}
}

// resolveBin returns the Codex executable path or ErrCLINotInstalled.
func (p *CodexProvider) resolveBin() (string, error) {
	lookPath := p.lookPath
	if lookPath == nil {
		lookPath = exec.LookPath
	}
	path, err := lookPath(codexBin)
	if err != nil {
		return "", fmt.Errorf("%w: install the Codex CLI and run `codex login`", ErrCLINotInstalled)
	}
	return path, nil
}

// Generate runs Codex non-interactively in a read-only sandbox. The generation
// rules are passed as the positional prompt; the diff is piped via stdin so no
// diff content is ever interpreted as a shell argument. The final message is
// read from the --output-last-message file, falling back to stdout.
func (p *CodexProvider) Generate(ctx context.Context, diff string) (string, error) {
	bin, err := p.resolveBin()
	if err != nil {
		return "", err
	}

	outFile, err := os.CreateTemp("", "aicommit-codex-*.txt")
	if err != nil {
		return "", fmt.Errorf("failed to create temp file: %w", err)
	}
	outPath := outFile.Name()
	outFile.Close()
	defer os.Remove(outPath)

	timeout := p.genTimeout
	if timeout <= 0 {
		timeout = cliGenerateTimeout
	}
	runCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	spec := CommandSpec{
		Name: bin,
		Args: []string{
			"exec",
			"--ephemeral",
			"--skip-git-repo-check",
			"-s", "read-only",
			"-o", outPath,
			BuildPrompt(p.language),
		},
		Stdin: diff,
	}

	res, runErr := p.runner.Run(runCtx, spec)
	if runErr != nil {
		return "", fmt.Errorf("codex generation failed: %w", runErr)
	}

	// Prefer the message file; fall back to stdout if it is empty.
	msg := ""
	if data, readErr := os.ReadFile(outPath); readErr == nil {
		msg = trimMessage(string(data))
	}
	if msg == "" {
		msg = trimMessage(res.Stdout)
	}
	if msg == "" {
		return "", ErrEmptyMessage
	}
	return msg, nil
}

// CheckAuth verifies the Codex CLI is installed and logged in via
// `codex login status` (authenticated => stdout "Logged in using ..." + exit 0).
func (p *CodexProvider) CheckAuth(ctx context.Context) error {
	bin, err := p.resolveBin()
	if err != nil {
		return err
	}

	runCtx, cancel := context.WithTimeout(ctx, cliAuthTimeout)
	defer cancel()

	res, runErr := p.runner.Run(runCtx, CommandSpec{
		Name: bin,
		Args: []string{"login", "status"},
	})
	if runErr == nil && res.ExitCode == 0 && strings.Contains(res.Stdout, "Logged in") {
		return nil
	}
	return fmt.Errorf("%w: run `codex login` to authenticate", ErrCLINotAuthenticated)
}

// Installed reports whether the Codex CLI is available, returning its path.
func (p *CodexProvider) Installed() (string, bool) {
	path, err := p.resolveBin()
	if err != nil {
		return "", false
	}
	return path, true
}
