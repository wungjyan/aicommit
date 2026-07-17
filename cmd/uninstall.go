package cmd

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"
	"github.com/wungjyan/aicommit/internal/config"
)

func newUninstallCommand(deps Dependencies) *cobra.Command {
	var purge bool
	var yes bool

	cmd := &cobra.Command{
		Use:   "uninstall",
		Short: "Remove the installed aicommit binary",
		Args:  noArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			if purge && !yes {
				confirmed, err := confirmPurge(deps, cmd.OutOrStdout())
				if err != nil {
					return err
				}
				if !confirmed {
					fmt.Fprintln(cmd.OutOrStdout(), "Uninstall cancelled.")
					return nil
				}
			}

			result, err := deps.Uninstaller.Uninstall(purge)
			if err != nil {
				return err
			}
			fmt.Fprintf(cmd.OutOrStdout(), "Removed aicommit binary: %s\n", result.Executable)
			if !purge {
				fmt.Fprintln(cmd.OutOrStdout(), "Configuration was kept. Use --purge to remove it too.")
				return nil
			}
			if result.ConfigRemoved {
				fmt.Fprintf(cmd.OutOrStdout(), "Removed configuration: %s\n", result.ConfigDir)
			} else {
				fmt.Fprintln(cmd.OutOrStdout(), "No aicommit configuration was found.")
			}
			return nil
		},
	}

	cmd.Flags().BoolVar(&purge, "purge", false, "also remove saved configuration")
	cmd.Flags().BoolVarP(&yes, "yes", "y", false, "skip the --purge confirmation")
	return cmd
}

func confirmPurge(deps Dependencies, out io.Writer) (bool, error) {
	isTTY := deps.IsTTY
	if isTTY == nil {
		isTTY = isTerminal
	}
	if !isTTY(deps.In) {
		return false, usageErrorf("--purge requires interactive confirmation; use --yes to continue")
	}

	fmt.Fprint(out, "Also remove ~/.aicommit and its saved API configuration? [y/N] ")
	input, err := bufio.NewReader(deps.In).ReadString('\n')
	if err != nil && !errors.Is(err, io.EOF) {
		return false, fmt.Errorf("failed to read confirmation: %w", err)
	}
	input = strings.TrimSpace(strings.ToLower(input))
	return input == "y" || input == "yes", nil
}

type systemUninstaller struct{}

func (systemUninstaller) Uninstall(purge bool) (UninstallResult, error) {
	executable, err := os.Executable()
	if err != nil {
		return UninstallResult{}, fmt.Errorf("cannot find the running aicommit binary: %w", err)
	}
	if err := removeCurrentExecutable(executable); err != nil {
		return UninstallResult{}, fmt.Errorf("failed to remove aicommit binary %q: %w", executable, err)
	}

	result := UninstallResult{Executable: executable}
	if !purge {
		return result, nil
	}

	configPath, err := config.ConfigPath()
	if err != nil {
		return UninstallResult{}, err
	}
	configDir := filepath.Dir(configPath)
	if _, err := os.Stat(configDir); err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return result, nil
		}
		return UninstallResult{}, fmt.Errorf("failed to inspect configuration directory: %w", err)
	}
	if err := os.RemoveAll(configDir); err != nil {
		return UninstallResult{}, fmt.Errorf("failed to remove configuration directory: %w", err)
	}
	result.ConfigDir = configDir
	result.ConfigRemoved = true
	return result, nil
}
