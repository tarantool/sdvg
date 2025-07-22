package general

import (
	"context"
	"encoding/json"
	"log/slog"
	"os"
	"path/filepath"
	"slices"

	"github.com/pkg/errors"
	"github.com/tarantool/sdvg/internal/generator/models"
	"github.com/tarantool/sdvg/internal/generator/output"
)

// Verify interface compliance in compile time.
var _ output.Output = (*Output)(nil)

// Output type is implementation of output.
type Output struct {
	config             *models.OutputConfig
	models             map[string]*models.Model
	continueGeneration bool
	forceGeneration    bool
	writersByModelName map[string]*ModelWriter
}

// NewOutput function creates Output object.
func NewOutput(cfg *models.GenerationConfig, continueGeneration, forceGeneration bool) output.Output {
	filteredModels := make(map[string]*models.Model)

	for modelName, model := range cfg.Models {
		if !slices.Contains(cfg.ModelsToIgnore, modelName) {
			filteredModels[modelName] = model
		}
	}

	return &Output{
		config:             cfg.OutputConfig,
		models:             filteredModels,
		continueGeneration: continueGeneration,
		forceGeneration:    forceGeneration,
		writersByModelName: make(map[string]*ModelWriter),
	}
}

// Setup function creates model writers.
func (o *Output) Setup() error {
	if !o.continueGeneration && slices.Contains(models.DiskFilesOutputTypes, o.config.Type) {
		err := o.checkOutputConflicts(o.models, o.forceGeneration)
		if err != nil {
			return err
		}
	}

	writersByModelName := make(map[string]*ModelWriter)

	for modelName, model := range o.models {
		modelWriter, err := newModelWriter(model, o.config, o.continueGeneration)
		if err != nil {
			return err
		}

		writersByModelName[modelName] = modelWriter
	}

	o.writersByModelName = writersByModelName

	return nil
}

// HandleRowsBatch function get batch of rows from use case and send it to model writer.
func (o *Output) HandleRowsBatch(ctx context.Context, modelName string, rows []*models.DataRow) error {
	modelWriter, ok := o.writersByModelName[modelName]
	if !ok {
		return errors.Errorf("model writer for the model %q not found", modelName)
	}

	if err := modelWriter.WriteRows(ctx, rows); err != nil {
		return err
	}

	slog.Debug(
		"successfully wrote to file",
		slog.Any("model", modelName),
		slog.Any("number of lines", len(rows)),
	)

	return nil
}

func (o *Output) GetSavedRowsCountByModel() map[string]uint64 {
	savedRowsCountByModel := make(map[string]uint64, len(o.writersByModelName))

	for modelName, writer := range o.writersByModelName {
		savedRowsCountByModel[modelName] = writer.getSavedRows()
	}

	return savedRowsCountByModel
}

func (o *Output) SaveBackup(backup map[string]any) error {
	backupFilePath := o.getBackupFilePath()

	err := os.MkdirAll(filepath.Dir(backupFilePath), os.ModePerm)
	if err != nil {
		return errors.New(err.Error())
	}

	jsonData, err := json.MarshalIndent(backup, "", "  ")
	if err != nil {
		return errors.New(err.Error())
	}

	err = os.WriteFile(backupFilePath, jsonData, os.ModePerm)
	if err != nil {
		return errors.New(err.Error())
	}

	return nil
}

func (o *Output) ParseBackup() (*models.GenerationConfig, error) {
	backupFilePath := o.getBackupFilePath()

	data, err := os.ReadFile(backupFilePath)
	if err != nil {
		return nil, errors.New(err.Error())
	}

	var (
		check  map[string]any
		backup models.GenerationConfig
	)

	err = json.Unmarshal(data, &check)
	if err != nil {
		return nil, errors.New(err.Error())
	}

	if _, ok := check["random_seed"]; !ok {
		return nil, errors.New("random_seed not found in backup")
	}

	err = backup.ParseFromFile(backupFilePath)
	if err != nil {
		return nil, errors.New(err.Error())
	}

	return &backup, nil
}

func (o *Output) ParseCheckpoints() (map[string]*output.Checkpoint, error) {
	checkpoints := make(map[string]*output.Checkpoint)

	for modelName, writer := range o.writersByModelName {
		checkpoint, err := writer.ParseCheckpoint()
		if err != nil {
			return nil, err
		}

		checkpoints[modelName] = checkpoint
	}

	return checkpoints, nil
}

// Teardown function call the teardown method of each model writer.
func (o *Output) Teardown() error {
	for modelName, currentModelWriter := range o.writersByModelName {
		if err := currentModelWriter.Teardown(); err != nil {
			return err
		}

		slog.Debug("successfully tore down model writer", slog.Any("model", modelName))
	}

	return nil
}

func (o *Output) getBackupFilePath() string {
	return filepath.Join(o.config.Dir, output.BackupName)
}

// checkOutputConflicts finds possible conflicts and handles them.
// Removes all possible conflicts if forceGeneration flag is true.
// Otherwise, returns pretty error string.
func (o *Output) checkOutputConflicts(models map[string]*models.Model, forceGeneration bool) error {
	conflicts := make(map[string][]string) // key is cause, value is file names

	if err := checkBackupFile(conflicts, o.getBackupFilePath()); err != nil {
		return err
	}

	if !o.config.CreateModelDir {
		if err := checkPossiblePartitionDirs(conflicts, o.config.Dir); err != nil {
			return err
		}
	}

	for _, model := range models {
		modelConflicts, err := checkDirForModel(
			o.config.Dir,
			model.Name,
			o.config.Type,
			o.config.CreateModelDir,
		)
		if err != nil {
			return err
		}

		for cause, fileNames := range modelConflicts {
			conflicts[cause] = append(conflicts[cause], fileNames...)
		}
	}

	if err := handleConflicts(conflicts, forceGeneration); err != nil {
		return err
	}

	return nil
}
