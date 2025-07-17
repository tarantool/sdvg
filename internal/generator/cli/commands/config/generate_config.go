package config

import (
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"

	"sdvg/internal/generator/cli/commands"
	"sdvg/internal/generator/cli/openai"
	"sdvg/internal/generator/cli/options"
	"sdvg/internal/generator/cli/render"
	"sdvg/internal/generator/cli/utils"
)

// generateConfigOptions type is used to describe 'generate-config' options.
type generateConfigOptions struct {
	openAI                   openai.Service
	renderer                 render.Renderer
	generationConfigSavePath string
	extraFilePath            string
	extraInput               bool
}

// NewGenerateConfigCommand creates 'generate-config' command for CLI.
func NewGenerateConfigCommand(cliOpts *options.CliOptions) *cobra.Command {
	opts := &generateConfigOptions{}

	cmd := &cobra.Command{
		Use:                   "generate-config [FLAGS] [COMMAND]",
		Short:                 "Generates generation config using one of the modes",
		Args:                  commands.NoArgs,
		DisableFlagsInUseLine: true,
		PreRun: func(_ *cobra.Command, _ []string) {
			opts.renderer = cliOpts.Renderer()
			opts.openAI = cliOpts.OpenAI()
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			return utils.ChooseCommand(cmd, args, opts.renderer)
		},
	}

	cmd.SetOut(cliOpts.Out())

	setupConfigCmdFlags(cmd.PersistentFlags(), opts)

	cmd.AddCommand(
		NewDescriptionCommand(cliOpts, opts),
		NewDataSampleCommand(cliOpts, opts),
		NewSQLQueryCommand(cliOpts, opts),
	)

	return cmd
}

// setupConfigCmdFlags sets flags for 'generate-config' command and bind them to generateConfigOptions fields.
func setupConfigCmdFlags(flags *pflag.FlagSet, opts *generateConfigOptions) {
	flags.StringVarP(
		&opts.generationConfigSavePath,
		commands.GenerationConfigSavePathFlag,
		commands.GenerationConfigSavePathShortFlag,
		commands.GenerationConfigSavePathDefaultValue,
		commands.GenerationConfigSavePathUsage,
	)
}
