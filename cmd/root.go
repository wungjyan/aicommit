package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
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
			return NewCommitWorkflow(deps).Run(cmd.Context())
		},
	}

	rootCmd.SetIn(deps.In)
	rootCmd.SetOut(deps.Out)
	rootCmd.SetErr(deps.ErrOut)

	rootCmd.AddCommand(newVersionCommand(deps))
	rootCmd.AddCommand(newConfigCommand(deps))

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

// Execute builds the production command tree and runs it. Errors are printed via
// the UI adapter for colored output, mirroring the previous behavior.
func Execute() error {
	deps := productionDeps(VersionInfo{Version: version, Commit: commit, Date: date})
	deps.In = os.Stdin
	deps.Out = os.Stdout
	deps.ErrOut = os.Stderr

	rootCmd := NewRootCommand(deps)

	if err := rootCmd.Execute(); err != nil {
		deps.UI.Error(err.Error())
		return err
	}
	return nil
}
