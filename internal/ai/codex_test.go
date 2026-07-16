package ai

import (
	"context"
	"errors"
	"os"
	"strings"
	"testing"
)

// fakeRunner is a CommandRunner that records the last spec and returns a canned
// result, so provider tests never spawn a real CLI. When writeOutputFile is set,
// it writes that content to the -o output file path found in the spec args,
// mimicking Codex's --output-last-message behavior.
type fakeRunner struct {
	result CommandResult
	err    error

	lastSpec        CommandSpec
	writeOutputFile string
}

func (f *fakeRunner) Run(ctx context.Context, spec CommandSpec) (CommandResult, error) {
	f.lastSpec = spec
	if f.writeOutputFile != "" {
		if path := outputPathFromArgs(spec.Args); path != "" {
			_ = os.WriteFile(path, []byte(f.writeOutputFile), 0600)
		}
	}
	return f.result, f.err
}

// outputPathFromArgs returns the value following "-o" in args, if present.
func outputPathFromArgs(args []string) string {
	for i := 0; i < len(args)-1; i++ {
		if args[i] == "-o" {
			return args[i+1]
		}
	}
	return ""
}

// codexTestProvider builds a CodexProvider wired to a fake runner and a lookPath
// that reports the binary as installed.
func codexTestProvider(runner CommandRunner) *CodexProvider {
	return &CodexProvider{
		runner:   runner,
		language: "English",
		lookPath: func(string) (string, error) { return "/usr/bin/codex", nil },
	}
}

func TestCodexGenerateReadsOutputFile(t *testing.T) {
	runner := &fakeRunner{
		result:          CommandResult{Stdout: "noisy diagnostic on stdout", ExitCode: 0},
		writeOutputFile: "feat: add codex support\n",
	}
	p := codexTestProvider(runner)

	msg, err := p.Generate(context.Background(), "diff --git a b")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// The message comes from the -o file, not the noisy stdout.
	if msg != "feat: add codex support" {
		t.Errorf("message = %q, want %q", msg, "feat: add codex support")
	}
}

func TestCodexGenerateFallsBackToStdout(t *testing.T) {
	// No output file written -> the provider falls back to stdout.
	runner := &fakeRunner{result: CommandResult{Stdout: "fix: fallback message", ExitCode: 0}}
	p := codexTestProvider(runner)

	msg, err := p.Generate(context.Background(), "diff")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if msg != "fix: fallback message" {
		t.Errorf("message = %q, want fallback from stdout", msg)
	}
}

// The generation command must use exec, ephemeral session, read-only sandbox and
// skip-git-repo-check, with the diff on stdin and the prompt as a positional arg.
func TestCodexGenerateCommandShape(t *testing.T) {
	runner := &fakeRunner{writeOutputFile: "feat: x"}
	p := codexTestProvider(runner)

	diff := "diff --git a/x b/x\n+ $(rm -rf /) `id`"
	if _, err := p.Generate(context.Background(), diff); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	spec := runner.lastSpec
	args := strings.Join(spec.Args, " ")
	for _, want := range []string{"exec", "--ephemeral", "--skip-git-repo-check", "-s read-only", "-o "} {
		if !strings.Contains(args, want) {
			t.Errorf("args missing %q: %v", want, spec.Args)
		}
	}
	// diff travels via stdin, never as an argument.
	if spec.Stdin != diff {
		t.Errorf("diff not passed via stdin")
	}
	if strings.Contains(args, "rm -rf") {
		t.Errorf("diff content leaked into args: %v", spec.Args)
	}
	// The prompt (last arg) carries the Conventional Commit rules.
	last := spec.Args[len(spec.Args)-1]
	if !strings.Contains(last, "Conventional Commit") {
		t.Errorf("last arg is not the prompt: %q", last)
	}
}

func TestCodexGenerateEmptyOutput(t *testing.T) {
	runner := &fakeRunner{result: CommandResult{Stdout: "   ", ExitCode: 0}}
	p := codexTestProvider(runner)

	_, err := p.Generate(context.Background(), "diff")
	if !errors.Is(err, ErrEmptyMessage) {
		t.Errorf("expected ErrEmptyMessage, got %v", err)
	}
}

func TestCodexGenerateRunnerError(t *testing.T) {
	runner := &fakeRunner{err: ErrCommandTimeout}
	p := codexTestProvider(runner)

	_, err := p.Generate(context.Background(), "diff")
	if !errors.Is(err, ErrCommandTimeout) {
		t.Errorf("expected wrapped ErrCommandTimeout, got %v", err)
	}
}

func TestCodexNotInstalled(t *testing.T) {
	p := &CodexProvider{
		runner:   &fakeRunner{},
		lookPath: func(string) (string, error) { return "", errors.New("not found") },
	}

	if _, err := p.Generate(context.Background(), "diff"); !errors.Is(err, ErrCLINotInstalled) {
		t.Errorf("Generate: expected ErrCLINotInstalled, got %v", err)
	}
	if err := p.CheckAuth(context.Background()); !errors.Is(err, ErrCLINotInstalled) {
		t.Errorf("CheckAuth: expected ErrCLINotInstalled, got %v", err)
	}
	if _, ok := p.Installed(); ok {
		t.Error("Installed() = true, want false")
	}
}

func TestCodexCheckAuthAuthenticated(t *testing.T) {
	runner := &fakeRunner{result: CommandResult{Stdout: "Logged in using ChatGPT\n", ExitCode: 0}}
	p := codexTestProvider(runner)

	if err := p.CheckAuth(context.Background()); err != nil {
		t.Errorf("expected authenticated, got %v", err)
	}
	if got := strings.Join(runner.lastSpec.Args, " "); got != "login status" {
		t.Errorf("auth args = %q, want \"login status\"", got)
	}
}

func TestCodexCheckAuthUnauthenticated(t *testing.T) {
	// Non-zero exit / no "Logged in" line => unauthenticated.
	runner := &fakeRunner{result: CommandResult{Stdout: "Not logged in", ExitCode: 1}}
	p := codexTestProvider(runner)

	if err := p.CheckAuth(context.Background()); !errors.Is(err, ErrCLINotAuthenticated) {
		t.Errorf("expected ErrCLINotAuthenticated, got %v", err)
	}
}
