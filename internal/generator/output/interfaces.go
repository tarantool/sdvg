package output

import (
	"context"

	"github.com/tarantool/sdvg/internal/generator/models"
)

// Output interface implementation should send all rows
// from use case to appropriate model writer.
type Output interface {
	// Setup function should configure some output parameters.
	Setup() error
	// HandleRowsBatch function should receive batch of rows from
	// use case and send it to appropriate model writer.
	HandleRowsBatch(ctx context.Context, modelName string, rows []*models.DataRow) error
	// GetSavedRowsCountByModel function should return number of saved rows for each models
	GetSavedRowsCountByModel() map[string]uint64
	// SaveBackup function should save backup of generation config
	SaveBackup(backup map[string]any) error
	// ParseBackup function should parse backup and return it
	ParseBackup() (*models.GenerationConfig, error)
	// ParseCheckpoints function should parse checkpoint for models and return it
	ParseCheckpoints() (map[string]*Checkpoint, error)
	// Teardown function should call the teardown method of each model writer.
	Teardown() error
}
