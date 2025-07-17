package validate

import (
	"context"
	"fmt"
	"io"

	"github.com/pkg/errors"
	"github.com/spf13/cobra"

	"sdvg/internal/generator/cli/commands"
	"sdvg/internal/generator/cli/options"
	"sdvg/internal/generator/cli/render"
	"sdvg/internal/generator/cli/utils"
	"sdvg/internal/generator/models"
)

// validateOptions type is used to describe 'validate-config' command options.
type validateOptions struct {
	renderer             render.Renderer
	generationConfigPath string
}

// NewValidateConfigCommand creates 'validate-config' command for CLI.
func NewValidateConfigCommand(cliOpts *options.CliOptions) *cobra.Command {
	opts := &validateOptions{}

	cmd := &cobra.Command{
		Use:                   "validate-config [PATH]",
		Short:                 "Validate models config",
		Args:                  commands.RequiresMaxArgs(1),
		DisableFlagsInUseLine: true,
		PreRun: func(_ *cobra.Command, _ []string) {
			opts.renderer = cliOpts.Renderer()
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			err := getGenerationConfigFilePath(cmd.Context(), opts, args)
			if err != nil {
				return errors.WithMessagef(err, "failed to get generation config file path")
			}

			err = runValidate(cmd.OutOrStdout(), opts)
			if err != nil {
				return err
			}

			return nil
		},
	}

	cmd.SetOut(cliOpts.Out())

	return cmd
}

// getGenerationConfigFilePath gets generation config file path from arguments or user input.
func getGenerationConfigFilePath(ctx context.Context, opts *validateOptions, args []string) error {
	if len(args) > 0 {
		opts.generationConfigPath = args[0]

		return nil
	}

	filePath, err := opts.renderer.InputMenu(
		ctx,
		"Enter path to generation config file",
		utils.ValidateFileFormat(".yml", ".yaml", ".json"),
	)
	if err != nil {
		return err
	}

	opts.generationConfigPath = filePath

	return nil
}

// runValidate executes an `validate-config` command.
func runValidate(out io.Writer, opts *validateOptions) error {
	var generatorCfg models.GenerationConfig

	err := generatorCfg.ParseFromFile(opts.generationConfigPath)
	if err != nil {
		return err
	}

	_, _ = fmt.Fprintln(out, "Generation config is valid")

	return nil
}
