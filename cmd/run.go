package cmd

import (
	"context"
	"fmt"
	"io"

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
	editor   MessageEditor
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
		editor:   deps.Editor,
	}
}

// Run reads the staged diff once, generates a message, then drives the
// confirmation state machine. Regeneration deliberately reuses the same
// bounded diff so each attempt sees an identical input.
func (w *CommitWorkflow) Run(ctx context.Context) error {
	return w.runWithOptions(ctx, runOptions{}, io.Discard)
}

// runWithOptions applies the default workflow in an automatic or interactive
// mode. The root command validates the mode combinations before calling it.
func (w *CommitWorkflow) runWithOptions(ctx context.Context, options runOptions, out io.Writer) error {
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
		return err
	}

	message, err := w.generate(ctx, provider, diff, "Generating commit message", !options.dryRun)
	if err != nil {
		return err
	}
	if options.edit {
		message, err = w.editor.Edit(message)
		if err != nil {
			return err
		}
	}
	if options.dryRun {
		_, writeErr := fmt.Fprintln(out, message)
		if writeErr != nil {
			return writeErr
		}
		return validateGeneratedMessage(message)
	}
	if options.yes {
		if err := validateGeneratedMessage(message); err != nil {
			return err
		}
		return w.commit(message)
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
			if err := validateGeneratedMessage(editedMessage); err != nil {
				return err
			}
			return w.commit(editedMessage)
		case "edit":
			message = editedMessage
		case "regenerate":
			message, err = w.generate(ctx, provider, diff, "Regenerating commit message", true)
			if err != nil {
				return err
			}
		case "quit":
			w.ui.Info("Aborted.")
			return nil
		}
	}
}

func validateGeneratedMessage(message string) error {
	if err := prompt.ValidateMessage(message); err != nil {
		return invalidMessageError(err)
	}
	return nil
}

func (w *CommitWorkflow) commit(message string) error {
	if err := w.git.Commit(message); err != nil {
		return err
	}
	w.ui.Success("Committed: " + message)
	return nil
}

func (w *CommitWorkflow) generate(ctx context.Context, provider Provider, diff, label string, showProgress bool) (string, error) {
	var message string
	generate := func() error {
		var err error
		message, err = provider.Generate(ctx, diff)
		return err
	}
	if !showProgress {
		return message, generate()
	}
	err := w.ui.Spinner(label, generate)
	return message, err
}
