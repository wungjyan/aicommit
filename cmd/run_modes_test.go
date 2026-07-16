package cmd

import (
	"errors"
	"slices"
	"testing"

	"github.com/wungjyan/aicommit/internal/config"
)

func TestDryRunWritesOnlyMessageAndSkipsConfirmation(t *testing.T) {
	deps, out, _ := testDeps()
	gitService := &fakeGit{diff: "diff"}
	provider := &fakeProvider{messages: []string{"feat: add dry run"}}
	confirm := &scriptedConfirm{actions: []confirmStep{{action: "commit"}}}
	deps.Git = gitService
	deps.Config = &fakeConfig{cfg: config.Config{APIKey: "sk-test"}}
	deps.Provider = fakeFactory{provider: provider}
	deps.Confirm = confirm

	if err := execute(NewRootCommand(deps), "--dry-run"); err != nil {
		t.Fatalf("dry-run returned error: %v", err)
	}
	if got, want := out.String(), "feat: add dry run\n"; got != want {
		t.Errorf("stdout = %q, want %q", got, want)
	}
	if len(gitService.committed) != 0 {
		t.Errorf("committed = %v, want no commits", gitService.committed)
	}
	if confirm.calls != 0 {
		t.Errorf("Confirm called %d times, want 0", confirm.calls)
	}
}

func TestDryRunInvalidMessageWritesItAndFails(t *testing.T) {
	deps, out, _ := testDeps()
	gitService := &fakeGit{diff: "diff"}
	deps.Git = gitService
	deps.Config = &fakeConfig{cfg: config.Config{APIKey: "sk-test"}}
	deps.Provider = fakeFactory{provider: &fakeProvider{messages: []string{"invalid message"}}}

	err := execute(NewRootCommand(deps), "--dry-run")
	if err == nil {
		t.Fatal("expected invalid dry-run message to fail")
	}
	if got, want := out.String(), "invalid message\n"; got != want {
		t.Errorf("stdout = %q, want %q", got, want)
	}
	if len(gitService.committed) != 0 {
		t.Errorf("committed = %v, want no commits", gitService.committed)
	}
}

func TestYesCommitsValidMessageWithoutConfirmation(t *testing.T) {
	deps, _, _ := testDeps()
	gitService := &fakeGit{diff: "diff"}
	confirm := &scriptedConfirm{}
	deps.Git = gitService
	deps.Config = &fakeConfig{cfg: config.Config{APIKey: "sk-test"}}
	deps.Confirm = confirm

	if err := execute(NewRootCommand(deps), "--yes"); err != nil {
		t.Fatalf("--yes returned error: %v", err)
	}
	if got, want := gitService.committed, []string{"feat: add thing"}; !slices.Equal(got, want) {
		t.Errorf("committed = %v, want %v", got, want)
	}
	if confirm.calls != 0 {
		t.Errorf("Confirm called %d times, want 0", confirm.calls)
	}
}

func TestYesRejectsInvalidMessageWithoutCommit(t *testing.T) {
	deps, _, _ := testDeps()
	gitService := &fakeGit{diff: "diff"}
	deps.Git = gitService
	deps.Config = &fakeConfig{cfg: config.Config{APIKey: "sk-test"}}
	deps.Provider = fakeFactory{provider: &fakeProvider{messages: []string{"invalid message"}}}

	if err := execute(NewRootCommand(deps), "--yes"); err == nil {
		t.Fatal("expected --yes with invalid message to fail")
	}
	if len(gitService.committed) != 0 {
		t.Errorf("committed = %v, want no commits", gitService.committed)
	}
}

func TestYesEditEditsThenCommitsWithoutConfirmation(t *testing.T) {
	deps, _, _ := testDeps()
	gitService := &fakeGit{diff: "diff"}
	editor := &fakeEditor{edited: "fix: edit before commit"}
	confirm := &scriptedConfirm{}
	deps.Git = gitService
	deps.Config = &fakeConfig{cfg: config.Config{APIKey: "sk-test"}}
	deps.Editor = editor
	deps.Confirm = confirm

	if err := execute(NewRootCommand(deps), "--yes", "--edit"); err != nil {
		t.Fatalf("--yes --edit returned error: %v", err)
	}
	if got, want := gitService.committed, []string{"fix: edit before commit"}; !slices.Equal(got, want) {
		t.Errorf("committed = %v, want %v", got, want)
	}
	if editor.calls != 1 {
		t.Errorf("Edit called %d times, want 1", editor.calls)
	}
	if confirm.calls != 0 {
		t.Errorf("Confirm called %d times, want 0", confirm.calls)
	}
}

func TestEditEditsThenContinuesToConfirmation(t *testing.T) {
	deps, _, _ := testDeps()
	gitService := &fakeGit{diff: "diff"}
	editor := &fakeEditor{edited: "fix: edit before confirmation"}
	confirm := &scriptedConfirm{actions: []confirmStep{{action: "commit"}}}
	deps.Git = gitService
	deps.Config = &fakeConfig{cfg: config.Config{APIKey: "sk-test"}}
	deps.Editor = editor
	deps.Confirm = confirm

	if err := execute(NewRootCommand(deps), "--edit"); err != nil {
		t.Fatalf("--edit returned error: %v", err)
	}
	if got, want := gitService.committed, []string{"fix: edit before confirmation"}; !slices.Equal(got, want) {
		t.Errorf("committed = %v, want %v", got, want)
	}
	if editor.calls != 1 {
		t.Errorf("Edit called %d times, want 1", editor.calls)
	}
	if confirm.calls != 1 {
		t.Errorf("Confirm called %d times, want 1", confirm.calls)
	}
}

func TestRunModeValidation(t *testing.T) {
	tests := []struct {
		name string
		args []string
	}{
		{name: "dry run and yes", args: []string{"--dry-run", "--yes"}},
		{name: "dry run and edit", args: []string{"--dry-run", "--edit"}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			deps, _, _ := testDeps()
			err := execute(NewRootCommand(deps), tt.args...)
			if !errors.Is(err, ErrUsage) {
				t.Errorf("error = %v, want ErrUsage", err)
			}
		})
	}
}

func TestNonTTYRejectsInteractiveAndEditModes(t *testing.T) {
	for _, args := range [][]string{nil, {"--edit"}, {"--yes", "--edit"}} {
		deps, _, _ := testDeps()
		editor := &fakeEditor{edited: "fix: should not edit"}
		deps.Editor = editor
		deps.IsTTY = func(any) bool { return false }

		err := execute(NewRootCommand(deps), args...)
		if !errors.Is(err, ErrUsage) {
			t.Errorf("args %v: error = %v, want ErrUsage", args, err)
		}
		if editor.calls != 0 {
			t.Errorf("args %v: Edit called %d times, want 0", args, editor.calls)
		}
	}
}

func TestEditRequiresAllStreamsToBeTTY(t *testing.T) {
	deps, _, _ := testDeps()
	deps.IsTTY = func(stream any) bool { return stream != deps.Out }

	err := execute(NewRootCommand(deps), "--edit")
	if !errors.Is(err, ErrUsage) {
		t.Errorf("error = %v, want ErrUsage", err)
	}
}

func TestNoColorDisablesColorOutput(t *testing.T) {
	deps, _, _ := testDeps()
	ui := &recordingUI{}
	deps.UI = ui

	if err := execute(NewRootCommand(deps), "--dry-run"); err != nil {
		t.Fatalf("--no-color dry-run returned error: %v", err)
	}
	if ui.noColor != 0 {
		t.Fatalf("DisableColor called without --no-color: %d", ui.noColor)
	}

	deps, _, _ = testDeps()
	ui = &recordingUI{}
	deps.UI = ui
	if err := execute(NewRootCommand(deps), "--no-color", "--dry-run"); err != nil {
		t.Fatalf("--no-color --dry-run returned error: %v", err)
	}
	if ui.noColor != 1 {
		t.Errorf("DisableColor called %d times, want 1", ui.noColor)
	}
}
