package config

import (
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
	"github.com/tarantool/sdvg/internal/generator/cli/commands"
	"github.com/tarantool/sdvg/internal/generator/cli/options"
)

// NewDescriptionCommand creates 'description' command for CLI.
func NewDescriptionCommand(cliOpts *options.CliOptions, opts *generateConfigOptions) *cobra.Command {
	cmd := &cobra.Command{
		Use:                   "description",
		Short:                 "Generates generation config according to description",
		Args:                  commands.NoArgs,
		DisableFlagsInUseLine: true,
		PreRun: func(_ *cobra.Command, _ []string) {
			opts.renderer = cliOpts.Renderer()
			opts.openAI = cliOpts.OpenAI()
		},
		RunE: func(cmd *cobra.Command, _ []string) error {
			err := generate(cmd.Context(), opts, description)
			if err != nil {
				return errors.WithMessagef(err, "failed to generate config")
			}

			return nil
		},
	}

	cmd.SetOut(cliOpts.Out())

	return cmd
}
