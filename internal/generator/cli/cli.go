package cli

import (
	"context"
	"fmt"
	"log/slog"
	"os"

	"github.com/pkg/errors"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
	"github.com/tarantool/sdvg/internal/generator/cli/commands"
	"github.com/tarantool/sdvg/internal/generator/cli/commands/sdvg"
	clierrors "github.com/tarantool/sdvg/internal/generator/cli/errors"
	openaiService "github.com/tarantool/sdvg/internal/generator/cli/openai/general"
	"github.com/tarantool/sdvg/internal/generator/cli/options"
	"github.com/tarantool/sdvg/internal/generator/cli/render/prompt"
	"github.com/tarantool/sdvg/internal/generator/logger/handlers"
	openaiAPI "github.com/tarantool/sdvg/internal/openai/general"
)

// Cli type is used to describe SDVG CLI.
type Cli struct {
	opts *options.CliOptions
	cmd  *cobra.Command
}

func NewCli(opts *options.CliOptions) *Cli {
	return &Cli{
		opts: opts,
		cmd:  sdvg.NewSDVGCommand(opts),
	}
}

func (cli *Cli) MustSetup() {
	err := cli.handleAppFlags()
	if err != nil {
		_, _ = fmt.Fprintln(cli.cmd.OutOrStdout(), err.Error())

		os.Exit(1)
	}

	err = cli.initialize()
	if err != nil {
		_, _ = fmt.Fprintln(cli.cmd.OutOrStdout(), err.Error())

		os.Exit(1)
	}
}

func (cli *Cli) Run(ctx context.Context) error {
	var usageErr *clierrors.UsageError

	err := cli.cmd.ExecuteContext(ctx)
	if err != nil && errors.As(err, &usageErr) {
		_, _ = fmt.Fprintln(cli.cmd.OutOrStdout(), err.Error())

		os.Exit(1)
	}

	return err //nolint:wrapcheck
}

func (cli *Cli) Options() *options.CliOptions {
	return cli.opts
}

// handleRootFlags parses flags of root command before executing it.
func (cli *Cli) handleAppFlags() error {
	cmd := cli.cmd

	flags := pflag.NewFlagSet(cmd.Name(), pflag.ContinueOnError)
	flags.SetInterspersed(false)

	flags.AddFlagSet(cmd.Flags())
	flags.AddFlagSet(cmd.PersistentFlags())

	if err := flags.Parse(os.Args[1:]); err != nil {
		return commands.FlagErrorFunc(cmd, err)
	}

	return nil
}

// Initialize initializes the SDVG CLI, configuring it using config and flags.
func (cli *Cli) initialize() error {
	cliOpts := cli.opts

	appConfig := cliOpts.AppConfig()
	sdvgOpts := cliOpts.SDVGOpts()

	// set tty mode
	if !*sdvgOpts.NoTTY.Changed && !*sdvgOpts.TTY.Changed {
		cliOpts.SetUseTTY(sdvgOpts.TTY.Value)
	} else {
		cliOpts.SetUseTTY(*sdvgOpts.TTY.Changed)
	}

	// merge values from config and flags
	configPath := sdvgOpts.ConfigPath

	err := appConfig.ParseFromFile(configPath)
	if err != nil {
		return errors.WithMessage(err, "error during initializing cli")
	}

	if sdvgOpts.OpenAIAPIKey != "" {
		appConfig.OpenAI.APIKey = sdvgOpts.OpenAIAPIKey
	}

	if sdvgOpts.OpenAIBaseURL != "" {
		appConfig.OpenAI.BaseURL = sdvgOpts.OpenAIBaseURL
	}

	if sdvgOpts.OpenAIModel != "" {
		appConfig.OpenAI.Model = sdvgOpts.OpenAIModel
	}

	// setup logger
	logLevel := slog.LevelInfo
	if sdvgOpts.DebugMode {
		logLevel = slog.LevelDebug
	}

	handlerOpts := &slog.HandlerOptions{
		Level: logLevel,
	}

	var logHandler slog.Handler

	if appConfig.LogFormat == "json" {
		logHandler = slog.NewJSONHandler(cliOpts.Out(), handlerOpts)
	} else {
		logHandler = handlers.NewTextHandler(cliOpts.Out(), handlerOpts)
	}

	slog.SetDefault(slog.New(logHandler))

	// setup renderer
	renderer := prompt.NewRenderer(cliOpts.In(), cliOpts.Out(), cliOpts.UseTTY())
	cliOpts.SetRenderer(renderer)

	// setup Open AI service
	openAIAPI := openaiAPI.NewOpenAIAPI(appConfig.OpenAI)
	openAIService := openaiService.NewOpenAIService(openAIAPI)
	cliOpts.SetOpenAI(openAIService)

	return nil
}
