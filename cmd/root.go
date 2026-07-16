package cmd

import (
	"errors"
	"fmt"

	"github.com/spf13/cobra"
	"github.com/wungjyan/aicommit/internal/ai"
	"github.com/wungjyan/aicommit/internal/git"
	"github.com/wungjyan/aicommit/internal/prompt"
)

// Build metadata, injected via -ldflags at build time. These stay as package
// variables because ldflags can only patch package-level symbols; they are read
// once in Execute and passed into the command tree as VersionInfo.
var (
	version = "dev"
	commit  = "none"
	date    = "unknown"
)

// NewRootCommand builds a fresh command tree bound to the given dependencies.
//
// Every invocation returns an independent *cobra.Command with its flags bound to
// command-local state, so tests can construct and run the tree repeatedly in one
// process without global flag pollution.
func NewRootCommand(deps Dependencies) *cobra.Command {
	rootCmd := &cobra.Command{
		Use:   "aicommit",
		Short: "AI-powered Git commit message generator",
		Long: `aicommit generates Conventional Commit messages using AI.

It reads your staged changes (git diff --cached) and uses your
existing AI environment (API key or local CLI tool) to generate
a commit message following the Conventional Commits specification.

Usage:
  git add .
  aicommit`,
		Args:          cobra.NoArgs,
		SilenceErrors: true,
		SilenceUsage:  true,
		RunE: func(cmd *cobra.Command, args []string) error {
			return run(cmd, deps)
		},
	}

	rootCmd.SetIn(deps.In)
	rootCmd.SetOut(deps.Out)
	rootCmd.SetErr(deps.ErrOut)

	rootCmd.AddCommand(newVersionCommand(deps))

	return rootCmd
}

func newVersionCommand(deps Dependencies) *cobra.Command {
	return &cobra.Command{
		Use:   "version",
		Short: "Print version information",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			v := deps.Version
			fmt.Fprintf(cmd.OutOrStdout(), "aicommit %s (commit: %s, built: %s)\n", v.Version, v.Commit, v.Date)
			return nil
		},
	}
}

func run(cmd *cobra.Command, deps Dependencies) error {
	if err := deps.Git.IsGitRepo(); err != nil {
		return err
	}

	diff, err := deps.Git.GetStagedDiff()
	if err != nil {
		return err
	}

	diff = git.TruncateDiff(diff, 0) // 0 = use default limit

	cfg, err := deps.Config.Load()
	if err != nil {
		return err
	}

	provider, err := deps.Provider.New(cfg)
	if err != nil {
		if errors.Is(err, ai.ErrNotConfigured) {
			deps.UI.Error("AI is not configured yet.")
			deps.UI.Info("Run `aicommit ai --setup` to set up your API key first.")
			return nil
		}
		return err
	}

	var message string
	spinErr := deps.UI.Spinner("Generating commit message", func() error {
		message, err = provider.Generate(cmd.Context(), diff)
		return err
	})
	if spinErr != nil {
		return spinErr
	}

	for {
		valid := prompt.ValidateMessage(message) == nil
		if !valid {
			deps.UI.Warn("Message does not follow Conventional Commits format.")
		}

		action, editedMsg, err := deps.Confirm.Confirm(message, valid)
		if err != nil {
			return err
		}

		switch action {
		case "commit":
			if err := deps.Git.Commit(editedMsg); err != nil {
				return err
			}
			deps.UI.Success("Committed: " + editedMsg)
			return nil
		case "edit":
			message = editedMsg
			continue
		case "regenerate":
			spinErr := deps.UI.Spinner("Regenerating commit message", func() error {
				message, err = provider.Generate(cmd.Context(), diff)
				return err
			})
			if spinErr != nil {
				return spinErr
			}
			continue
		case "quit":
			deps.UI.Info("Aborted.")
			return nil
		}
	}
}

// Execute builds the production command tree and runs it. Errors are printed via
// the UI adapter for colored output, mirroring the previous behavior.
func Execute() error {
	deps := productionDeps(VersionInfo{Version: version, Commit: commit, Date: date})

	rootCmd := NewRootCommand(deps)
	registerLegacyAICommand(rootCmd)

	if err := rootCmd.Execute(); err != nil {
		deps.UI.Error(err.Error())
		return err
	}
	return nil
}
