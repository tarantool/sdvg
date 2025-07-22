package writer

import (
	"github.com/tarantool/sdvg/internal/generator/models"
)

// Writer interface implementation should write data to destination storage in a specific format.
type Writer interface {
	// Init function should initialize writer.
	Init() error
	// WriteRow function should write row to destination storage.
	WriteRow(row *models.DataRow) error
	// Teardown function should wait recording finish.
	Teardown() error
}
