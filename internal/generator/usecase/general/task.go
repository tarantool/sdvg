package general

import (
	"context"
	"log/slog"
	"path/filepath"
	"slices"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/pkg/errors"
	"github.com/tarantool/sdvg/internal/generator/common"
	"github.com/tarantool/sdvg/internal/generator/models"
	"github.com/tarantool/sdvg/internal/generator/output"
	"github.com/tarantool/sdvg/internal/generator/usecase"
	"github.com/tarantool/sdvg/internal/generator/usecase/general/backup"
	"github.com/tarantool/sdvg/internal/generator/usecase/general/generator"
	"github.com/tarantool/sdvg/internal/generator/usecase/general/progress"
)

const TTL = 5 * time.Minute

// Task type is implementation of one task from usecase.
type Task struct {
	config             *models.GenerationConfig
	continueGeneration bool
	ID                 string
	output             output.Output
	generators         map[string]*generator.ColumnGenerator
	progress           *progress.Handler
	runMutex           *sync.Mutex
	statusMutex        *sync.RWMutex
	finished           bool
	error              error
}

// NewTask function creates context for one generation job.
func NewTask(cfg usecase.TaskConfig) (*Task, error) {
	taskID := uuid.NewString()

	if cfg.HTTPDelivery {
		outputDir := cfg.GenerationConfig.OutputConfig.Dir
		cfg.GenerationConfig.OutputConfig.Dir = filepath.Join(outputDir, taskID)
	}

	if err := cfg.Output.Setup(); err != nil {
		return nil, errors.WithMessage(err, "failed to setup output")
	}

	if cfg.ContinueGeneration {
		err := backup.ProcessContinueGeneration(cfg.GenerationConfig, cfg.Output)
		if err != nil {
			return nil, errors.WithMessage(err, "failed to continue generation")
		}
	} else {
		err := backup.SaveBackup(cfg.GenerationConfig, cfg.Output)
		if err != nil {
			return nil, errors.WithMessage(err, "failed to save backup")
		}
	}

	generators, err := newGenerators(cfg.GenerationConfig)
	if err != nil {
		return nil, err
	}

	return &Task{
		config:             cfg.GenerationConfig,
		continueGeneration: cfg.ContinueGeneration,
		ID:                 taskID,
		output:             cfg.Output,
		generators:         generators,
		progress:           progress.NewHandler(),
		runMutex:           &sync.Mutex{},
		statusMutex:        &sync.RWMutex{},
		finished:           false,
		error:              nil,
	}, nil
}

func newGenerators(cfg *models.GenerationConfig) (map[string]*generator.ColumnGenerator, error) {
	generators := make(map[string]*generator.ColumnGenerator)

	for modelName, model := range cfg.Models {
		for _, column := range model.Columns {
			dataModelName := modelName
			dataModel := model
			dataColumn := column

			if column.ForeignKeyColumn != nil {
				dataModelName = strings.Split(column.ForeignKey, ".")[0]
				dataModel = cfg.Models[dataModelName]
				dataColumn = column.ForeignKeyColumn
			}

			columnKey := common.GetKey(modelName, column.Name)

			gen, err := generator.NewColumnGenerator(
				cfg.RandomSeed,
				modelName, model, column,
				dataModelName, dataModel, dataColumn,
			)
			if err != nil {
				return nil, err
			}

			generators[columnKey] = gen
		}
	}

	return generators, nil
}

// RunTask function generates unique values and then all values for selected model.
func (t *Task) RunTask(ctx context.Context, callback func()) {
	started := make(chan struct{})

	go func() {
		t.runMutex.Lock()
		defer t.runMutex.Unlock()

		t.statusMutex.Lock()
		t.finished = false
		t.error = nil
		t.statusMutex.Unlock()

		started <- struct{}{}

		err := t.runTask(ctx)

		t.statusMutex.Lock()
		t.finished = true
		t.error = err
		t.statusMutex.Unlock()

		time.AfterFunc(TTL, callback)
	}()

	<-started
}

func (t *Task) runTask(ctx context.Context) error {
	t.skipRows()

	err := t.generateAndSaveValues(ctx)
	if err != nil {
		return errors.WithMessage(err, "failed to generate and save values")
	}

	return err
}

func (t *Task) GetProgress() map[string]usecase.Progress {
	return t.progress.GetAll()
}

func (t *Task) GetError() (bool, error) {
	t.statusMutex.RLock()
	defer t.statusMutex.RUnlock()

	return t.finished, t.error
}

func (t *Task) WaitError() error {
	t.runMutex.Lock()
	defer t.runMutex.Unlock()

	return t.error
}

// generateAndSaveValues function generates values for all model.
func (t *Task) generateAndSaveValues(ctx context.Context) error {
	var err error

	ctx, cancelCtx := context.WithCancelCause(ctx)
	defer cancelCtx(err)

	defer func() {
		tErr := t.output.Teardown()
		if tErr != nil {
			if err != nil {
				slog.Error("failed to teardown output", slog.Any("error", tErr))
			} else {
				err = errors.WithMessage(tErr, "failed to teardown output")
			}
		}
	}()

	pool := common.NewWorkerPool(t.generateAndSaveBatch, 0, t.config.WorkersCount)
	pool.Start()
	defer pool.Stop()

	// Found models to generate

	for modelName, model := range t.config.Models {
		if slices.Index(t.config.ModelsToIgnore, modelName) >= 0 {
			slog.Debug("skip generating values", "model", modelName)

			continue
		}

		pool.Add(1)

		go func() {
			defer pool.Done()

			slog.Debug("start generating values", "model", modelName)
			t.progress.Create(modelName, model.GenerateTo-model.GenerateFrom)

			outputSyncer := common.NewSyncer()

			for i := model.GenerateFrom; i < model.GenerateTo; i += t.config.BatchSize {
				rowsCount := min(t.config.BatchSize, model.GenerateTo-i)

				generators := make([]*generator.BatchGenerator, 0, len(model.Columns))

				for _, column := range model.Columns {
					columnKey := common.GetKey(modelName, column.Name)
					generators = append(generators, t.generators[columnKey].NewBatchGenerator(rowsCount))
				}

				pool.Submit(ctx, outputSyncer.WorkerSyncer(), model, generators, rowsCount)
			}
		}()
	}

	if err = pool.WaitOrError(); err != nil {
		err = errors.WithMessage(err, "failed to generate model values")
		cancelCtx(err)

		return err
	}

	if common.CtxClosed(ctx) {
		err = &common.ContextCancelError{}
		cancelCtx(err)

		return err
	}

	slog.Debug("generating values for all models finished")

	return nil
}

func (t *Task) skipRows() {
	for key, gen := range t.generators {
		modelName := strings.Split(key, ".")[0]
		gen.SkipRows(t.config.Models[modelName].GenerateFrom)
	}
}

// generateAndSaveBatch function generate batch of values for selected column and send it to output.
func (t *Task) generateAndSaveBatch(
	ctx context.Context, outputSync *common.WorkerSyncer,
	model *models.Model, generators []*generator.BatchGenerator, count uint64,
) error {
	defer outputSync.Done(ctx)

	batch := make([]*models.DataRow, count)
	for i := range count {
		batch[i] = &models.DataRow{
			Values: make([]any, len(generators)),
		}
	}

	sortedColumn, err := models.TopologicalSort(model.Columns)
	if err != nil {
		return err
	}

	originIndexes := make(map[string]int, len(model.Columns))
	for index, column := range model.Columns {
		originIndexes[column.Name] = index
	}

	for i := range count {
		generatedValues := make(map[string]any)

		for _, columnName := range sortedColumn {
			if common.CtxClosed(ctx) {
				return &common.ContextCancelError{}
			}

			value, err := generators[originIndexes[columnName]].Value(generatedValues)
			if err != nil {
				return errors.WithMessage(err, "failed to get or generate value")
			}

			generatedValues[columnName] = value
			batch[i].Values[originIndexes[columnName]] = value
		}
	}

	outputSync.WaitPrevious(ctx)

	err = t.output.HandleRowsBatch(ctx, model.Name, batch)
	if err != nil {
		return errors.WithMessage(err, "failed to save batch to output")
	}

	t.progress.Add(model.Name, count)

	return nil
}
