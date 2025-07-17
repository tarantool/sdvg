//nolint:dupl
package config

import (
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"

	"sdvg/internal/generator/cli/commands"
	"sdvg/internal/generator/cli/options"
)

// NewSQLQueryCommand creates 'sql-query' command for CLI.
func NewSQLQueryCommand(cliOpts *options.CliOptions, opts *generateConfigOptions) *cobra.Command {
	cmd := &cobra.Command{
		Use:                   "sql-query [FLAGS]",
		Short:                 "Generates generation config according to sql query",
		Args:                  commands.NoArgs,
		DisableFlagsInUseLine: true,
		PreRun: func(_ *cobra.Command, _ []string) {
			opts.renderer = cliOpts.Renderer()
			opts.openAI = cliOpts.OpenAI()
		},
		RunE: func(cmd *cobra.Command, _ []string) error {
			err := generate(cmd.Context(), opts, sqlQuery)
			if err != nil {
				return errors.WithMessagef(err, "failed to generate config")
			}

			return nil
		},
	}

	cmd.SetOut(cliOpts.Out())

	setupSQLQueryCmdFlags(cmd.Flags(), opts)

	return cmd
}

// setupSQLQueryCmdFlags sets flags for 'sql-query' command and bind them to generateConfigOptions fields.
func setupSQLQueryCmdFlags(flags *pflag.FlagSet, opts *generateConfigOptions) {
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
		commands.ExtraSQLFileUsage,
	)
}
