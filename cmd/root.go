package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"golang.org/x/term"
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
	var options runOptions

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
		PreRunE: func(cmd *cobra.Command, args []string) error {
			if err := validateRunOptions(deps, options); err != nil {
				return err
			}
			if options.noColor {
				deps.UI.DisableColor()
			}
			return nil
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			return NewCommitWorkflow(deps).runWithOptions(cmd.Context(), options, cmd.OutOrStdout())
		},
	}

	rootCmd.SetIn(deps.In)
	rootCmd.SetOut(deps.Out)
	rootCmd.SetErr(deps.ErrOut)
	rootCmd.Flags().BoolVar(&options.dryRun, "dry-run", false, "generate and print a commit message without committing")
	rootCmd.Flags().BoolVarP(&options.yes, "yes", "y", false, "commit the generated message without confirmation")
	rootCmd.Flags().BoolVarP(&options.edit, "edit", "e", false, "edit the generated message before continuing")
	rootCmd.Flags().BoolVar(&options.noColor, "no-color", false, "disable ANSI color output")

	rootCmd.AddCommand(newVersionCommand(deps))
	rootCmd.AddCommand(newConfigCommand(deps))

	return rootCmd
}

type runOptions struct {
	dryRun  bool
	yes     bool
	edit    bool
	noColor bool
}

func validateRunOptions(deps Dependencies, options runOptions) error {
	if options.dryRun && options.yes {
		return usageErrorf("--dry-run cannot be used with --yes")
	}
	if options.dryRun && options.edit {
		return usageErrorf("--dry-run cannot be used with --edit")
	}

	isTTY := deps.IsTTY
	if isTTY == nil {
		isTTY = isTerminal
	}
	if options.edit && (!isTTY(deps.In) || !isTTY(deps.Out) || !isTTY(deps.ErrOut)) {
		return usageErrorf("--edit requires stdin, stdout, and stderr to be terminals")
	}
	if !options.dryRun && !options.yes && !isTTY(deps.In) {
		return usageErrorf("non-interactive input requires --dry-run or --yes")
	}
	return nil
}

func isTerminal(stream any) bool {
	file, ok := stream.(*os.File)
	return ok && term.IsTerminal(int(file.Fd()))
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
