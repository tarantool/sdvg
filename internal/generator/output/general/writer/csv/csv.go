package csv

import (
	"context"
	stdCSV "encoding/csv"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"reflect"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/pkg/errors"
	"github.com/tarantool/sdvg/internal/generator/common"
	"github.com/tarantool/sdvg/internal/generator/models"
	"github.com/tarantool/sdvg/internal/generator/output/general/writer"
)

const (
	flushInterval = time.Second
)

// Verify interface compliance in compile time.
var _ writer.Writer = (*Writer)(nil)

// Writer type is implementation of Writer to CSV file.
type Writer struct {
	ctx context.Context //nolint:containedctx

	model              *models.Model
	columnsToDiscard   map[string]struct{}
	config             *models.CSVConfig
	outputPath         string
	continueGeneration bool

	fileDescriptor *os.File
	csvWriter      *stdCSV.Writer
	flushTicker    *time.Ticker
	flushWg        *sync.WaitGroup
	flushStopChan  chan struct{}

	totalWrittenRows uint64
	bufferedRows     uint64
	writtenRowsChan  chan<- uint64

	writerChan  chan *models.DataRow
	errorsChan  chan error
	writerWg    *sync.WaitGroup
	writerMutex *sync.Mutex
	started     bool
}

// NewWriter function creates Writer object.
func NewWriter(
	ctx context.Context,
	model *models.Model,
	config *models.CSVConfig,
	columnsToDiscard map[string]struct{},
	outputPath string,
	continueGeneration bool,
	writtenRowsChan chan<- uint64,
) *Writer {
	return &Writer{
		ctx:                ctx,
		model:              model,
		columnsToDiscard:   columnsToDiscard,
		config:             config,
		outputPath:         outputPath,
		continueGeneration: continueGeneration,
		flushTicker:        time.NewTicker(flushInterval),
		flushWg:            &sync.WaitGroup{},
		flushStopChan:      make(chan struct{}),
		writtenRowsChan:    writtenRowsChan,
		writerChan:         make(chan *models.DataRow),
		errorsChan:         make(chan error, 1),
		writerWg:           &sync.WaitGroup{},
		writerMutex:        &sync.Mutex{},
		started:            false,
	}
}

// Init function creates output file and starts flushing and receiving row from internal queue.
func (w *Writer) Init() error {
	if w.started {
		return errors.Errorf("writer for model %q with output path %q has already been initialized",
			w.model.Name, w.outputPath)
	}

	if err := os.MkdirAll(w.outputPath, os.ModePerm); err != nil {
		return errors.New(err.Error())
	}

	if w.continueGeneration {
		savedRows, err := w.getSavedRowsCount()
		if err != nil {
			return err
		}

		w.totalWrittenRows = savedRows
	}

	w.started = true
	w.writerWg.Add(1)
	w.flushWg.Add(1)

	go w.writer()
	go w.flusher()

	return nil
}

// writer function receives row from internal queue and processes it.
func (w *Writer) writer() {
	defer w.writerWg.Done()

	for {
		select {
		case <-w.ctx.Done():
			return
		case data, ok := <-w.writerChan:
			if !ok {
				return
			}

			parsedRow, err := w.parseDataRow(data)
			if err != nil {
				w.errorsChan <- err

				return
			}

			err = w.writeRow(parsedRow)
			if err != nil {
				w.errorsChan <- err

				return
			}
		}
	}
}

func (w *Writer) flusher() {
	defer w.flushWg.Done()

	for {
		select {
		case <-w.flushStopChan:
			return
		case <-w.flushTicker.C:
			if w.csvWriter != nil {
				err := w.flush()
				if err != nil {
					w.errorsChan <- err
				}
			}
		}
	}
}

func (w *Writer) getSavedRowsCount() (uint64, error) {
	fileNumber, err := w.getFileNumber()
	if err != nil {
		return 0, err
	}

	savedRowsCount := uint64(fileNumber) * w.model.RowsPerFile
	fileName := fmt.Sprintf("%s_%d.csv", w.model.Name, fileNumber)

	rowsInLastFile, err := w.getRowsInFile(fileName)
	if err != nil {
		return 0, errors.WithMessagef(err, "failed to count the number of written rows for file %q", fileName)
	}

	savedRowsCount += rowsInLastFile

	return savedRowsCount, nil
}

func (w *Writer) getFileNumber() (int, error) {
	fileNames, err := common.WalkWithFilter(w.outputPath, func(entry os.DirEntry) bool {
		return !entry.IsDir() && filepath.Ext(entry.Name()) == ".csv"
	})
	if err != nil {
		return 0, errors.WithMessagef(err, "failed to get number of file in directory %q", w.outputPath)
	}

	if len(fileNames) == 0 {
		return 0, nil
	}

	var fileNumber int

	re := regexp.MustCompile(fmt.Sprintf("^%s_(\\d+)\\.csv$", w.model.Name))

	for _, name := range fileNames {
		matches := re.FindStringSubmatch(name)

		if len(matches) > 1 {
			number, _ := strconv.Atoi(matches[1])
			fileNumber = max(fileNumber, number)
		}
	}

	return fileNumber, nil
}

func (w *Writer) getRowsInFile(fileName string) (uint64, error) {
	fullPath := filepath.Join(w.outputPath, fileName)

	if _, err := os.Stat(fullPath); os.IsNotExist(err) {
		return 0, nil
	}

	file, err := os.Open(fullPath)
	if err != nil {
		return 0, errors.New(err.Error())
	}
	defer file.Close()

	var rowsCount uint64

	reader := stdCSV.NewReader(file)
	reader.Comma = []rune(w.config.Delimiter)[0]

	for {
		row, err := reader.Read()
		_ = row

		if err != nil {
			if errors.Is(err, io.EOF) {
				break
			}

			return 0, errors.New(err.Error())
		}

		rowsCount++
	}

	if !w.config.WithoutHeaders {
		rowsCount--
	}

	return rowsCount, nil
}

// parseDataRow function parses raw data into string that can be written to CSV.
//
//nolint:cyclop
func (w *Writer) parseDataRow(row *models.DataRow) ([]string, error) {
	parsedRow := make([]string, 0, len(row.Values))

	for _, value := range row.Values {
		if value == nil {
			parsedRow = append(parsedRow, "")

			continue
		}

		v := reflect.ValueOf(value)
		switch v.Kind() {
		case reflect.String:
			parsedRow = append(parsedRow, v.String())
		case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64,
			reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
			parsedRow = append(parsedRow, strconv.FormatInt(v.Int(), 10))
		case reflect.Float32:
			parsedRow = append(parsedRow, strconv.FormatFloat(v.Float(), 'f', w.config.FloatPrecision, 32))
		case reflect.Float64:
			parsedRow = append(parsedRow, strconv.FormatFloat(v.Float(), 'f', w.config.FloatPrecision, 64))
		case reflect.Bool:
			parsedRow = append(parsedRow, strconv.FormatBool(v.Bool()))
		default:
			if timeValue, ok := value.(time.Time); ok {
				if strings.ToLower(w.config.DatetimeFormat) == "unix" {
					parsedRow = append(parsedRow, strconv.FormatInt(timeValue.Unix(), 10))
				} else {
					parsedRow = append(parsedRow, timeValue.Format(w.config.DatetimeFormat))
				}
			} else if uuidValue, ok := value.(uuid.UUID); ok {
				parsedRow = append(parsedRow, uuidValue.String())
			} else {
				return nil, errors.Errorf("unsupported type of value %v for CSV writer: %T", value, value)
			}
		}
	}

	return parsedRow, nil
}

// writeRow function write row to CSV file.
func (w *Writer) writeRow(row []string) error {
	if w.csvWriter == nil || w.totalWrittenRows%w.model.RowsPerFile == 0 {
		err := w.switchToNextFile(w.totalWrittenRows / w.model.RowsPerFile)
		if err != nil {
			return err
		}
	}

	w.writerMutex.Lock()
	if err := w.csvWriter.Write(row); err != nil {
		return errors.New(err.Error())
	}

	w.bufferedRows++
	w.writerMutex.Unlock()

	w.totalWrittenRows++

	return nil
}

// switchToNextFile function stops writing to the file and switches to a new one.
func (w *Writer) switchToNextFile(fileNumber uint64) error {
	if w.csvWriter != nil {
		if err := w.flush(); err != nil {
			return errors.New(err.Error())
		}

		w.writerMutex.Lock()

		if err := w.fileDescriptor.Close(); err != nil {
			return errors.New(err.Error())
		}
	}

	w.writerMutex.TryLock()

	err := w.replaceFile(w.getFileName(fileNumber))
	if err != nil {
		return err
	}

	w.writerMutex.Unlock()

	return nil
}

func (w *Writer) getFileName(fileNumber uint64) string {
	return filepath.Join(w.outputPath, fmt.Sprintf("%s_%d.csv", w.model.Name, fileNumber))
}

// replaceFile function replaces the output file with a new one and creates a CSV writer for it.
func (w *Writer) replaceFile(fileName string) error {
	fileExists := true
	if _, err := os.Stat(fileName); os.IsNotExist(err) {
		fileExists = false
	}

	flags := os.O_WRONLY | os.O_CREATE
	if w.continueGeneration {
		flags |= os.O_APPEND
	} else {
		flags |= os.O_TRUNC
	}

	file, err := os.OpenFile(fileName, flags, os.ModePerm)
	if err != nil {
		return errors.New(err.Error())
	}

	csvWriter := stdCSV.NewWriter(file)
	csvWriter.Comma = []rune(w.config.Delimiter)[0]

	w.csvWriter = csvWriter
	w.fileDescriptor = file

	if !w.config.WithoutHeaders && (!w.continueGeneration || !fileExists) {
		err = w.csvWriter.Write(w.getHeaders())
		if err != nil {
			return errors.New(err.Error())
		}
	}

	return nil
}

func (w *Writer) getHeaders() []string {
	headers := make([]string, 0, len(w.model.Columns)-len(w.columnsToDiscard))

	for _, column := range w.model.Columns {
		if _, exists := w.columnsToDiscard[column.Name]; exists {
			continue
		}

		headers = append(headers, column.Name)
	}

	return headers
}

// WriteRow function sends row to internal queue.
func (w *Writer) WriteRow(row *models.DataRow) error {
	select {
	case <-w.ctx.Done():
		return errors.Errorf("failed to write row: %s", w.ctx.Err().Error())
	case err := <-w.errorsChan:
		return errors.Errorf("failed to write row: %s", err)
	case w.writerChan <- row:
	}

	return nil
}

func (w *Writer) flush() error {
	w.writerMutex.Lock()
	defer w.writerMutex.Unlock()

	w.csvWriter.Flush()

	if err := w.csvWriter.Error(); err != nil {
		return errors.New(err.Error())
	}

	if w.writtenRowsChan != nil {
		w.writtenRowsChan <- w.bufferedRows
	}

	w.bufferedRows = 0

	return nil
}

// Teardown function waits recording finish flushes csv writer buffer and closes opened file descriptor.
func (w *Writer) Teardown() error {
	close(w.writerChan)
	w.writerWg.Wait()

	w.flushTicker.Stop()
	close(w.flushStopChan)
	w.flushWg.Wait()

	w.writerMutex.Lock()
	if w.csvWriter != nil {
		w.writerMutex.Unlock()

		if err := w.flush(); err != nil {
			return err
		}
	}

	w.writerMutex.TryLock()

	if w.fileDescriptor != nil {
		if err := w.fileDescriptor.Close(); err != nil {
			return errors.New(err.Error())
		}
	}

	select {
	case err := <-w.errorsChan:
		return errors.Errorf("failed write row: %s", err)
	default:
		return nil
	}
}
