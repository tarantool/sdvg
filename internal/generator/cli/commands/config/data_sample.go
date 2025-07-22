//nolint:dupl
package config

import (
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
	"github.com/tarantool/sdvg/internal/generator/cli/commands"
	"github.com/tarantool/sdvg/internal/generator/cli/options"
)

// NewDataSampleCommand creates 'data-sample' command for CLI.
func NewDataSampleCommand(cliOpts *options.CliOptions, opts *generateConfigOptions) *cobra.Command {
	cmd := &cobra.Command{
		Use:                   "data-sample [FLAGS]",
		Short:                 "Generates generation config according to data samples",
		Args:                  commands.NoArgs,
		DisableFlagsInUseLine: true,
		PreRun: func(_ *cobra.Command, _ []string) {
			opts.renderer = cliOpts.Renderer()
			opts.openAI = cliOpts.OpenAI()
		},
		RunE: func(cmd *cobra.Command, _ []string) error {
			err := generate(cmd.Context(), opts, dataSample)
			if err != nil {
				return errors.WithMessagef(err, "failed to generate config")
			}

			return nil
		},
	}

	cmd.SetOut(cliOpts.Out())

	setupDataSampleCmdFlags(cmd.Flags(), opts)

	return cmd
}

// setupDataSampleCmdFlags sets flags for 'data-sample' command and bind them to generateConfigOptions fields.
func setupDataSampleCmdFlags(flags *pflag.FlagSet, opts *generateConfigOptions) {
	flags.BoolVarP(
		&opts.extraInput,
		commands.ExtraInputFlag,
		commands.ExtraInputShortFlag,
		commands.ExtraInputDefaultValue,
		commands.ExtraInputUsage,
	)

	flags.StringVarP(
		&opts.extraFilePath,
		commands.ExtraFileFlag,
		commands.ExtraFileShortFlag,
		commands.ExtraFileDefaultValue,
		commands.ExtraSampleFileUsage,
	)
}
