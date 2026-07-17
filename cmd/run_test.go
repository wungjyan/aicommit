package cmd

import (
	"context"
	"slices"
	"strings"
	"testing"

	"github.com/wungjyan/aicommit/internal/config"
	"github.com/wungjyan/aicommit/internal/git"
)

func TestCommitWorkflowRegenerateReusesTruncatedDiff(t *testing.T) {
	deps, _, _ := testDeps()
	largeDiff := strings.Repeat("x", 80*1024+1)
	provider := &fakeProvider{messages: []string{"feat: first", "feat: second"}}
	deps.Git = &fakeGit{diff: largeDiff}
	deps.Config = &fakeConfig{cfg: config.Config{APIKey: "sk-test"}}
	deps.Provider = fakeFactory{provider: provider}
	deps.Confirm = &scriptedConfirm{actions: []confirmStep{
		{action: "regenerate"},
		{action: "commit"},
	}}

	if err := NewCommitWorkflow(deps).Run(context.Background()); err != nil {
		t.Fatalf("workflow returned error: %v", err)
	}

	if len(provider.diffs) != 2 {
		t.Fatalf("Generate received %d diffs, want 2", len(provider.diffs))
	}
	wantDiff := git.TruncateDiff(largeDiff, 0)
	for i, got := range provider.diffs {
		if got != wantDiff {
			t.Errorf("Generate diff %d = %q, want truncated diff", i, got)
		}
	}
}

func TestCommitWorkflowRevalidatesEditedMessage(t *testing.T) {
	deps, _, _ := testDeps()
	gitService := &fakeGit{diff: "diff"}
	ui := &recordingUI{}
	deps.Git = gitService
	deps.Config = &fakeConfig{cfg: config.Config{APIKey: "sk-test"}}
	deps.UI = ui
	deps.Confirm = &scriptedConfirm{actions: []confirmStep{
		{action: "edit", edited: "not a commit message"},
		{action: "quit"},
	}}

	if err := NewCommitWorkflow(deps).Run(context.Background()); err != nil {
		t.Fatalf("workflow returned error: %v", err)
	}
	if len(ui.warns) != 1 {
		t.Errorf("validation warnings = %v, want one warning after edit", ui.warns)
	}
	if len(gitService.committed) != 0 {
		t.Errorf("committed = %v, want no commit", gitService.committed)
	}
}

func TestCommitWorkflowValidEditedMessageCommitsWithoutRegenerating(t *testing.T) {
	deps, _, _ := testDeps()
	gitService := &fakeGit{diff: "diff"}
	provider := &fakeProvider{messages: []string{"feat: generated message"}}
	deps.Git = gitService
	deps.Config = &fakeConfig{cfg: config.Config{APIKey: "sk-test"}}
	deps.Provider = fakeFactory{provider: provider}
	deps.Confirm = &scriptedConfirm{actions: []confirmStep{
		{action: "edit", edited: "fix: edited message"},
		{action: "commit"},
	}}

	if err := NewCommitWorkflow(deps).Run(context.Background()); err != nil {
		t.Fatalf("workflow returned error: %v", err)
	}

	if provider.calls != 1 {
		t.Errorf("Generate called %d times, want 1", provider.calls)
	}
	if got, want := gitService.committed, []string{"fix: edited message"}; !slices.Equal(got, want) {
		t.Errorf("committed = %v, want %v", got, want)
	}
}

func TestCommitWorkflowCommitFailurePropagates(t *testing.T) {
	deps, _, _ := testDeps()
	deps.Git = &fakeGit{diff: "diff", commitErr: errBoom}
	deps.Config = &fakeConfig{cfg: config.Config{APIKey: "sk-test"}}
	deps.Confirm = &scriptedConfirm{actions: []confirmStep{{action: "commit"}}}

	if err := NewCommitWorkflow(deps).Run(context.Background()); err != errBoom {
		t.Fatalf("workflow error = %v, want %v", err, errBoom)
	}
}
