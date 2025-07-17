package tcs

import (
	"context"

	"sdvg/internal/generator/models"
	"sdvg/internal/generator/output/general/writer/http"
)

// Writer type is implementation of writer to TCS destination storage.
type Writer struct {
	*http.Writer
}

// NewWriter function creates Writer object.
func NewWriter(
	ctx context.Context,
	model *models.Model,
	config *models.TCSConfig,
	writtenRowsChan chan<- uint64,
) *Writer {
	return &Writer{
		Writer: http.NewWriter(ctx, model, &config.HTTPParams, writtenRowsChan),
	}
}
