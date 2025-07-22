package parquet

import (
	"context"
	"fmt"
	"io"
	"math"
	"path/filepath"
	"testing"
	"time"

	"github.com/apache/arrow-go/v18/arrow"
	"github.com/apache/arrow-go/v18/arrow/array"
	"github.com/apache/arrow-go/v18/arrow/memory"
	"github.com/apache/arrow-go/v18/parquet"
	"github.com/apache/arrow-go/v18/parquet/file"
	"github.com/apache/arrow-go/v18/parquet/pqarrow"
	"github.com/google/uuid"
	"github.com/pkg/errors"
	"github.com/stretchr/testify/require"
	"github.com/tarantool/sdvg/internal/generator/models"
)

//nolint:maintidx
func TestGetModelSchema(t *testing.T) {
	type testCase struct {
		name                     string
		cfg                      *models.ParquetConfig
		model                    *models.Model
		expectedModelSchema      arrow.Schema
		expectedWriterProperties []parquet.WriterProperty
	}

	testCases := []testCase{
		{
			name: "numbers model: integers, float and double",
			cfg: &models.ParquetConfig{
				CompressionCodec: "UNCOMPRESSED",
				FloatPrecision:   3,
				DateTimeFormat:   models.ParquetDateTimeMillisFormat,
			},
			model: &models.Model{
				Name: "numbers",
				Columns: []*models.Column{
					{
						Name: "integer_8-name",
						Type: "integer",
						Ranges: []*models.Params{
							{
								IntegerParams: &models.ColumnIntegerParams{
									BitWidth: 8,
								},
							},
						},
						ParquetParams: &models.ColumnParquetParams{
							Encoding: "PLAIN",
						},
					},
					{
						Name: "integer_16-name",
						Type: "integer",
						Ranges: []*models.Params{
							{
								IntegerParams: &models.ColumnIntegerParams{
									BitWidth: 16,
								},
							},
						},
						ParquetParams: &models.ColumnParquetParams{
							Encoding: "PLAIN",
						},
					},
					{
						Name: "integer_32-name",
						Type: "integer",
						Ranges: []*models.Params{
							{
								IntegerParams: &models.ColumnIntegerParams{
									BitWidth: 32,
								},
							},
						},
						ParquetParams: &models.ColumnParquetParams{
							Encoding: "PLAIN",
						},
					},
					{
						Name: "integer_64-name-plain",
						Type: "integer",
						Ranges: []*models.Params{
							{
								IntegerParams: &models.ColumnIntegerParams{
									BitWidth: 64,
								},
							},
						},
						ParquetParams: &models.ColumnParquetParams{
							Encoding: "PLAIN",
						},
					},
					{
						Name: "integer_64-name-optional-rle",
						Type: "integer",
						Ranges: []*models.Params{
							{
								IntegerParams: &models.ColumnIntegerParams{
									BitWidth: 64,
								},
								NullPercentage: 0.5,
							},
						},
						ParquetParams: &models.ColumnParquetParams{
							Encoding: "RLE_DICTIONARY",
						},
					},
					{
						Name: "integer_64-name-dbp",
						Type: "integer",
						Ranges: []*models.Params{
							{
								IntegerParams: &models.ColumnIntegerParams{
									BitWidth: 64,
								},
							},
						},
						ParquetParams: &models.ColumnParquetParams{
							Encoding: "DELTA_BINARY_PACKED",
						},
					},
					{
						Name: "float_32-dba",
						Type: "float",
						ParquetParams: &models.ColumnParquetParams{
							Encoding: "DELTA_BYTE_ARRAY",
						},
					},
					{
						Name: "float_64-dlba",
						Type: "float",
						Ranges: []*models.Params{
							{
								FloatParams: &models.ColumnFloatParams{
									BitWidth: 64,
								},
							},
						},
						ParquetParams: &models.ColumnParquetParams{
							Encoding: "DELTA_LENGTH_BYTE_ARRAY",
						},
					},
				},
				PartitionColumns: make([]*models.PartitionColumn, 0),
			},
			expectedModelSchema: *arrow.NewSchema([]arrow.Field{
				{Name: "integer_8-name", Type: arrow.PrimitiveTypes.Int8, Nullable: false},
				{Name: "integer_16-name", Type: arrow.PrimitiveTypes.Int16, Nullable: false},
				{Name: "integer_32-name", Type: arrow.PrimitiveTypes.Int32, Nullable: false},
				{Name: "integer_64-name-plain", Type: arrow.PrimitiveTypes.Int64, Nullable: false},
				{Name: "integer_64-name-optional-rle", Type: arrow.PrimitiveTypes.Int64, Nullable: true},
				{Name: "integer_64-name-dbp", Type: arrow.PrimitiveTypes.Int64, Nullable: false},
				{Name: "float_32-dba", Type: arrow.PrimitiveTypes.Float32, Nullable: false},
				{Name: "float_64-dlba", Type: arrow.PrimitiveTypes.Float64, Nullable: false},
			}, nil),
			expectedWriterProperties: []parquet.WriterProperty{
				parquet.WithCompression(codecsByName["UNCOMPRESSED"]),
				parquet.WithDictionaryDefault(false),
				parquet.WithEncodingFor("integer_8-name", encodingsByName["PLAIN"]),
				parquet.WithEncodingFor("integer_16-name", encodingsByName["PLAIN"]),
				parquet.WithEncodingFor("integer_32-name", encodingsByName["PLAIN"]),
				parquet.WithEncodingFor("integer_64-name-plain", encodingsByName["PLAIN"]),
				parquet.WithDictionaryFor("integer_64-name-optional-rle", true),
				parquet.WithEncodingFor("integer_64-name-dbp", encodingsByName["DELTA_BINARY_PACKED"]),
				parquet.WithEncodingFor("float_32-dba", encodingsByName["DELTA_BYTE_ARRAY"]),
				parquet.WithEncodingFor("float_64-dlba", encodingsByName["DELTA_LENGTH_BYTE_ARRAY"]),
			},
		},
		{
			name: "model with string and uuid",
			cfg: &models.ParquetConfig{
				CompressionCodec: "UNCOMPRESSED",
				FloatPrecision:   3,
				DateTimeFormat:   models.ParquetDateTimeMillisFormat,
			},
			model: &models.Model{
				Name: "strings",
				Columns: []*models.Column{
					{
						Name: "utf8StringName",
						Type: "string",
						ParquetParams: &models.ColumnParquetParams{
							Encoding: "PLAIN",
						},
					},
					{
						Name: "uuidStringName",
						Type: "uuid",
						ParquetParams: &models.ColumnParquetParams{
							Encoding: "PLAIN",
						},
					},
				},
				PartitionColumns: make([]*models.PartitionColumn, 0),
			},
			expectedModelSchema: *arrow.NewSchema([]arrow.Field{
				{Name: "utf8StringName", Type: arrow.BinaryTypes.String, Nullable: false},
				{Name: "uuidStringName", Type: arrow.BinaryTypes.String, Nullable: false},
			}, nil),
			expectedWriterProperties: []parquet.WriterProperty{
				parquet.WithCompression(codecsByName["UNCOMPRESSED"]),
				parquet.WithDictionaryDefault(false),
				parquet.WithEncodingFor("utf8StringName", encodingsByName["PLAIN"]),
				parquet.WithEncodingFor("uuidStringName", encodingsByName["PLAIN"]),
			},
		},

		{
			name: "DateTime millis",
			cfg: &models.ParquetConfig{
				CompressionCodec: "UNCOMPRESSED",
				FloatPrecision:   3,
				DateTimeFormat:   models.ParquetDateTimeMillisFormat,
			},
			model: &models.Model{
				Name: "milliseconds",
				Columns: []*models.Column{
					{
						Name: "datetimeMillisName",
						Type: "datetime",
						ParquetParams: &models.ColumnParquetParams{
							Encoding: "PLAIN",
						},
					},
				},
				PartitionColumns: make([]*models.PartitionColumn, 0),
			},
			expectedModelSchema: *arrow.NewSchema([]arrow.Field{
				{Name: "datetimeMillisName", Type: arrow.FixedWidthTypes.Timestamp_ms, Nullable: false},
			}, nil),
			expectedWriterProperties: []parquet.WriterProperty{
				parquet.WithCompression(codecsByName["UNCOMPRESSED"]),
				parquet.WithDictionaryDefault(false),
				parquet.WithEncodingFor("datetimeMillisName", encodingsByName["PLAIN"]),
			},
		},
		{
			name: "DateTime micros",
			cfg: &models.ParquetConfig{
				CompressionCodec: "UNCOMPRESSED",
				FloatPrecision:   3,
				DateTimeFormat:   models.ParquetDateTimeMicrosFormat,
			},
			model: &models.Model{
				Name: "microseconds",
				Columns: []*models.Column{
					{
						Name: "datetimeMicrosName",
						Type: "datetime",
						ParquetParams: &models.ColumnParquetParams{
							Encoding: "PLAIN",
						},
					},
				},
				PartitionColumns: make([]*models.PartitionColumn, 0),
			},
			expectedModelSchema: *arrow.NewSchema([]arrow.Field{
				{Name: "datetimeMicrosName", Type: arrow.FixedWidthTypes.Timestamp_us, Nullable: false},
			}, nil),
			expectedWriterProperties: []parquet.WriterProperty{
				parquet.WithCompression(codecsByName["UNCOMPRESSED"]),
				parquet.WithDictionaryDefault(false),
				parquet.WithEncodingFor("datetimeMicrosName", encodingsByName["PLAIN"]),
			},
		},
		{
			name: "foreign key column",
			cfg: &models.ParquetConfig{
				CompressionCodec: "UNCOMPRESSED",
				FloatPrecision:   3,
				DateTimeFormat:   models.ParquetDateTimeMicrosFormat,
			},
			model: &models.Model{
				Name: "foreign",
				Columns: []*models.Column{
					{
						Name: "foreignKeyColumnName",
						ForeignKeyColumn: &models.Column{
							Name: "referencedColumnName",
							Type: "integer",
							Ranges: []*models.Params{
								{
									IntegerParams: &models.ColumnIntegerParams{
										BitWidth: 8,
									},
									NullPercentage: 0.5,
								},
								{
									IntegerParams: &models.ColumnIntegerParams{
										BitWidth: 64,
									},
								},
							},
							ParquetParams: &models.ColumnParquetParams{
								Encoding: "RLE_DICTIONARY",
							},
						},
					},
				},
				PartitionColumns: make([]*models.PartitionColumn, 0),
			},
			expectedModelSchema: *arrow.NewSchema([]arrow.Field{
				{Name: "foreignKeyColumnName", Type: arrow.PrimitiveTypes.Int64, Nullable: true},
			}, nil),
			expectedWriterProperties: []parquet.WriterProperty{
				parquet.WithCompression(codecsByName["UNCOMPRESSED"]),
				parquet.WithDictionaryDefault(false),
				parquet.WithDictionaryFor("foreignKeyColumnName", true),
			},
		},
		{
			name: "enum values column nullable",
			cfg: &models.ParquetConfig{
				CompressionCodec: "UNCOMPRESSED",
				FloatPrecision:   3,
				DateTimeFormat:   models.ParquetDateTimeMicrosFormat,
			},
			model: &models.Model{
				Name: "enum_nullable",
				Columns: []*models.Column{
					{
						Name: "valuesColumn",
						Type: "integer",
						Ranges: []*models.Params{
							{
								Values: []any{111, nil, 333},
							},
						},
					},
				},
			},
			expectedModelSchema: *arrow.NewSchema([]arrow.Field{
				{Name: "valuesColumn", Type: arrow.PrimitiveTypes.Int64, Nullable: true},
			}, nil),
			expectedWriterProperties: []parquet.WriterProperty{
				parquet.WithCompression(codecsByName["UNCOMPRESSED"]),
				parquet.WithDictionaryDefault(false),
			},
		},
		{
			name: "enum values column NON nullable",
			cfg: &models.ParquetConfig{
				CompressionCodec: "UNCOMPRESSED",
				FloatPrecision:   3,
				DateTimeFormat:   models.ParquetDateTimeMicrosFormat,
			},
			model: &models.Model{
				Name: "enum_non_nullable",
				Columns: []*models.Column{
					{
						Name: "valuesColumn",
						Type: "integer",
						Ranges: []*models.Params{
							{
								Values: []any{111, 333},
							},
						},
					},
				},
			},
			expectedModelSchema: *arrow.NewSchema([]arrow.Field{
				{Name: "valuesColumn", Type: arrow.PrimitiveTypes.Int64, Nullable: false},
			}, nil),
			expectedWriterProperties: []parquet.WriterProperty{
				parquet.WithCompression(codecsByName["UNCOMPRESSED"]),
				parquet.WithDictionaryDefault(false),
			},
		},
		{
			name: "Partition with not writeable column",
			cfg: &models.ParquetConfig{
				CompressionCodec: "UNCOMPRESSED",
				FloatPrecision:   3,
				DateTimeFormat:   models.ParquetDateTimeMillisFormat,
			},
			model: &models.Model{
				Name:        "users",
				RowsCount:   100,
				RowsPerFile: 100,
				Columns: []*models.Column{
					{
						Name: "uuid_field",
						Type: "uuid",
						ParquetParams: &models.ColumnParquetParams{
							Encoding: "PLAIN",
						},
					},
					{
						Name: "int_field",
						Type: "integer",
						Ranges: []*models.Params{
							{
								IntegerParams: &models.ColumnIntegerParams{
									BitWidth: 32,
									From:     math.MinInt32,
									To:       math.MaxInt32,
								},
							},
						},

						ParquetParams: &models.ColumnParquetParams{
							Encoding: "PLAIN",
						},
					},
					// NOT WRITEABLE PARTITION COLUMNS MUST BE AT THE END
					{
						Name: "float_field",
						Type: "float",
						Ranges: []*models.Params{
							{
								FloatParams: &models.ColumnFloatParams{
									BitWidth: 32,
									From:     -math.MaxFloat32,
									To:       math.MaxFloat32,
								},
							},
						},
						ParquetParams: &models.ColumnParquetParams{
							Encoding: "DELTA_BYTE_ARRAY",
						},
					},
				},
				PartitionColumns: []*models.PartitionColumn{
					{
						Name:          "float_field",
						WriteToOutput: false,
					},
					{
						Name:          "uuid_field",
						WriteToOutput: true,
					},
				},
			},
			expectedModelSchema: *arrow.NewSchema([]arrow.Field{
				{Name: "uuid_field", Type: arrow.BinaryTypes.String, Nullable: false},
				{Name: "int_field", Type: arrow.PrimitiveTypes.Int32, Nullable: false},
			}, nil),
			expectedWriterProperties: []parquet.WriterProperty{
				parquet.WithCompression(codecsByName["UNCOMPRESSED"]),
				parquet.WithDictionaryDefault(false),
				parquet.WithEncodingFor("uuid_field", encodingsByName["PLAIN"]),
				parquet.WithEncodingFor("int_field", encodingsByName["PLAIN"]),
			},
		},
	}

	fsMock := newFileSystemMock()

	testFunc := func(t *testing.T, tc testCase) {
		t.Helper()

		require.NotEqual(t, "", tc.model.Name)

		writer := &Writer{
			model:      tc.model,
			config:     tc.cfg,
			fs:         fsMock,
			outputPath: "./",
		}

		modelSchemaPointer, writerProperties, err := writer.generateModelSchema()
		require.NoError(t, err)

		require.Equal(t, tc.expectedModelSchema, *modelSchemaPointer)

		gotWriterProps := parquet.NewWriterProperties(writerProperties...)
		expectedWriterProps := parquet.NewWriterProperties(tc.expectedWriterProperties...)

		require.Equal(t, expectedWriterProps, gotWriterProps)
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) { testFunc(t, tc) })
	}
}

func getFileNumber(rows, rowsPerFile int) int {
	fileNumber := rows / rowsPerFile
	if rows%rowsPerFile != 0 {
		fileNumber++
	}

	return fileNumber
}

func TestWriteRow(t *testing.T) {
	// GIVEN
	parquetConfig := &models.ParquetConfig{
		CompressionCodec: "UNCOMPRESSED",
		FloatPrecision:   3,
		DateTimeFormat:   models.ParquetDateTimeMillisFormat,
	}

	testColumns := []*models.Column{
		{
			Name: "integerName",
			Type: "integer",
			Ranges: []*models.Params{
				{IntegerParams: &models.ColumnIntegerParams{BitWidth: 64}},
			},

			ParquetParams: &models.ColumnParquetParams{
				Encoding: "PLAIN",
			},
		},
		{
			Name: "stringName",
			Type: "string",
			ParquetParams: &models.ColumnParquetParams{
				Encoding: "RLE_DICTIONARY",
			},
		},
		{
			Name: "floatName",
			Type: "float",
			Ranges: []*models.Params{
				{FloatParams: &models.ColumnFloatParams{BitWidth: 32}},
			},

			ParquetParams: &models.ColumnParquetParams{
				Encoding: "PLAIN",
			},
		},
		{
			Name: "doubleName",
			Type: "float",
			Ranges: []*models.Params{
				{FloatParams: &models.ColumnFloatParams{BitWidth: 64}},
			},

			ParquetParams: &models.ColumnParquetParams{
				Encoding: "PLAIN",
			},
		},
		{
			Name: "datetimeName",
			Type: "datetime",
			ParquetParams: &models.ColumnParquetParams{
				Encoding: "PLAIN",
			},
		},
		{
			Name: "nilName",
			Type: "string",
			Ranges: []*models.Params{
				{NullPercentage: 1},
			},
			ParquetParams: &models.ColumnParquetParams{
				Encoding: "PLAIN",
			},
		},
		{
			Name: "uuidName",
			Type: "uuid",
			ParquetParams: &models.ColumnParquetParams{
				Encoding: "PLAIN",
			},
		},
	}

	uuidValue := uuid.New()

	//nolint:lll
	expectedData := []*models.DataRow{
		{Values: []any{int64(1), "first value", float32(0.2), 0.03, time.UnixMilli(time.Now().UnixMilli()).UTC(), nil, uuidValue}},
		{Values: []any{int64(5), "second value", float32(0.6), 0.07, time.UnixMilli(time.Now().UnixMilli()).UTC(), nil, uuidValue}},
		{Values: []any{int64(6), "third value", float32(0.7), 0.08, time.UnixMilli(time.Now().UnixMilli()).UTC(), nil, uuidValue}},
		{Values: []any{int64(10), "fourth value", float32(0.11), 0.012, time.UnixMilli(time.Now().UnixMilli()).UTC(), nil, uuidValue}},
	}

	type testCase struct {
		name   string
		config *models.ParquetConfig
		model  *models.Model
		data   []*models.DataRow
	}

	testCases := []testCase{
		{
			name: "Rows per file equals rows count",
			model: &models.Model{
				Name:             "",
				RowsCount:        4,
				RowsPerFile:      4,
				Columns:          testColumns,
				PartitionColumns: make([]*models.PartitionColumn, 0),
			},
			config: parquetConfig,
			data:   expectedData,
		},
		{
			name: "Rows per file not equals rows count",
			model: &models.Model{
				Name:             "",
				RowsCount:        4,
				RowsPerFile:      2,
				Columns:          testColumns,
				PartitionColumns: make([]*models.PartitionColumn, 0),
			},
			config: parquetConfig,
			data:   expectedData,
		},
	}

	testFunc := func(t *testing.T, tc testCase) {
		t.Helper()

		// WHEN

		fsMock := newFileSystemMock()
		parquetWriter := NewWriter(tc.model, parquetConfig, fsMock, "./", false, nil)

		err := parquetWriter.Init()
		require.NoError(t, err)

		for _, row := range tc.data {
			err = parquetWriter.WriteRow(row)
			require.NoError(t, err)
		}

		err = parquetWriter.Teardown()
		require.NoError(t, err)

		// THEN

		filesAmountExpected := getFileNumber(len(tc.data), int(tc.model.RowsPerFile))

		var numberReadRows int

		filesRows := make(map[string][][]any, filesAmountExpected)
		fileNamesOrdered := make([]string, 0, len(filesRows))

		for i := range filesAmountExpected {
			fileName, err := parquetWriter.getFileName(uint64(i))
			require.NoError(t, err)

			fileNamesOrdered = append(fileNamesOrdered, fileName)

			rows := readRows(t, fsMock, fileName)

			numberReadRows += len(rows)

			filesRows[fileName] = rows
		}

		require.Len(t, filesRows, filesAmountExpected)
		require.Len(t, tc.data, numberReadRows)

		var i int

		for _, fileName := range fileNamesOrdered {
			for _, row := range filesRows[fileName] {
				require.Equal(t, tc.data[i].Values, row,
					"Mismatch at filename: %v, expected: %v, got: %v", fileName, tc.data[i].Values, row)

				i++
			}
		}
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) { testFunc(t, tc) })
	}
}

//nolint:cyclop
func readRows(t *testing.T, fsMock *fsMock, fileName string) [][]any {
	t.Helper()

	f, err := fsMock.NewLocalFileReader(fileName)
	require.NoError(t, err)

	rdr, err := file.NewParquetReader(f)
	if err != nil {
		require.NoError(t, err)
	}
	defer rdr.Close()

	arrRdr, err := pqarrow.NewFileReader(rdr, pqarrow.ArrowReadProperties{BatchSize: 3}, memory.DefaultAllocator)
	if err != nil {
		require.NoError(t, err)
	}

	rr, err := arrRdr.GetRecordReader(context.TODO(), nil, nil)
	if err != nil {
		require.NoError(t, err)
	}

	rows := make([][]any, 0)

	for {
		rec, err := rr.Read()

		if errors.Is(err, io.EOF) || rec == nil {
			break
		}

		if err != nil {
			require.NoError(t, err)
		}

		numRows := int(rec.NumRows())

		numCols := int(rec.NumCols())

		for rowIdx := range numRows {
			row := make([]any, numCols)

			for colIdx := range numCols {
				arr := rec.Column(colIdx)
				field := rec.Schema().Field(colIdx)

				if arr.IsNull(rowIdx) {
					row[colIdx] = nil

					continue
				}

				//nolint:forcetypeassert
				switch field.Type.ID() {
				case arrow.INT64:
					colArr := arr.(*array.Int64)
					row[colIdx] = colArr.Value(rowIdx)
				case arrow.INT32:
					colArr := arr.(*array.Int32)
					row[colIdx] = colArr.Value(rowIdx)
				case arrow.FLOAT32:
					colArr := arr.(*array.Float32)
					row[colIdx] = colArr.Value(rowIdx)
				case arrow.FLOAT64:
					colArr := arr.(*array.Float64)
					row[colIdx] = colArr.Value(rowIdx)
				case arrow.STRING:
					colArr := arr.(*array.String)
					row[colIdx] = colArr.Value(rowIdx)
				case arrow.TIMESTAMP:
					colArr := arr.(*array.Timestamp)
					row[colIdx] = colArr.Value(rowIdx).ToTime(arrow.Millisecond)
				default:
					t.Fatalf("unhandled type: %s", field.Type)
				}
			}

			rows = append(rows, row)
		}
	}

	return rows
}

func TestWriteToCorrectFiles(t *testing.T) {
	type testCase struct {
		name         string
		rowsPerFile  uint64
		rows         []*models.DataRow
		writersCount int
	}

	config := &models.ParquetConfig{
		CompressionCodec: "UNCOMPRESSED",
	}

	model := &models.Model{
		Name: "test",
		Columns: []*models.Column{
			{
				Name: "id",
				Type: "integer",
				Ranges: []*models.Params{
					{IntegerParams: &models.ColumnIntegerParams{BitWidth: 32}},
				},
			},
		},
	}

	rows := []*models.DataRow{
		{Values: []any{1}},
		{Values: []any{2}},
		{Values: []any{3}},
		{Values: []any{4}},
		{Values: []any{5}},
		{Values: []any{6}},
		{Values: []any{7}},
		{Values: []any{8}},
		{Values: []any{9}},
	}

	testCases := []testCase{
		{
			name:         "Rows per file equals rows count",
			rowsPerFile:  uint64(len(rows)),
			rows:         rows,
			writersCount: 5,
		},
		{
			name:         "Rows per file less than rows count",
			rowsPerFile:  3,
			rows:         rows,
			writersCount: 2,
		},
	}

	testFunc := func(t *testing.T, tc testCase) {
		t.Helper()

		dir := t.TempDir()

		model.RowsPerFile = tc.rowsPerFile

		writersCount := tc.writersCount
		if writersCount == 0 {
			writersCount = 1
		}

		expectedFiles, expectedData := getExpected(tc.rows, tc.rowsPerFile, tc.writersCount)

		fsMock := newFileSystemMock()

		write := func(from, to int, continueGeneration bool) {
			writer := NewWriter(model, config, fsMock, dir, continueGeneration, nil)
			require.NoError(t, writer.Init())

			for i := from; i < to; i++ {
				require.NoError(t, writer.WriteRow(rows[i]))
			}

			require.NoError(t, writer.Teardown())
		}

		rowsCount := len(rows)
		rowsPerWriter := rowsCount / writersCount
		remainder := rowsCount % writersCount

		for i := range writersCount {
			from := i * rowsPerWriter
			to := (i + 1) * rowsPerWriter
			continueGeneration := false

			if i == tc.writersCount-1 {
				to += remainder
			}

			if i != 0 {
				continueGeneration = true
			}

			write(from, to, continueGeneration)
		}

		files, err := fsMock.FindFilesWithExtension(dir, ".parquet")
		require.NoError(t, err)

		require.ElementsMatch(t, expectedFiles, files)

		for _, fileName := range files {
			actualData := readRows(t, fsMock, filepath.Join(dir, fileName))

			require.Equal(t, expectedData[fileName], actualData)
		}
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) { testFunc(t, tc) })
	}
}

func getExpected(rows []*models.DataRow, rowsPerFile uint64, writersCount int) ([]string, map[string][][]any) {
	var expectedFiles []string

	expectedData := make(map[string][][]any)

	totalRows := len(rows)
	currentFileNum := 0
	currentPartNum := 0
	rowsInCurrentFileNum := 0

	for writer := range writersCount {
		start := writer * (totalRows / writersCount)

		end := start + (totalRows / writersCount)
		if writer == writersCount-1 {
			end = totalRows
		}

		if writer > 0 {
			currentPartNum++
		}

		for rowIdx := start; rowIdx < end; rowIdx++ {
			if rowsInCurrentFileNum >= int(rowsPerFile) {
				currentFileNum++
				currentPartNum = 0
				rowsInCurrentFileNum = 0
			}

			fileName := fmt.Sprintf("test_%d_%d.parquet", currentFileNum, currentPartNum)

			if _, exists := expectedData[fileName]; !exists {
				expectedFiles = append(expectedFiles, fileName)
			}

			expectedData[fileName] = append(expectedData[fileName], rows[rowIdx].Values)
			rowsInCurrentFileNum++
		}
	}

	return expectedFiles, expectedData
}
