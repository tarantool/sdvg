package devnull

import (
	"github.com/tarantool/sdvg/internal/generator/models"
	"github.com/tarantool/sdvg/internal/generator/output/general/writer"
)

// Verify interface compliance in compile time.
var _ writer.Writer = (*Writer)(nil)

// Writer type is implementation of null writer.
type Writer struct {
	model   *models.Model
	handler func(row *models.DataRow, modelName string) error
}

// NewWriter function creates Writer object.
func NewWriter(model *models.Model, config *models.DevNullConfig) *Writer {
	return &Writer{
		model:   model,
		handler: config.Handler,
	}
}

// Init does nothing.
func (w *Writer) Init() error {
	return nil
}

// WriteRow does nothing.
func (w *Writer) WriteRow(row *models.DataRow) error {
	if w.handler != nil {
		return w.handler(row, w.model.Name)
	}

	return nil
}

// Teardown does nothing.
func (w *Writer) Teardown() error {
	return nil
}
