package generate

import (
	"context"
	"fmt"
	"log/slog"
	"sync"
	"sync/atomic"
	"time"

	"github.com/pkg/errors"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
	"github.com/tarantool/sdvg/internal/generator/cli/commands"
	"github.com/tarantool/sdvg/internal/generator/cli/confirm"
	"github.com/tarantool/sdvg/internal/generator/cli/options"
	"github.com/tarantool/sdvg/internal/generator/cli/progress"
	"github.com/tarantool/sdvg/internal/generator/cli/progress/bar"
	"github.com/tarantool/sdvg/internal/generator/cli/progress/log"
	"github.com/tarantool/sdvg/internal/generator/cli/render"
	"github.com/tarantool/sdvg/internal/generator/cli/utils"
	"github.com/tarantool/sdvg/internal/generator/models"
	"github.com/tarantool/sdvg/internal/generator/output/general"
	"github.com/tarantool/sdvg/internal/generator/usecase"
)

// generateOptions type is used to describe 'generate' command options.
type generateOptions struct {
	useCase              usecase.UseCase
	renderer             render.Renderer
	continueGeneration   bool
	generationConfigPath string
	useTTY               bool
	forceGeneration      bool
}

// NewGenerateCommand creates 'generate' command for CLI.
func NewGenerateCommand(cliOpts *options.CliOptions) *cobra.Command {
	opts := &generateOptions{}

	cmd := &cobra.Command{
		Use:                   "generate [FLAGS] [PATH]",
		Short:                 "Generates data based on provided models config",
		Args:                  commands.RequiresMaxArgs(1),
		DisableFlagsInUseLine: true,
		PreRun: func(_ *cobra.Command, _ []string) {
			opts.useCase = cliOpts.UseCase()
			opts.renderer = cliOpts.Renderer()
			opts.useTTY = cliOpts.UseTTY()
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := cmd.Context()

			err := getGenerationConfigFilePath(ctx, opts, args)
			if err != nil {
				return errors.WithMessagef(err, "failed to get generation config file path")
			}

			slog.Info("generator started", slog.String("version", cliOpts.Version()))

			err = runGenerate(ctx, opts)
			if err != nil {
				return errors.WithMessagef(err, "failed to generate")
			}

			slog.Info("generator finished")

			return nil
		},
	}

	cmd.SetOut(cliOpts.Out())

	setupFlags(cmd.Flags(), opts)

	return cmd
}

func setupFlags(flags *pflag.FlagSet, opts *generateOptions) {
	flags.BoolVarP(
		&opts.continueGeneration,
		commands.ContinueGenerationFlag,
		commands.ContinueGenerationShortFlag,
		commands.ContinueGenerationDefaultValue,
		commands.ContinueGenerationUsage,
	)

	flags.BoolVarP(
		&opts.forceGeneration,
		commands.ForceGenerationFlag,
		commands.ForceGenerationShortFlag,
		commands.ForceGenerationFlagDefaultValue,
		commands.ForceGenerationUsage,
	)
}

// getGenerationConfigFilePath gets generation config file path from arguments or user input.
func getGenerationConfigFilePath(ctx context.Context, opts *generateOptions, args []string) error {
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

// runGenerate executes an `generate` command.
func runGenerate(ctx context.Context, opts *generateOptions) error {
	generationCfg := &models.GenerationConfig{}

	err := generationCfg.ParseFromFile(opts.generationConfigPath)
	if err != nil {
		return err
	}

	progressTrackerManager, confirm := initProgressTrackerManager(ctx, opts.renderer, opts.useTTY, opts.forceGeneration)

	out := general.NewOutput(generationCfg, opts.continueGeneration, opts.forceGeneration, confirm)

	taskID, err := opts.useCase.CreateTask(
		ctx, usecase.TaskConfig{
			GenerationConfig:   generationCfg,
			Output:             out,
			ContinueGeneration: opts.continueGeneration,
		},
	)
	if err != nil {
		return err
	}

	var (
		finished atomic.Bool
		wg       sync.WaitGroup
	)

	startProgressTracking(
		progressTrackerManager,
		opts.useCase,
		taskID,
		&finished,
		&wg,
	)

	err = opts.useCase.WaitResult(taskID)

	finished.Store(true)

	if err == nil {
		wg.Wait()
	}

	if err != nil {
		slog.Info("generation seed", slog.Uint64("seed", generationCfg.RandomSeed))

		savedRowsCountByModel := out.GetSavedRowsCountByModel()
		for modelName, count := range savedRowsCountByModel {
			slog.Info("saved rows", slog.String("model", modelName), slog.Uint64("count", count))
		}

		return err
	}

	return nil
}

// initProgressTrackerManager inits progress bar manager (progress.Tracker)
// and builds confirm.Confirm func based on useTTY and forceGeneration.
func initProgressTrackerManager(
	ctx context.Context,
	renderer render.Renderer,
	useTTY bool,
	forceGeneration bool,
) (progress.Tracker, confirm.Confirm) {
	var (
		progressTrackerManager progress.Tracker
		confirmFunc            confirm.Confirm
	)

	if useTTY {
		progressTrackerManager = bar.NewProgressBarManager(ctx)

		confirmFunc = confirm.BuildConfirmTTY(renderer, progressTrackerManager)
	} else {
		isUpdatePaused := &atomic.Bool{}

		progressTrackerManager = log.NewProgressLogManager(ctx, isUpdatePaused)

		confirmFunc = confirm.BuildConfirmNoTTY(renderer, progressTrackerManager, isUpdatePaused)
	}

	if forceGeneration {
		confirmFunc = func(_ context.Context, _ string) (bool, error) {
			return true, nil
		}
	}

	return progressTrackerManager, confirmFunc
}

// startProgressTracking runs function to track progress of task
// by getting progress from usecase object and displaying it.
func startProgressTracking(
	progressTrackerManager progress.Tracker,
	uc usecase.UseCase,
	taskID string,
	finished *atomic.Bool,
	wg *sync.WaitGroup,
) {
	const delay = 500 * time.Millisecond

	wg.Add(1)

	go func() {
		defer wg.Done()

		lastUpdate := false

		for {
			progresses, err := uc.GetProgress(taskID)
			if err != nil {
				slog.Error("error getting progress", slog.Any("taskID", taskID))
			}

			for k, p := range progresses {
				progressTrackerManager.AddTask(
					k,
					fmt.Sprintf("generating data for model %q", k),
					p.Total,
				)
				progressTrackerManager.UpdateProgress(k, p)
			}

			if lastUpdate {
				break
			}

			if finished.Load() {
				lastUpdate = true
			} else {
				time.Sleep(delay)
			}
		}

		progressTrackerManager.Wait()
	}()
}
