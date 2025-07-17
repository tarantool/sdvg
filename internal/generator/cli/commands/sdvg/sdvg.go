package sdvg

import (
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"

	"sdvg/internal/generator/cli/commands"
	"sdvg/internal/generator/cli/commands/config"
	"sdvg/internal/generator/cli/commands/generate"
	"sdvg/internal/generator/cli/commands/serve"
	"sdvg/internal/generator/cli/commands/validate"
	"sdvg/internal/generator/cli/commands/version"
	"sdvg/internal/generator/cli/options"
	"sdvg/internal/generator/cli/streams"
	"sdvg/internal/generator/cli/utils"
)

// NewSDVGCommand creates 'sdvg' command for CLI.
func NewSDVGCommand(cliOpts *options.CliOptions) *cobra.Command {
	cobra.EnableCommandSorting = false

	opts := cliOpts.SDVGOpts()

	cmd := &cobra.Command{
		Use:                   "sdvg [FLAGS] [COMMAND]",
		Short:                 "CLI for using Synthetic Data Values Generator",
		Args:                  commands.NoArgs,
		SilenceUsage:          true,
		SilenceErrors:         true,
		TraverseChildren:      true,
		DisableFlagsInUseLine: true,
		CompletionOptions: cobra.CompletionOptions{
			DisableDefaultCmd: true,
			HiddenDefaultCmd:  true,
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			renderer := cliOpts.Renderer()
			renderer.Logo()

			return utils.ChooseCommand(cmd, args, renderer)
		},
	}

	cmd.SetOut(cliOpts.Out())

	cmd.SetFlagErrorFunc(commands.FlagErrorFunc)

	setupFlags(cmd.Flags(), opts, cliOpts.In())

	cmd.PersistentFlags().BoolP("help", "h", false, "Print usage")

	cmd.PersistentFlags().Lookup("help").Hidden = true

	cmd.MarkFlagsMutuallyExclusive(commands.TTYFlag, commands.NoTTYFlag)
	cmd.SetUsageTemplate(usageTemplate)

	cmd.AddCommand(
		generate.NewGenerateCommand(cliOpts),
		serve.NewServeCommand(cliOpts),
		config.NewGenerateConfigCommand(cliOpts),
		validate.NewValidateConfigCommand(cliOpts),
		version.NewVersionCommand(cliOpts),
	)

	return cmd
}

// // setupFlags sets flags for 'sdvg' command and bind them to rootOptions fields.
func setupFlags(flags *pflag.FlagSet, opts *options.SDVGOptions, in *streams.In) {
	flags.StringVarP(
		&opts.ConfigPath,
		commands.ConfigPathFlag,
		commands.ConfigPathShortFlag,
		commands.ConfigPathDefaultValue,
		commands.ConfigPathUsage,
	)

	flags.BoolVarP(
		&opts.TTY.Value,
		commands.TTYFlag,
		commands.TTYShortFlag,
		in.IsTerminal(),
		commands.TTYUsage,
	)

	opts.TTY.Changed = &flags.Lookup(commands.TTYFlag).Changed

	flags.BoolVarP(
		&opts.NoTTY.Value,
		commands.NoTTYFlag,
		commands.NoTTYShortFlag,
		commands.NoTTYDefaultValue,
		commands.NoTTYUsage,
	)

	opts.NoTTY.Changed = &flags.Lookup(commands.NoTTYFlag).Changed

	flags.BoolVarP(
		&opts.DebugMode,
		commands.DebugModeFlag,
		commands.DebugModeShortFlag,
		commands.DebugModeDefaultValue,
		commands.DebugModeUsage,
	)

	flags.StringVarP(
		&opts.CPUProfile,
		commands.CPUProfileFlag,
		commands.CPUProfileShortFlag,
		commands.CPUProfileDefaultValue,
		commands.CPUProfileUsage,
	)

	flags.StringVarP(
		&opts.MemoryProfile,
		commands.MemoryProfileFlag,
		commands.MemoryProfileShortFlag,
		commands.MemoryProfileDefaultValue,
		commands.MemoryProfileUsage,
	)

	flags.StringVarP(
		&opts.OpenAIAPIKey,
		commands.OpenAIAPIKeyFlag,
		commands.OpenAIAPIKeyShortFlag,
		commands.OpenAIAPIKeyDefaultValue,
		commands.OpenAIAPIKeyUsage,
	)

	flags.StringVarP(
		&opts.OpenAIBaseURL,
		commands.OpenAIBaseURLFlag,
		commands.OpenAIBaseURLShortFlag,
		commands.OpenAIBaseURLDefaultValue,
		commands.OpenAIBaseURLUsage,
	)

	flags.StringVarP(
		&opts.OpenAIModel,
		commands.OpenAIModelFlag,
		commands.OpenAIModelShortFlag,
		commands.OpenAIModelDefaultValue,
		commands.OpenAIModelUsage,
	)
}

const usageTemplate = `Usage:{{if .Runnable}}
  {{.UseLine}}{{end}}{{if gt (len .Aliases) 0}}

Aliases:
  {{.NameAndAliases}}{{end}}{{if .HasExample}}

Examples:
{{.Example}}{{end}}{{if .HasAvailableSubCommands}}{{$cmds := .Commands}}{{if eq (len .Groups) 0}}

Available Commands:{{range $cmds}}{{if (or .IsAvailableCommand (eq .Name "help"))}}
  {{rpad .Name .NamePadding }} {{.Short}}{{end}}{{end}}{{else}}{{range $group := .Groups}}

{{.Title}}{{range $cmds}}{{if (and (eq .GroupID $group.ID) (or .IsAvailableCommand (eq .Name "help")))}}
  {{rpad .Name .NamePadding }} {{.Short}}{{end}}{{end}}{{end}}{{if not .AllChildCommandsHaveGroup}}

Additional Commands:{{range $cmds}}{{if (and (eq .GroupID "") (or .IsAvailableCommand (eq .Name "help")))}}
  {{rpad .Name .NamePadding }} {{.Short}}{{end}}{{end}}{{end}}{{end}}{{end}}{{if .HasAvailableLocalFlags}}

Flags:
{{.LocalFlags.FlagUsages | trimTrailingWhitespaces}}{{end}}{{if .HasAvailableInheritedFlags}}

Global Flags:
{{.InheritedFlags.FlagUsages | trimTrailingWhitespaces}}{{end}}{{if .HasHelpSubCommands}}

Additional help topics:{{range .Commands}}{{if .IsAdditionalHelpTopicCommand}}
  {{rpad .CommandPath .CommandPathPadding}} {{.Short}}{{end}}{{end}}{{end}}{{if .HasAvailableSubCommands}}

Use "{{.CommandPath}} [command] --help" for more information about a command.{{end}}
`
