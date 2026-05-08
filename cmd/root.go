package cmd

import (
	"errors"
	"fmt"

	"github.com/spf13/cobra"
	"github.com/wungjyan/aicommit/internal/ai"
	"github.com/wungjyan/aicommit/internal/config"
	"github.com/wungjyan/aicommit/internal/git"
	"github.com/wungjyan/aicommit/internal/prompt"
	"github.com/wungjyan/aicommit/internal/ui"
)

var (
	version = "dev"
	commit  = "none"
	date    = "unknown"
)

var rootCmd = &cobra.Command{
	Use:   "aicommit",
	Short: "AI-powered Git commit message generator",
	Long: `aicommit generates Conventional Commit messages using AI.

It reads your staged changes (git diff --cached) and uses your
existing AI environment (API key or local CLI tool) to generate
a commit message following the Conventional Commits specification.

Usage:
  git add .
  aicommit`,
	RunE: run,
}

var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Print version information",
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Printf("aicommit %s (commit: %s, built: %s)\n", version, commit, date)
	},
}

func init() {
	rootCmd.AddCommand(versionCmd)
}

func run(cmd *cobra.Command, args []string) error {
	if err := git.IsGitRepo(); err != nil {
		return err
	}

	diff, err := git.GetStagedDiff()
	if err != nil {
		return err
	}

	diff = git.TruncateDiff(diff, 0) // 0 = use default limit

	cfg, err := config.LoadConfig()
	if err != nil {
		return err
	}

	provider, err := ai.NewOpenAIProvider(cfg)
	if err != nil {
		if errors.Is(err, ai.ErrNotConfigured) {
			ui.Error("AI is not configured yet.")
			fmt.Println()
			ui.Info("Run `aicommit ai --setup` to set up your API key first.")
			return nil
		}
		return err
	}

	var message string
	spinErr := ui.Spinner("Generating commit message", func() error {
		message, err = provider.Generate(cmd.Context(), diff)
		return err
	})
	if spinErr != nil {
		return spinErr
	}

	for {
		valid := prompt.ValidateMessage(message) == nil
		if !valid {
			ui.Warn("Message does not follow Conventional Commits format.")
		}

		action, editedMsg, err := prompt.Confirm(message, valid)
		if err != nil {
			return err
		}

		switch action {
		case "commit":
			if err := git.Commit(editedMsg); err != nil {
				return err
			}
			ui.Success("Committed: " + editedMsg)
			return nil
		case "edit":
			message = editedMsg
			continue
		case "regenerate":
			spinErr := ui.Spinner("Regenerating commit message", func() error {
				message, err = provider.Generate(cmd.Context(), diff)
				return err
			})
			if spinErr != nil {
				return spinErr
			}
			continue
		case "quit":
			ui.Info("Aborted.")
			return nil
		}
	}
}

func Execute() error {
	// Silence cobra's default error printing so we can format it ourselves.
	rootCmd.SilenceErrors = true
	rootCmd.SilenceUsage = true

	if err := rootCmd.Execute(); err != nil {
		ui.Error(err.Error())
		return err
	}
	return nil
}

func SetVersionInfo(v, c, d string) {
	version = v
	commit = c
	date = d
}
