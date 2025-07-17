package version

import (
	"fmt"

	"github.com/spf13/cobra"

	"sdvg/internal/generator/cli/commands"
	"sdvg/internal/generator/cli/options"
)

// NewVersionCommand creates 'version' command for CLI.
func NewVersionCommand(cliOpts *options.CliOptions) *cobra.Command {
	cmd := &cobra.Command{
		Use:                   "version",
		Short:                 "Show SDVG version",
		Args:                  commands.NoArgs,
		DisableFlagsInUseLine: true,
		Run: func(cmd *cobra.Command, _ []string) {
			versionPrompt := "SDVG version " + cliOpts.Version()

			_, _ = fmt.Fprintln(cmd.OutOrStdout(), versionPrompt)
		},
	}

	cmd.SetOut(cliOpts.Out())

	return cmd
}
