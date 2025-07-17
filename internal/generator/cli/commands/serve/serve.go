package serve

import (
	"context"
	"log/slog"
	"net/http"
	"time"

	"github.com/labstack/echo/v4"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"

	"sdvg/internal/generator/cli/commands"
	"sdvg/internal/generator/cli/openai"
	"sdvg/internal/generator/cli/options"
	"sdvg/internal/generator/models"
	"sdvg/internal/generator/usecase"
)

// serveOptions type is used to describe 'serve' command options.
type serveOptions struct {
	useCase       usecase.UseCase
	openAI        openai.Service
	listenAddress string
	readTimeout   time.Duration
	writeTimeout  time.Duration
	idleTimeout   time.Duration
}

// NewServeCommand creates 'serve' command for CLI.
func NewServeCommand(cliOpts *options.CliOptions) *cobra.Command {
	opts := &serveOptions{}

	cmd := &cobra.Command{
		Use:                   "serve [FLAGS]",
		Short:                 "Runs HTTP API for generator",
		Args:                  commands.NoArgs,
		DisableFlagsInUseLine: true,
		PreRun: func(_ *cobra.Command, _ []string) {
			opts.useCase = cliOpts.UseCase()
			opts.openAI = cliOpts.OpenAI()
		},
		RunE: func(cmd *cobra.Command, _ []string) error {
			slog.Info("generator started", slog.String("version", cliOpts.Version()))

			configureOptions(opts, cliOpts.AppConfig().HTTPConfig)

			err := runServe(cmd.Context(), opts)
			if err != nil {
				return errors.WithMessagef(err, "failed to serve http server")
			}

			slog.Info("generator finished")

			return nil
		},
	}

	cmd.SetOut(cliOpts.Out())

	setupFlags(cmd.Flags(), opts)

	return cmd
}

// setupFlags sets flags for 'serve' command and bind them to serveOptions fields.
func setupFlags(flags *pflag.FlagSet, opts *serveOptions) {
	flags.StringVarP(
		&opts.listenAddress,
		commands.HTTPListenAddressFlag,
		commands.HTTPListenAddressShortFlag,
		commands.HTTPListenAddressDefaultValue,
		commands.HTTPListenAddressUsage,
	)

	flags.DurationVarP(
		&opts.readTimeout,
		commands.HTTPReadTimeoutFlag,
		commands.HTTPReadTimeoutShortFlag,
		commands.HTTPReadTimeoutDefaultValue,
		commands.HTTPReadTimeoutUsage,
	)

	flags.DurationVarP(
		&opts.writeTimeout,
		commands.HTTPWriteTimeoutFlag,
		commands.HTTPWriteTimeoutShortFlag,
		commands.HTTPWriteTimeoutDefaultValue,
		commands.HTTPWriteTimeoutUsage,
	)

	flags.DurationVarP(
		&opts.idleTimeout,
		commands.HTTPIdleTimeoutFlag,
		commands.HTTPIdleTimeoutShortFlag,
		commands.HTTPIdleTimeoutDefaultValue,
		commands.HTTPIdleTimeoutUsage,
	)
}

// configureOptions configures serve options using config and flags.
func configureOptions(opts *serveOptions, httpConfig models.HTTPConfig) {
	if opts.listenAddress == "" {
		opts.listenAddress = httpConfig.ListenAddress
	}

	if opts.readTimeout == 0 {
		opts.readTimeout = httpConfig.ReadTimeout
	}

	if opts.writeTimeout == 0 {
		opts.writeTimeout = httpConfig.WriteTimeout
	}

	if opts.idleTimeout == 0 {
		opts.idleTimeout = httpConfig.IdleTimeout
	}
}

// runServe executes an `serve` command.
func runServe(
	ctx context.Context,
	opts *serveOptions,
) error {
	const timeout = 5 * time.Second

	handler := echo.New()

	setupRoutes(handlerOptions{
		useCase: opts.useCase,
		openAI:  opts.openAI,
	}, handler)

	server := &http.Server{
		Handler:      handler,
		Addr:         opts.listenAddress,
		ReadTimeout:  opts.readTimeout,
		WriteTimeout: opts.writeTimeout,
		IdleTimeout:  opts.idleTimeout,
	}

	go func() {
		<-ctx.Done()

		shutdownCtx, cancel := context.WithTimeout(context.Background(), timeout)
		defer cancel()

		_ = server.Shutdown(shutdownCtx)
	}()

	slog.Info("running HTTP server with params:",
		slog.Any("listen-address", opts.listenAddress),
		slog.Any("read-timeout", opts.readTimeout),
		slog.Any("write-timeout", opts.writeTimeout),
		slog.Any("idle-timeout", opts.idleTimeout),
	)

	err := server.ListenAndServe()
	if err != nil && !errors.Is(err, http.ErrServerClosed) {
		return errors.New(err.Error())
	}

	return nil
}
