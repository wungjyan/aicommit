package cmd

import (
	"bytes"
	"errors"
	"strings"
	"testing"

	"github.com/spf13/cobra"
	"github.com/wungjyan/aicommit/internal/ai"
	"github.com/wungjyan/aicommit/internal/config"
)

// execute runs the command tree with the given args and returns the error.
func execute(cmd *cobra.Command, args ...string) error {
	cmd.SetArgs(args)
	return cmd.Execute()
}

func TestRootHelp(t *testing.T) {
	deps, out, _ := testDeps()
	cmd := NewRootCommand(deps)
	cmd.SetOut(out)

	if err := execute(cmd, "--help"); err != nil {
		t.Fatalf("--help returned error: %v", err)
	}

	got := out.String()
	for _, want := range []string{"aicommit", "Usage:", "version"} {
		if !strings.Contains(got, want) {
			t.Errorf("help output missing %q\n%s", want, got)
		}
	}
}

func TestVersionCommand(t *testing.T) {
	deps, out, _ := testDeps()
	cmd := NewRootCommand(deps)
	cmd.SetOut(out)

	if err := execute(cmd, "version"); err != nil {
		t.Fatalf("version returned error: %v", err)
	}

	got := strings.TrimSpace(out.String())
	want := "aicommit 1.2.3 (commit: abc1234, built: 2026-07-16T00:00:00Z)"
	if got != want {
		t.Errorf("version output = %q, want %q", got, want)
	}
}

func TestVersionFlags(t *testing.T) {
	for _, flag := range []string{"-v", "--version"} {
		t.Run(flag, func(t *testing.T) {
			deps, out, _ := testDeps()
			if err := execute(NewRootCommand(deps), flag); err != nil {
				t.Fatalf("%s returned error: %v", flag, err)
			}
			want := "aicommit 1.2.3 (commit: abc1234, built: 2026-07-16T00:00:00Z)"
			if got := strings.TrimSpace(out.String()); got != want {
				t.Errorf("version output = %q, want %q", got, want)
			}
		})
	}
}

func TestUnknownCommand(t *testing.T) {
	deps, _, _ := testDeps()
	cmd := NewRootCommand(deps)
	// Discard cobra's own error text; we only care about the returned error.
	cmd.SetOut(new(bytes.Buffer))
	cmd.SetErr(new(bytes.Buffer))

	err := execute(cmd, "does-not-exist")
	if err == nil {
		t.Fatal("expected error for unknown command, got nil")
	}
	if !strings.Contains(err.Error(), "unknown command") {
		t.Errorf("error = %v, want it to mention unknown command", err)
	}
	if got := ExitCode(err); got != ExitUsageError {
		t.Errorf("ExitCode = %d, want %d", got, ExitUsageError)
	}
}

func TestUnknownFlagReturnsUsageExitCode(t *testing.T) {
	deps, _, _ := testDeps()
	err := execute(NewRootCommand(deps), "--does-not-exist")
	if !errors.Is(err, ErrUsage) {
		t.Fatalf("error = %v, want ErrUsage", err)
	}
	if got := ExitCode(err); got != ExitUsageError {
		t.Errorf("ExitCode = %d, want %d", got, ExitUsageError)
	}
}

func TestRootRejectsExtraArgs(t *testing.T) {
	deps, _, _ := testDeps()
	cmd := NewRootCommand(deps)
	cmd.SetOut(new(bytes.Buffer))
	cmd.SetErr(new(bytes.Buffer))

	err := execute(cmd, "unexpected-arg")
	if err == nil {
		t.Fatal("expected error for extra positional arg, got nil")
	}
}

func TestVersionRejectsExtraArgs(t *testing.T) {
	deps, _, _ := testDeps()
	cmd := NewRootCommand(deps)
	cmd.SetOut(new(bytes.Buffer))
	cmd.SetErr(new(bytes.Buffer))

	if err := execute(cmd, "version", "extra"); err == nil {
		t.Fatal("expected error for extra arg to version, got nil")
	}
}

// TestNoGlobalFlagPollution verifies the tree can be built and run repeatedly in
// one process without leaking flag state between constructions.
func TestNoGlobalFlagPollution(t *testing.T) {
	for i := 0; i < 3; i++ {
		deps, out, _ := testDeps()
		cmd := NewRootCommand(deps)
		cmd.SetOut(out)
		if err := execute(cmd, "version"); err != nil {
			t.Fatalf("iteration %d: %v", i, err)
		}
	}
}

func TestRunCommitsGeneratedMessage(t *testing.T) {
	deps, _, _ := testDeps()
	git := &fakeGit{diff: "diff --git a b"}
	deps.Git = git
	deps.Config = &fakeConfig{cfg: config.Config{APIKey: "sk-test"}}
	deps.Provider = fakeFactory{provider: &fakeProvider{messages: []string{"feat: add feature"}}}
	deps.Confirm = &scriptedConfirm{actions: []confirmStep{{action: "commit"}}}

	cmd := NewRootCommand(deps)
	if err := execute(cmd); err != nil {
		t.Fatalf("run returned error: %v", err)
	}

	if len(git.committed) != 1 || git.committed[0] != "feat: add feature" {
		t.Errorf("committed = %v, want [feat: add feature]", git.committed)
	}
}

func TestRunQuitDoesNotCommit(t *testing.T) {
	deps, _, _ := testDeps()
	git := &fakeGit{diff: "diff"}
	deps.Git = git
	deps.Config = &fakeConfig{cfg: config.Config{APIKey: "sk-test"}}
	deps.Confirm = &scriptedConfirm{actions: []confirmStep{{action: "quit"}}}

	cmd := NewRootCommand(deps)
	if err := execute(cmd); err != nil {
		t.Fatalf("run returned error: %v", err)
	}

	if len(git.committed) != 0 {
		t.Errorf("expected no commits on quit, got %v", git.committed)
	}
}

func TestRunRegenerateReusesDiff(t *testing.T) {
	deps, _, _ := testDeps()
	git := &fakeGit{diff: "diff"}
	provider := &fakeProvider{messages: []string{"feat: first", "feat: second"}}
	deps.Git = git
	deps.Config = &fakeConfig{cfg: config.Config{APIKey: "sk-test"}}
	deps.Provider = fakeFactory{provider: provider}
	deps.Confirm = &scriptedConfirm{actions: []confirmStep{
		{action: "regenerate"},
		{action: "commit"},
	}}

	cmd := NewRootCommand(deps)
	if err := execute(cmd); err != nil {
		t.Fatalf("run returned error: %v", err)
	}

	if provider.calls != 2 {
		t.Errorf("Generate called %d times, want 2", provider.calls)
	}
	if len(git.committed) != 1 || git.committed[0] != "feat: second" {
		t.Errorf("committed = %v, want [feat: second]", git.committed)
	}
}

func TestRunNotConfiguredReturnsAIError(t *testing.T) {
	deps, _, _ := testDeps()
	deps.Git = &fakeGit{diff: "diff"}
	deps.Provider = fakeFactory{err: ai.ErrNotConfigured}

	cmd := NewRootCommand(deps)
	err := execute(cmd)
	if !errors.Is(err, ai.ErrNotConfigured) {
		t.Fatalf("error = %v, want ErrNotConfigured", err)
	}
	if got := ExitCode(err); got != ExitAIError {
		t.Errorf("ExitCode = %d, want %d", got, ExitAIError)
	}
}

func TestRunGitFailurePropagates(t *testing.T) {
	deps, _, _ := testDeps()
	deps.Git = &fakeGit{isRepoErr: errBoom}

	cmd := NewRootCommand(deps)
	if err := execute(cmd); err == nil {
		t.Fatal("expected error when not in a git repo, got nil")
	}
}
