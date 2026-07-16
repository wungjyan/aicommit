package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
)

// newConfigPathCommand prints the absolute config file path to stdout, for
// inspection, backup or scripting.
func newConfigPathCommand(deps Dependencies) *cobra.Command {
	return &cobra.Command{
		Use:   "path",
		Short: "Print the absolute path of the config file",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			path, err := deps.Config.Path()
			if err != nil {
				return err
			}
			fmt.Fprintln(cmd.OutOrStdout(), path)
			return nil
		},
	}
}
