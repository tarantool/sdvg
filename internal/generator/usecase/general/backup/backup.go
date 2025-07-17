package backup

import (
	"log/slog"

	"github.com/pkg/errors"

	"sdvg/internal/generator/models"
	"sdvg/internal/generator/output"
)

func ProcessContinueGeneration(cfg *models.GenerationConfig, out output.Output) error {
	slog.Debug("start processing continue generation")

	slog.Debug("read backup")

	if err := handleBackup(cfg, out); err != nil {
		return err
	}

	slog.Debug("read checkpoints")

	if err := handleCheckpoint(cfg, out); err != nil {
		return err
	}

	return nil
}

func SaveBackup(cfg *models.GenerationConfig, out output.Output) error {
	slog.Debug("saving backup")

	backup := extractBackupFields(cfg)

	err := out.SaveBackup(backup)
	if err != nil {
		return err
	}

	return nil
}

func handleBackup(cfg *models.GenerationConfig, out output.Output) error {
	backup, err := out.ParseBackup()
	if err != nil {
		return errors.WithMessage(err, "failed to parse backup file")
	}

	if cfg.RealRandomSeed == 0 {
		cfg.RandomSeed = backup.RandomSeed
		slog.Debug("set random seed", slog.Uint64("seed", backup.RandomSeed))
	}

	ok, diffs := compareBackupField(cfg, backup)
	if !ok {
		return errors.New(formatDiff(diffs))
	}

	slog.Info("backup successfully read")

	return nil
}

func handleCheckpoint(cfg *models.GenerationConfig, out output.Output) error {
	checkpoints, err := out.ParseCheckpoints()
	if err != nil {
		return errors.WithMessage(err, "failed to parse checkpoint file")
	}

	for modelName, checkpoint := range checkpoints {
		model, ok := cfg.Models[modelName]
		if !ok {
			return errors.Errorf("model %q not found", modelName)
		}

		generateFrom := checkpoint.SavedRows
		if generateFrom >= model.GenerateTo {
			generateFrom = model.GenerateTo
		}

		model.GenerateFrom = generateFrom

		slog.Debug("set generate_from", slog.String("model", modelName), slog.Uint64("value", generateFrom))
	}

	return nil
}
