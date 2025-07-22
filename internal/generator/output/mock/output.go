package mock

import (
	"context"

	"github.com/tarantool/sdvg/internal/generator/models"
	"github.com/tarantool/sdvg/internal/generator/output"
)

// Verify interface compliance in compile time.
var _ output.Output = (*Output)(nil)

// Output type is implementation of output.
type Output struct {
	handler func(ctx context.Context, modelName string, rows []*models.DataRow) error
}

// NewOutput function creates Output object.
func NewOutput(handler func(ctx context.Context, modelName string, rows []*models.DataRow) error) output.Output {
	return &Output{handler: handler}
}

// Setup function do nothing.
func (o *Output) Setup() error {
	return nil
}

// HandleRowsBatch function get batch of rows from use case and send it to model writer.
func (o *Output) HandleRowsBatch(ctx context.Context, modelName string, rows []*models.DataRow) error {
	return o.handler(ctx, modelName, rows)
}

func (o *Output) GetSavedRowsCountByModel() map[string]uint64 {
	return nil
}

func (o *Output) SaveBackup(_ map[string]any) error {
	return nil
}

func (o *Output) ParseBackup() (*models.GenerationConfig, error) {
	return nil, nil //nolint:nilnil
}

func (o *Output) ParseCheckpoints() (map[string]*output.Checkpoint, error) {
	return nil, nil //nolint:nilnil
}

// Teardown function call the teardown method of each model writer.
func (o *Output) Teardown() error { return nil }
