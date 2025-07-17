package general

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/pkg/errors"

	"sdvg/internal/generator/common"
	"sdvg/internal/generator/models"
	"sdvg/internal/generator/output"
	"sdvg/internal/generator/output/general/writer"
	"sdvg/internal/generator/output/general/writer/csv"
	"sdvg/internal/generator/output/general/writer/devnull"
	"sdvg/internal/generator/output/general/writer/http"
	"sdvg/internal/generator/output/general/writer/parquet"
	"sdvg/internal/generator/output/general/writer/tcs"
)

const buffer = 100

// ModelWriter type implements the general logic of writing data.
type ModelWriter struct {
	model              *models.Model
	config             *models.OutputConfig
	basePath           string
	continueGeneration bool

	numberColumnsToDiscard  int
	partitionColumnsIndexes []int
	orderedColumnNames      []string

	checkpointFilePath   string
	checkpointTicker     *time.Ticker
	checkpointErrorsChan chan error

	writerByPartition map[string]writer.Writer
	writersMutex      *sync.RWMutex

	writtenRows     *atomic.Uint64
	writtenRowsWg   *sync.WaitGroup
	writtenRowsChan chan uint64
	stopChan        chan struct{}
}

// NewModelWriter creates ModelWriter object.
func newModelWriter(
	model *models.Model,
	config *models.OutputConfig,
	continueGeneration bool,
) (*ModelWriter, error) {
	orderedColumnNames := make([]string, 0, len(model.Columns))
	for _, column := range model.Columns {
		orderedColumnNames = append(orderedColumnNames, column.Name)
	}

	numberColumnsToDiscard := 0
	partitionOrderedColumnNames := make([]string, 0, len(model.PartitionColumns))

	for _, column := range model.PartitionColumns {
		if !column.WriteToOutput {
			numberColumnsToDiscard++
		}

		partitionOrderedColumnNames = append(partitionOrderedColumnNames, column.Name)
	}

	partitionColumnsIndexes := common.GetIndicesInOrder(
		orderedColumnNames,
		partitionOrderedColumnNames,
	)

	if slices.Contains(partitionColumnsIndexes, -1) {
		return nil, errors.Errorf("failed to create indexes for partition columns '%v'", partitionColumnsIndexes)
	}

	basePath := config.Dir
	if config.CreateModelDir {
		basePath = filepath.Join(basePath, model.ModelDir)
	}

	if err := os.MkdirAll(basePath, os.ModePerm); err != nil {
		return nil, errors.New(err.Error())
	}

	ticker := time.NewTicker(config.CheckpointInterval)

	modelWriter := &ModelWriter{
		model:                   model,
		config:                  config,
		basePath:                basePath,
		continueGeneration:      continueGeneration,
		numberColumnsToDiscard:  numberColumnsToDiscard,
		partitionColumnsIndexes: partitionColumnsIndexes,
		orderedColumnNames:      orderedColumnNames,
		checkpointTicker:        ticker,
		checkpointErrorsChan:    make(chan error, 1),
		writerByPartition:       make(map[string]writer.Writer),
		writersMutex:            &sync.RWMutex{},
		writtenRows:             &atomic.Uint64{},
		writtenRowsWg:           &sync.WaitGroup{},
		writtenRowsChan:         make(chan uint64, buffer),
		stopChan:                make(chan struct{}),
	}

	modelWriter.checkpointFilePath = modelWriter.getCheckpointFilePath()

	go modelWriter.updater()

	return modelWriter, nil
}

func (w *ModelWriter) updater() {
	w.writtenRowsWg.Add(1)
	go w.updateWrittenRows()

	for {
		select {
		case <-w.stopChan:
			return
		case <-w.checkpointTicker.C:
			if err := w.updateCheckpoint(); err != nil {
				w.checkpointErrorsChan <- errors.WithMessagef(
					err, "failed to update checkpoint file for model %q", w.model.Name,
				)

				return
			}
		}
	}
}

func (w *ModelWriter) updateWrittenRows() {
	defer w.writtenRowsWg.Done()

	for rows := range w.writtenRowsChan {
		w.writtenRows.Add(rows)
	}
}

func (w *ModelWriter) updateCheckpoint() error {
	checkpoint := output.Checkpoint{
		SavedRows: w.writtenRows.Load(),
	}

	jsonData, err := json.Marshal(checkpoint)
	if err != nil {
		return errors.New(err.Error())
	}

	err = os.WriteFile(w.checkpointFilePath, jsonData, os.ModePerm)
	if err != nil {
		return errors.New(err.Error())
	}

	return nil
}

// WriteRows function determines the partitioning key and sends the data to the appropriate writer.
func (w *ModelWriter) WriteRows(ctx context.Context, rows []*models.DataRow) error {
	for _, row := range rows {
		partitionPath := w.getPartitionPath(row)

		w.writersMutex.RLock()
		dataWriter, ok := w.writerByPartition[partitionPath]
		w.writersMutex.RUnlock()

		if !ok {
			newDataWriter, err := w.newWriter(ctx, partitionPath)
			if err != nil {
				return err
			}

			err = newDataWriter.Init()
			if err != nil {
				return err
			}

			w.writersMutex.Lock()
			w.writerByPartition[partitionPath] = newDataWriter
			w.writersMutex.Unlock()

			dataWriter = newDataWriter
		}

		// discard not writeable columns
		sendRow := &models.DataRow{
			Values: row.Values[:len(row.Values)-w.numberColumnsToDiscard],
		}

		if err := dataWriter.WriteRow(sendRow); err != nil {
			return err
		}
	}

	select {
	case err := <-w.checkpointErrorsChan:
		return err
	default:
		return nil
	}
}

// getPartitionPath function returns the partitioning key based on the data in the row.
func (w *ModelWriter) getPartitionPath(row *models.DataRow) string {
	if len(w.model.PartitionColumns) == 0 {
		return w.basePath
	}

	var sb strings.Builder

	sb.WriteString(w.basePath)

	for _, columnIdx := range w.partitionColumnsIndexes {
		columnName := w.orderedColumnNames[columnIdx]
		value := row.Values[columnIdx]

		if value == nil {
			sb.WriteString(fmt.Sprintf("/%s=%s", columnName, "null"))
		} else {
			sb.WriteString(fmt.Sprintf("/%s=%v", columnName, value))
		}
	}

	return sb.String()
}

// newWriter function creates writer.Writer object based on output type from models.OutputConfig.
func (w *ModelWriter) newWriter(ctx context.Context, outPath string) (writer.Writer, error) {
	var dataWriter writer.Writer

	switch w.config.Type {
	case "csv":
		dataWriter = csv.NewWriter(
			ctx,
			w.model,
			w.config.CSVParams,
			outPath,
			w.continueGeneration,
			w.writtenRowsChan,
		)
	case "devnull":
		dataWriter = devnull.NewWriter(
			w.model,
			w.config.DevNullParams,
		)
	case "http":
		dataWriter = http.NewWriter(
			ctx,
			w.model,
			w.config.HTTPParams,
			w.writtenRowsChan,
		)
	case "tcs":
		dataWriter = tcs.NewWriter(
			ctx,
			w.model,
			w.config.TCSParams,
			w.writtenRowsChan,
		)
	case "parquet":
		dataWriter = parquet.NewWriter(
			w.model,
			w.config.ParquetParams,
			parquet.NewFileSystem(),
			outPath,
			w.continueGeneration,
			w.writtenRowsChan,
		)
	default:
		return nil, errors.Errorf("unknown output type: %q", w.config.Type)
	}

	return dataWriter, nil
}

func (w *ModelWriter) ParseCheckpoint() (*output.Checkpoint, error) {
	checkpointFilePath := w.getCheckpointFilePath()

	var checkpoint output.Checkpoint

	if err := models.DecodeFile(checkpointFilePath, &checkpoint); err != nil {
		return nil, errors.WithMessagef(err, "failed to parse checkpoint file for model %q", w.model.Name)
	}

	w.writtenRows.Store(checkpoint.SavedRows)

	return &checkpoint, nil
}

func (w *ModelWriter) Teardown() error {
	var errorsChan = make(chan error, len(w.writerByPartition))

	wg := &sync.WaitGroup{}

	w.writersMutex.RLock()
	for _, partitionWriter := range w.writerByPartition {
		wg.Add(1)

		go func(partitionWriter writer.Writer, errChan chan error) {
			defer wg.Done()

			if err := partitionWriter.Teardown(); err != nil {
				errChan <- err
			}
		}(partitionWriter, errorsChan)
	}
	w.writersMutex.RUnlock()

	wg.Wait()

	w.checkpointTicker.Stop()
	w.stopChan <- struct{}{}

	close(errorsChan)
	close(w.writtenRowsChan)
	close(w.stopChan)
	close(w.checkpointErrorsChan)

	w.writtenRowsWg.Wait()

	if err := w.updateCheckpoint(); err != nil {
		return errors.WithMessagef(err, "failed to update checkpoint file for model %q", w.model.Name)
	}

	var writersErrors = make([]string, 0, len(w.writerByPartition))
	for err := range errorsChan {
		writersErrors = append(writersErrors, err.Error())
	}

	if len(writersErrors) > 0 {
		return errors.New(strings.Join(writersErrors, ": "))
	}

	select {
	case err := <-w.checkpointErrorsChan:
		return err
	default:
		return nil
	}
}

func (w *ModelWriter) getSavedRows() uint64 {
	return w.writtenRows.Load()
}

func (w *ModelWriter) getCheckpointFilePath() string {
	return filepath.Join(w.basePath, w.model.Name+output.CheckpointSuffix)
}
