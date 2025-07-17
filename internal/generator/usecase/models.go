package usecase

import (
	"sdvg/internal/generator/models"
	"sdvg/internal/generator/output"
)

// TaskConfig type is used to describe config for task.
type TaskConfig struct {
	GenerationConfig   *models.GenerationConfig
	Output             output.Output
	ContinueGeneration bool
	HTTPDelivery       bool
}

// Progress type is used to represent progress of generation.
type Progress struct {
	Done  uint64
	Total uint64
}
