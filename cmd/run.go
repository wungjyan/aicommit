package cmd

import (
	"context"
	"errors"

	"github.com/wungjyan/aicommit/internal/ai"
	"github.com/wungjyan/aicommit/internal/git"
	"github.com/wungjyan/aicommit/internal/prompt"
)

// CommitWorkflow coordinates the default generate, confirm, and commit flow.
// Terminal interaction itself stays behind Confirmer, leaving this state
// transition logic independent of Cobra and concrete terminal streams.
type CommitWorkflow struct {
	git      GitService
	config   ConfigStore
	provider ProviderFactory
	ui       UI
	confirm  Confirmer
}

// NewCommitWorkflow creates the default command workflow from command
// dependencies. It is intentionally constructed per command execution, so no
// generation or confirmation state can leak between invocations.
func NewCommitWorkflow(deps Dependencies) *CommitWorkflow {
	return &CommitWorkflow{
		git:      deps.Git,
		config:   deps.Config,
		provider: deps.Provider,
		ui:       deps.UI,
		confirm:  deps.Confirm,
	}
}

// Run reads the staged diff once, generates a message, then drives the
// confirmation state machine. Regeneration deliberately reuses the same
// bounded diff so each attempt sees an identical input.
func (w *CommitWorkflow) Run(ctx context.Context) error {
	if err := w.git.IsGitRepo(); err != nil {
		return err
	}

	diff, err := w.git.GetStagedDiff()
	if err != nil {
		return err
	}
	diff = git.TruncateDiff(diff, 0) // 0 = use default limit

	cfg, err := w.config.Load()
	if err != nil {
		return err
	}

	provider, err := w.provider.New(cfg)
	if err != nil {
		if errors.Is(err, ai.ErrNotConfigured) {
			w.ui.Error("AI is not configured yet.")
			w.ui.Info("Run `aicommit config setup` to set up your API key first.")
			return nil
		}
		return err
	}

	message, err := w.generate(ctx, provider, diff, "Generating commit message")
	if err != nil {
		return err
	}

	for {
		valid := prompt.ValidateMessage(message) == nil
		if !valid {
			w.ui.Warn("Message does not follow Conventional Commits format.")
		}

		action, editedMessage, err := w.confirm.Confirm(message, valid)
		if err != nil {
			return err
		}

		switch action {
		case "commit":
			if err := w.git.Commit(editedMessage); err != nil {
				return err
			}
			w.ui.Success("Committed: " + editedMessage)
			return nil
		case "edit":
			message = editedMessage
		case "regenerate":
			message, err = w.generate(ctx, provider, diff, "Regenerating commit message")
			if err != nil {
				return err
			}
		case "quit":
			w.ui.Info("Aborted.")
			return nil
		}
	}
}

func (w *CommitWorkflow) generate(ctx context.Context, provider Provider, diff, label string) (string, error) {
	var message string
	err := w.ui.Spinner(label, func() error {
		var err error
		message, err = provider.Generate(ctx, diff)
		return err
	})
	return message, err
}
