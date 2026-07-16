package ai

import (
	"context"
	"errors"
	"strings"
	"testing"
)

// claudeTestProvider builds a ClaudeProvider wired to a fake runner and a
// lookPath that reports the binary as installed.
func claudeTestProvider(runner CommandRunner) *ClaudeProvider {
	return &ClaudeProvider{
		runner:   runner,
		language: "English",
		lookPath: func(string) (string, error) { return "/usr/bin/claude", nil },
	}
}

func TestClaudeGenerateReadsStdout(t *testing.T) {
	runner := &fakeRunner{result: CommandResult{Stdout: "feat: add claude support\n", ExitCode: 0}}
	p := claudeTestProvider(runner)

	msg, err := p.Generate(context.Background(), "diff --git a b")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if msg != "feat: add claude support" {
		t.Errorf("message = %q, want %q", msg, "feat: add claude support")
	}
}

// The generation command must use print mode, disable all tools and disable
// session persistence, with the diff on stdin and the prompt as a positional arg.
func TestClaudeGenerateCommandShape(t *testing.T) {
	runner := &fakeRunner{result: CommandResult{Stdout: "feat: x", ExitCode: 0}}
	p := claudeTestProvider(runner)

	diff := "diff --git a/x b/x\n+ $(rm -rf /) `id`"
	if _, err := p.Generate(context.Background(), diff); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	spec := runner.lastSpec

	// -p must be present, and --tools must be immediately followed by an empty
	// string (disable all tools).
	assertContainsArg(t, spec.Args, "-p")
	assertContainsArg(t, spec.Args, "--no-session-persistence")
	assertArgPair(t, spec.Args, "--tools", "")

	// diff travels via stdin, never as an argument.
	if spec.Stdin != diff {
		t.Errorf("diff not passed via stdin")
	}
	if strings.Contains(strings.Join(spec.Args, " "), "rm -rf") {
		t.Errorf("diff content leaked into args: %v", spec.Args)
	}
	// The prompt (last arg) carries the Conventional Commit rules.
	last := spec.Args[len(spec.Args)-1]
	if !strings.Contains(last, "Conventional Commit") {
		t.Errorf("last arg is not the prompt: %q", last)
	}
}

func TestClaudeGenerateEmptyOutput(t *testing.T) {
	runner := &fakeRunner{result: CommandResult{Stdout: "  \n ", ExitCode: 0}}
	p := claudeTestProvider(runner)

	_, err := p.Generate(context.Background(), "diff")
	if !errors.Is(err, ErrEmptyMessage) {
		t.Errorf("expected ErrEmptyMessage, got %v", err)
	}
}

func TestClaudeGenerateRunnerError(t *testing.T) {
	runner := &fakeRunner{err: ErrCommandTimeout}
	p := claudeTestProvider(runner)

	_, err := p.Generate(context.Background(), "diff")
	if !errors.Is(err, ErrCommandTimeout) {
		t.Errorf("expected wrapped ErrCommandTimeout, got %v", err)
	}
}

func TestClaudeNotInstalled(t *testing.T) {
	p := &ClaudeProvider{
		runner:   &fakeRunner{},
		lookPath: func(string) (string, error) { return "", errors.New("not found") },
	}

	if _, err := p.Generate(context.Background(), "diff"); !errors.Is(err, ErrCLINotInstalled) {
		t.Errorf("Generate: expected ErrCLINotInstalled, got %v", err)
	}
	if err := p.CheckInstalled(); !errors.Is(err, ErrCLINotInstalled) {
		t.Errorf("CheckInstalled: expected ErrCLINotInstalled, got %v", err)
	}
	if err := p.CheckAuth(context.Background()); !errors.Is(err, ErrCLINotInstalled) {
		t.Errorf("CheckAuth: expected ErrCLINotInstalled, got %v", err)
	}
	if _, ok := p.Installed(); ok {
		t.Error("Installed() = true, want false")
	}
}

func TestClaudeCheckInstalledDoesNotProbeAuth(t *testing.T) {
	runner := &fakeRunner{err: ErrCLINotAuthenticated}
	p := claudeTestProvider(runner)

	if err := p.CheckInstalled(); err != nil {
		t.Fatalf("CheckInstalled returned error: %v", err)
	}
	if runner.lastSpec.Name != "" {
		t.Errorf("installation check unexpectedly ran command: %+v", runner.lastSpec)
	}
}

func TestClaudeCheckAuthAuthenticated(t *testing.T) {
	runner := &fakeRunner{result: CommandResult{
		Stdout:   `{"loggedIn": true, "authMethod": "console"}`,
		ExitCode: 0,
	}}
	p := claudeTestProvider(runner)

	if err := p.CheckAuth(context.Background()); err != nil {
		t.Errorf("expected authenticated, got %v", err)
	}
	if got := strings.Join(runner.lastSpec.Args, " "); got != "auth status" {
		t.Errorf("auth args = %q, want \"auth status\"", got)
	}
}

func TestClaudeCheckAuthLoggedOut(t *testing.T) {
	// Valid JSON but loggedIn=false => unauthenticated.
	runner := &fakeRunner{result: CommandResult{
		Stdout:   `{"loggedIn": false}`,
		ExitCode: 0,
	}}
	p := claudeTestProvider(runner)

	if err := p.CheckAuth(context.Background()); !errors.Is(err, ErrCLINotAuthenticated) {
		t.Errorf("expected ErrCLINotAuthenticated, got %v", err)
	}
}

func TestClaudeCheckAuthUnparseable(t *testing.T) {
	// Non-JSON output must not be treated as authenticated.
	runner := &fakeRunner{result: CommandResult{Stdout: "not json", ExitCode: 0}}
	p := claudeTestProvider(runner)

	if err := p.CheckAuth(context.Background()); !errors.Is(err, ErrCLINotAuthenticated) {
		t.Errorf("expected ErrCLINotAuthenticated, got %v", err)
	}
}

// assertContainsArg fails if want is not present in args.
func assertContainsArg(t *testing.T, args []string, want string) {
	t.Helper()
	for _, a := range args {
		if a == want {
			return
		}
	}
	t.Errorf("args missing %q: %v", want, args)
}

// assertArgPair fails unless flag appears immediately followed by value.
func assertArgPair(t *testing.T, args []string, flag, value string) {
	t.Helper()
	for i := 0; i < len(args)-1; i++ {
		if args[i] == flag && args[i+1] == value {
			return
		}
	}
	t.Errorf("args missing pair %q %q: %v", flag, value, args)
}
