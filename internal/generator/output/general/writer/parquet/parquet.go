package parquet

import (
	"fmt"
	"io"
	"math"
	"os"
	"path/filepath"
	"reflect"
	"regexp"
	"slices"
	"strconv"
	"sync"
	"time"

	"github.com/apache/arrow-go/v18/arrow"
	"github.com/apache/arrow-go/v18/arrow/array"
	"github.com/apache/arrow-go/v18/arrow/memory"
	"github.com/apache/arrow-go/v18/parquet"
	"github.com/apache/arrow-go/v18/parquet/compress"
	"github.com/apache/arrow-go/v18/parquet/file"
	"github.com/apache/arrow-go/v18/parquet/pqarrow"
	"github.com/google/uuid"
	"github.com/pkg/errors"
	"github.com/tarantool/sdvg/internal/generator/models"
	"github.com/tarantool/sdvg/internal/generator/output/general/writer"
)

const (
	flushInterval = 5 * time.Second
)

var (
	codecsByName = map[string]compress.Compression{
		"UNCOMPRESSED": compress.Codecs.Uncompressed,
		"SNAPPY":       compress.Codecs.Snappy,
		"GZIP":         compress.Codecs.Gzip,
		"LZ4":          compress.Codecs.Lz4,
		"LZ4RAW":       compress.Codecs.Lz4Raw,
		"LZO":          compress.Codecs.Lzo,
		"ZSTD":         compress.Codecs.Zstd,
		"BROTLI":       compress.Codecs.Brotli,
	}

	encodingsByName = map[string]parquet.Encoding{
		"PLAIN":                   parquet.Encodings.Plain,
		"RLE":                     parquet.Encodings.RLE,
		"DELTA_BINARY_PACKED":     parquet.Encodings.DeltaBinaryPacked,
		"DELTA_BYTE_ARRAY":        parquet.Encodings.DeltaByteArray,
		"DELTA_LENGTH_BYTE_ARRAY": parquet.Encodings.DeltaLengthByteArray,
		"BYTE_STREAM_SPLIT":       parquet.Encodings.ByteStreamSplit,
	}
)

// Verify interface compliance in compile time.
var _ writer.Writer = (*Writer)(nil)

// Writer type is implementation of Writer to parquet file.
type Writer struct {
	model              *models.Model
	config             *models.ParquetConfig
	outputPath         string
	continueGeneration bool

	fs                 FileSystem
	parquetModelSchema *arrow.Schema
	parquetWriter      *pqarrow.FileWriter
	writerProperties   *parquet.WriterProperties
	recordBuilder      *array.RecordBuilder
	flushTicker        *time.Ticker

	totalWrittenRows uint64
	bufferedRows     uint64
	writtenRowsChan  chan<- uint64

	errorChan   chan error
	writerMutex *sync.Mutex
	started     bool
	stopCh      chan struct{}
}

type FileSystem interface {
	NewFileWriter(fileName string) (io.WriteCloser, error)
	NewLocalFileReader(fileName string) (parquet.ReaderAtSeeker, error)
	FindFilesWithExtension(dir, ext string) ([]string, error)
	FindFilesWithPrefix(dir, prefix string) ([]string, error)
	Stat(name string) (os.FileInfo, error)
}

// NewWriter function creates Writer object.
func NewWriter(
	model *models.Model,
	config *models.ParquetConfig,
	fs FileSystem,
	outputPath string,
	continueGeneration bool,
	writtenRowsChan chan<- uint64,
) *Writer {
	return &Writer{
		model:              model,
		config:             config,
		outputPath:         outputPath,
		continueGeneration: continueGeneration,
		fs:                 fs,
		flushTicker:        time.NewTicker(flushInterval),
		writtenRowsChan:    writtenRowsChan,
		errorChan:          make(chan error),
		writerMutex:        &sync.Mutex{},
		started:            false,
		stopCh:             make(chan struct{}),
	}
}

// generateModelSchema function generates scheme of model for parquet.
//
//nolint:cyclop
func (w *Writer) generateModelSchema() (*arrow.Schema, []parquet.WriterProperty, error) {
	writerProperties := []parquet.WriterProperty{
		parquet.WithCompression(codecsByName[w.config.CompressionCodec]),
		parquet.WithDictionaryDefault(false),
	}

	arrowFields := make([]arrow.Field, 0, len(w.model.Columns))

	partitionColumnsByName := map[string]*models.PartitionColumn{}
	for _, column := range w.model.PartitionColumns {
		partitionColumnsByName[column.Name] = column
	}

	for _, column := range w.model.Columns {
		colSettings, ok := partitionColumnsByName[column.Name]
		if ok && !colSettings.WriteToOutput { // filter partition columns in schema
			continue
		}

		var targetColumn *models.Column // if column is foreign key then create schema according to referenced column
		if column.ForeignKeyColumn != nil {
			targetColumn = column.ForeignKeyColumn
		} else {
			targetColumn = column
		}

		nullable := w.isNullableColumn(targetColumn)

		var columnSchemaParquet arrow.Field

		switch targetColumn.Type {
		case "integer":
			arrowType, err := w.getIntegerArrowType(targetColumn)
			if err != nil {
				return nil, nil, err
			}

			columnSchemaParquet = arrow.Field{Name: column.Name, Type: arrowType, Nullable: nullable}
		case "float":
			var arrowType = arrow.PrimitiveTypes.Float32

			for _, r := range targetColumn.Ranges {
				if r.FloatParams == nil || r.FloatParams.BitWidth == 64 {
					arrowType = arrow.PrimitiveTypes.Float64

					break
				}
			}

			columnSchemaParquet = arrow.Field{Name: column.Name, Type: arrowType, Nullable: nullable}
		case "string", "uuid":
			columnSchemaParquet = arrow.Field{Name: column.Name, Type: arrow.BinaryTypes.String, Nullable: nullable}
		case "datetime":
			arrowType, err := w.getDateTimeTypeByFormat()
			if err != nil {
				return nil, nil, err
			}

			columnSchemaParquet = arrow.Field{Name: column.Name, Type: arrowType, Nullable: nullable}
		default:
			return nil, nil, errors.Errorf("unknown column type: %v", column.Type)
		}

		if targetColumn.ParquetParams != nil {
			encoding := targetColumn.ParquetParams.Encoding
			switch encoding {
			case "PLAIN_DICT", "RLE_DICTIONARY":
				writerProperties = append(writerProperties,
					parquet.WithDictionaryFor(column.Name, true),
				)
			default:
				writerProperties = append(writerProperties,
					parquet.WithEncodingFor(column.Name, encodingsByName[encoding]))
			}
		}

		arrowFields = append(arrowFields, columnSchemaParquet)
	}

	schema := arrow.NewSchema(
		arrowFields,
		nil,
	)

	return schema, writerProperties, nil
}

func (w *Writer) isNullableColumn(column *models.Column) bool {
	for _, r := range column.Ranges {
		if r.NullPercentage > 0 {
			return true
		}

		if r.Values != nil {
			if slices.Contains(r.Values, nil) {
				return true
			}
		}
	}

	return false
}

func (w *Writer) getIntegerArrowType(column *models.Column) (arrow.DataType, error) {
	var (
		maxBitWidth int
	)

	for _, r := range column.Ranges {
		if r.IntegerParams == nil { // if enum values set make output type int64
			maxBitWidth = 64

			break
		}

		maxBitWidth = max(maxBitWidth, r.IntegerParams.BitWidth)
	}

	var arrowType arrow.DataType

	//nolint:mnd
	switch maxBitWidth {
	case 8:
		arrowType = arrow.PrimitiveTypes.Int8
	case 16:
		arrowType = arrow.PrimitiveTypes.Int16
	case 32:
		arrowType = arrow.PrimitiveTypes.Int32
	case 64:
		arrowType = arrow.PrimitiveTypes.Int64
	default:
		return nil, errors.Errorf("unknown integer maxBitWidth format %v", maxBitWidth)
	}

	return arrowType, nil
}

func (w *Writer) getDateTimeTypeByFormat() (arrow.DataType, error) {
	switch w.config.DateTimeFormat {
	case models.ParquetDateTimeMillisFormat:
		return arrow.FixedWidthTypes.Timestamp_ms, nil
	case models.ParquetDateTimeMicrosFormat:
		return arrow.FixedWidthTypes.Timestamp_us, nil
	default:
		return nil, errors.Errorf("unknown datetime format %v", w.config.DateTimeFormat)
	}
}

// Init function creates output file and starts receiving row from internal queue.
func (w *Writer) Init() error {
	if w.started {
		return errors.New("the writer has already been initialized")
	}

	modelSchema, writerProperties, err := w.generateModelSchema()
	if err != nil {
		return err
	}

	w.parquetModelSchema = modelSchema
	w.writerProperties = parquet.NewWriterProperties(writerProperties...)
	w.recordBuilder = array.NewRecordBuilder(memory.DefaultAllocator, w.parquetModelSchema)
	//nolint:mnd,godox
	// TODO: find optimal value, or calculate it to flush on disk 512Mb data
	w.recordBuilder.Reserve(5000)

	if err = os.MkdirAll(w.outputPath, os.ModePerm); err != nil {
		return errors.New(err.Error())
	}

	if w.continueGeneration {
		savedRows, err := w.getSavedRows()
		if err != nil {
			return err
		}

		w.totalWrittenRows = savedRows
	}

	w.started = true

	go w.flusher()

	return nil
}

func (w *Writer) flusher() {
	for {
		select {
		case <-w.stopCh:
			return
		case <-w.flushTicker.C:
			//nolint:godox
			// TODO: find optimal value, or calculate it to flush on disk 512Mb data
			if w.parquetWriter != nil {
				err := w.flush()
				if err != nil {
					w.errorChan <- err

					return
				}
			}
		}
	}
}

func (w *Writer) flush() error {
	w.writerMutex.Lock()
	defer w.writerMutex.Unlock()

	// reset the RecordBuilder, so it can be used to build a new record.
	record := w.recordBuilder.NewRecord()

	if err := w.parquetWriter.WriteBuffered(record); err != nil {
		return errors.New(err.Error())
	}

	if w.writtenRowsChan != nil {
		w.writtenRowsChan <- w.bufferedRows
	}

	w.bufferedRows = 0

	return nil
}

func (w *Writer) getSavedRows() (uint64, error) {
	fileNumber, err := w.getFileNumber()
	if err != nil {
		return 0, err
	}

	savedRows := uint64(fileNumber) * w.model.RowsPerFile
	fileName := fmt.Sprintf("%s_%d", w.model.Name, fileNumber)

	filePart, exists, err := w.getFilePartNumber(fileName)
	if err != nil {
		return 0, err
	}

	if !exists {
		return savedRows, nil
	}

	for i := 0; i <= filePart; i++ {
		fileNameWithPart := fmt.Sprintf("%s_%d.parquet", fileName, filePart)

		rowsCount, err := w.getRowsInFile(fileNameWithPart)
		if err != nil {
			return 0, errors.WithMessagef(
				err,
				"failed to count the number of written rows in file %q", fileName,
			)
		}

		savedRows += rowsCount
	}

	return savedRows, nil
}

func (w *Writer) getFileNumber() (int, error) {
	fileNames, err := w.fs.FindFilesWithExtension(w.outputPath, ".parquet")
	if err != nil {
		return 0, errors.WithMessagef(err, "failed to get number of file in directory %q", w.outputPath)
	}

	if len(fileNames) == 0 {
		return 0, nil
	}

	var fileNumber int

	re := regexp.MustCompile(fmt.Sprintf("^%s_(\\d+)_\\d+\\.parquet$", w.model.Name))

	for _, name := range fileNames {
		matches := re.FindStringSubmatch(name)

		if len(matches) > 1 {
			number, _ := strconv.Atoi(matches[1])
			fileNumber = max(fileNumber, number)
		}
	}

	return fileNumber, nil
}

func (w *Writer) getFilePartNumber(fileNameWithNumber string) (int, bool, error) {
	fileNames, err := w.fs.FindFilesWithPrefix(w.outputPath, fileNameWithNumber)
	if err != nil {
		return 0, false, errors.WithMessagef(err, "failed to get part number of file %q", fileNameWithNumber)
	}

	if len(fileNames) == 0 {
		return 0, false, nil
	}

	var (
		part  int
		exist bool
	)

	re := regexp.MustCompile(fmt.Sprintf("^%s_(\\d+)\\.parquet$", fileNameWithNumber))

	for _, name := range fileNames {
		matches := re.FindStringSubmatch(name)

		if len(matches) > 1 {
			filePart, _ := strconv.Atoi(matches[1])
			part = max(part, filePart)
			exist = true
		}
	}

	return part, exist, nil
}

func (w *Writer) getRowsInFile(fileName string) (uint64, error) {
	fullPath := filepath.Join(w.outputPath, fileName)

	if _, err := w.fs.Stat(fullPath); os.IsNotExist(err) {
		return 0, nil
	}

	f, err := w.fs.NewLocalFileReader(fullPath)
	if err != nil {
		return 0, errors.New(err.Error())
	}

	parquetReader, err := file.NewParquetReader(f)
	if err != nil {
		return 0, errors.New(err.Error())
	}
	defer parquetReader.Close()

	fileReader, err := pqarrow.NewFileReader(parquetReader, pqarrow.ArrowReadProperties{}, memory.DefaultAllocator)
	if err != nil {
		return 0, errors.New(err.Error())
	}

	return uint64(fileReader.ParquetReader().NumRows()), nil
}

// parseDataRow function parses raw data into data that can be written to parquet.
func (w *Writer) parseDataRow(row *models.DataRow) error {
	for i, value := range row.Values {
		if value == nil {
			continue
		}

		v := reflect.ValueOf(value)

		switch v.Kind() {
		case reflect.String, reflect.Int64:
		case reflect.Float32:
			row.Values[i] = float32(roundFloat(v.Float(), w.config.FloatPrecision))
		case reflect.Float64:
			row.Values[i] = roundFloat(v.Float(), w.config.FloatPrecision)
		case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32:
			row.Values[i] = int32(v.Int())
		default:
			if _, ok := value.(time.Time); ok {
				continue
			} else if uuidValue, ok := value.(uuid.UUID); ok {
				row.Values[i] = uuidValue.String()
			} else {
				return errors.Errorf("unsupported type of value %v for parquet writer: %T", value, value)
			}
		}
	}

	return nil
}

// roundFloat function rounds the float to the specified precision.
func roundFloat(val float64, precision int) float64 {
	ratio := math.Pow(10, float64(precision)) //nolint:mnd

	return math.Round(val*ratio) / ratio
}

// writeRow function write row to parquet file.
func (w *Writer) writeRow(row *models.DataRow) error {
	if w.parquetWriter == nil || w.totalWrittenRows%w.model.RowsPerFile == 0 {
		err := w.switchToNextFile(w.totalWrittenRows / w.model.RowsPerFile)
		if err != nil {
			return err
		}
	}

	w.writerMutex.Lock()
	if err := w.appendValuesToBuilder(w.recordBuilder, row.Values); err != nil {
		return errors.WithMessage(err, "failed to build parquet record")
	}

	w.bufferedRows++
	defer w.writerMutex.Unlock()

	w.totalWrittenRows++

	return nil
}

// switchToNextFile function stops writing to the file and switches to a new one.
func (w *Writer) switchToNextFile(fileNumber uint64) error {
	if w.parquetWriter != nil {
		err := w.flush()
		if err != nil {
			return errors.New(err.Error())
		}

		w.writerMutex.Lock()

		err = w.parquetWriter.Close()
		if err != nil {
			return errors.New(err.Error())
		}
	}

	w.writerMutex.TryLock()

	fileName, err := w.getFileName(fileNumber)
	if err != nil {
		return err
	}

	err = w.replaceFile(fileName)
	if err != nil {
		return err
	}

	w.writerMutex.Unlock()

	return nil
}

//nolint:cyclop
func (w *Writer) appendValuesToBuilder(rb *array.RecordBuilder, values []any) error {
	for i, value := range values {
		fb := rb.Fields()[i]

		if value == nil {
			fb.AppendNull()

			continue
		}

		v := reflect.ValueOf(value)
		//nolint:forcetypeassert
		switch v.Kind() {
		case reflect.Int8:
			fb.(*array.Int8Builder).AppendValues([]int8{int8(v.Int())}, nil)
		case reflect.Int16:
			fb.(*array.Int16Builder).AppendValues([]int16{int16(v.Int())}, nil)
		case reflect.Int32:
			fb.(*array.Int32Builder).AppendValues([]int32{int32(v.Int())}, nil)
		case reflect.Int, reflect.Int64:
			fb.(*array.Int64Builder).AppendValues([]int64{v.Int()}, nil)
		case reflect.Float32:
			fb.(*array.Float32Builder).AppendValues([]float32{float32(v.Float())}, nil)
		case reflect.Float64:
			fb.(*array.Float64Builder).AppendValues([]float64{v.Float()}, nil)
		case reflect.String:
			fb.(*array.StringBuilder).AppendValues([]string{v.String()}, nil)
		case reflect.Bool:
			fb.(*array.BooleanBuilder).AppendValues([]bool{v.Bool()}, nil)
		default:
			if timeValue, ok := value.(time.Time); ok {
				var intTimeValue int64

				switch w.config.DateTimeFormat {
				case models.ParquetDateTimeMillisFormat:
					intTimeValue = timeValue.UnixMilli()
				case models.ParquetDateTimeMicrosFormat:
					intTimeValue = timeValue.UnixMicro()
				default:
					return errors.New("unsupported datetime format for parquet writer")
				}

				fb.(*array.TimestampBuilder).Append(arrow.Timestamp(intTimeValue))
			} else {
				return errors.Errorf("unsupported type of value %v for parquet writer: %T", value, value)
			}
		}
	}

	return nil
}

func (w *Writer) getFileName(fileNumber uint64) (string, error) {
	fileName := fmt.Sprintf("%s_%d", w.model.Name, fileNumber)

	var filePart int

	if w.continueGeneration {
		part, exists, err := w.getFilePartNumber(fileName)
		if err != nil {
			return "", err
		}

		if exists {
			part++
		}

		filePart = part
	}

	return filepath.Join(w.outputPath, fmt.Sprintf("%s_%d.parquet", fileName, filePart)), nil
}

// replaceFile function replaces the output file with a new one and creates a parquet writer for it.
func (w *Writer) replaceFile(fileName string) error {
	parquetFileWriter, err := w.fs.NewFileWriter(fileName)
	if err != nil {
		return err
	}

	//nolint:lll
	pWriter, err := pqarrow.NewFileWriter(w.parquetModelSchema, parquetFileWriter, w.writerProperties, pqarrow.DefaultWriterProps())
	if err != nil {
		return errors.New(err.Error())
	}

	w.parquetWriter = pWriter

	return nil
}

// WriteRow function sends row to internal queue.
func (w *Writer) WriteRow(row *models.DataRow) error {
	if err := w.parseDataRow(row); err != nil {
		return errors.WithMessage(err, "failed to parse data row")
	}

	if err := w.writeRow(row); err != nil {
		return errors.Errorf("failed write row: %s", err)
	}

	select {
	case err := <-w.errorChan:
		return errors.Errorf("failed write row: %s", err)
	default:
		return nil
	}
}

// Teardown function waits recording finish and stops parquet writer and closes opened file descriptor.
func (w *Writer) Teardown() error {
	w.flushTicker.Stop()
	w.stopCh <- struct{}{}

	if err := w.flush(); err != nil {
		return errors.New(err.Error())
	}

	if err := w.parquetWriter.Close(); err != nil {
		return errors.New(err.Error())
	}

	select {
	case err := <-w.errorChan:
		return errors.Errorf("failed write row: %s", err)
	default:
		return nil
	}
}
