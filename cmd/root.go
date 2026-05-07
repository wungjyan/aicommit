package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
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
	// TODO: implement main flow
	// 1. Get staged diff
	// 2. Generate commit message via AI
	// 3. Show confirmation prompt
	// 4. Execute git commit
	fmt.Println("TODO: implement main flow")
	return nil
}

func Execute() error {
	return rootCmd.Execute()
}

func SetVersionInfo(v, c, d string) {
	version = v
	commit = c
	date = d
}
