package app

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"runtime/pprof"
	"strings"
	"syscall"

	"github.com/pkg/errors"

	"sdvg/internal/generator/cli"
	"sdvg/internal/generator/cli/options"
	"sdvg/internal/generator/usecase"
	"sdvg/internal/generator/usecase/general"
)

type App struct {
	cliOpts       *options.CliOptions
	cli           *cli.Cli
	useCase       usecase.UseCase
	cpuProfile    *os.File
	memoryProfile *os.File
}

func NewApp(version string) *App {
	useCase := general.NewUseCase(general.UseCaseConfig{})
	cliOpts := options.NewCliOptions(useCase, version)
	sdvgCli := cli.NewCli(cliOpts)
	sdvgCli.MustSetup()

	return &App{
		useCase: useCase,
		cliOpts: cliOpts,
		cli:     sdvgCli,
	}
}

func (a *App) Run() {
	ctx, cancelCtx := a.notifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM, syscall.SIGQUIT)

	a.run(ctx, cancelCtx)

	//nolint:errorlint
	switch err := context.Cause(ctx); err.(type) {
	case nil:
	case *SignalError:
		slog.Warn("sdvg finished due to event", slog.String("event", err.Error()))
	default:
		slog.Error("sdvg finished due error", slog.String("error", err.Error()))

		if a.cliOpts.DebugMode() {
			a.logStackTrace(err)
		}

		os.Exit(1)
	}
}

func (a *App) notifyContext(ctx context.Context, signals ...os.Signal) (context.Context, context.CancelCauseFunc) {
	osSignalChannel := make(chan os.Signal, 1)
	signal.Notify(osSignalChannel, signals...)

	ctxCause, cancelCtx := context.WithCancelCause(ctx)

	go func() {
		osSignal := <-osSignalChannel
		slog.Info("got os signal, canceling", slog.String("signal", osSignal.String()))
		cancelCtx(NewSignalError(osSignal))

		osSignal = <-osSignalChannel
		slog.Error("got os signal, force exit", slog.String("signal", osSignal.String()))
		os.Exit(1)
	}()

	return ctxCause, cancelCtx
}

func (a *App) run(ctx context.Context, cancelCtx context.CancelCauseFunc) {
	a.startProfiling()
	defer a.stopProfiling()

	if err := a.useCase.Setup(); err != nil {
		cancelCtx(err)

		return
	}

	if err := a.cli.Run(ctx); err != nil {
		cancelCtx(err)

		return
	}

	if err := a.useCase.Teardown(); err != nil {
		cancelCtx(err)

		return
	}
}

func (a *App) startProfiling() {
	var err error

	if a.cliOpts.CPUProfile() != "" {
		if a.cpuProfile, err = os.Create(a.cliOpts.CPUProfile()); err != nil {
			slog.Error("failed to create CPU profile file", slog.String("error", err.Error()))
		} else {
			if err = pprof.StartCPUProfile(a.cpuProfile); err != nil {
				slog.Error("failed to start CPU profiling", slog.String("error", err.Error()))
			}
		}
	}
}

func (a *App) stopProfiling() {
	var err error

	if a.cliOpts.CPUProfile() != "" {
		pprof.StopCPUProfile()
	}

	if a.cliOpts.MemoryProfile() != "" {
		if a.memoryProfile, err = os.Create(a.cliOpts.MemoryProfile()); err != nil {
			slog.Error("failed to create memory profile file", slog.String("error", err.Error()))
		} else {
			if err = pprof.WriteHeapProfile(a.memoryProfile); err != nil {
				slog.Error("failed to write memory profiling results", slog.String("error", err.Error()))
			}
		}
	}

	if a.cpuProfile != nil {
		if err = a.cpuProfile.Close(); err != nil {
			slog.Error("failed to close CPU profile file", slog.String("error", err.Error()))
		}
	}

	if a.memoryProfile != nil {
		if err = a.memoryProfile.Close(); err != nil {
			slog.Error("failed to close memory profile file", slog.String("error", err.Error()))
		}
	}
}

func (a *App) logStackTrace(err error) {
	if e, ok := errors.Cause(err).(stackTracer); ok {
		for _, frame := range e.StackTrace() {
			frameTrace := strings.Split(fmt.Sprintf("%+v", frame), "\n")
			slog.Error(frameTrace[0])
			slog.Error(frameTrace[1])
		}
	}
}
