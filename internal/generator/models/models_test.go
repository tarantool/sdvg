package models

import (
	"math"
	"os"
	"runtime"
	"testing"
	"time"

	"github.com/pkg/errors"
	"github.com/stretchr/testify/require"
)

func TestAppConfigYAMLParse(t *testing.T) {
	type testCase struct {
		name     string
		content  string
		expected AppConfig
		wantErr  error
	}

	defaultHTTPConfig := HTTPConfig{
		ListenAddress: ":8080",
		ReadTimeout:   time.Minute,
		WriteTimeout:  time.Minute,
		IdleTimeout:   time.Minute,
	}

	defaultOpenAIConfig := OpenAI{
		APIKey:  "",
		BaseURL: "",
		Model:   "",
	}

	testCases := []testCase{
		{
			name:    "EmptyConfig",
			content: "{}",
			expected: AppConfig{
				LogFormat:  "text",
				HTTPConfig: defaultHTTPConfig,
				OpenAI:     defaultOpenAIConfig,
			},
		},
		{
			name: "HttpFullConfig",
			content: `
http:
    listen_address: "http://127.4.4.2:80"
    read_timeout: 60s
    write_timeout: 1m
    idle_timeout: 30000ms
`,
			expected: AppConfig{
				LogFormat: "text",
				HTTPConfig: HTTPConfig{
					ListenAddress: "http://127.4.4.2:80",
					ReadTimeout:   time.Minute,
					WriteTimeout:  time.Minute,
					IdleTimeout:   30 * time.Second,
				},
				OpenAI: defaultOpenAIConfig,
			},
		},
		{
			name: "OpenAIFullConfig",
			content: `
open_ai:
    api_key: "secret_api_key"
    base_url: "https://url.test"
    model: "test"
`,
			expected: AppConfig{
				LogFormat:  "text",
				HTTPConfig: defaultHTTPConfig,
				OpenAI: OpenAI{
					APIKey:  "secret_api_key",
					BaseURL: "https://url.test",
					Model:   "test",
				},
			},
		},
		{
			name: "AllPossibleErrors",
			content: `
log_format: yaml
http:
    listen_address: "http://127.4.4.2:80"
    read_timeout: -1s
    write_timeout: -1m
    idle_timeout: -1ms
`,
			wantErr: errors.New(
				`failed to validate app config:
- unknown log format: yaml
failed to validate HTTP configuration:
- read timeout should be grater than 0, got -1s
- write timeout should be grater than 0, got -1m0s
- idle timeout should be grater than 0, got -1ms`,
			),
		},
	}

	testFunc := func(t *testing.T, tc testCase) {
		t.Helper()

		tempFile, err := os.CreateTemp(t.TempDir(), "sdvg-config-*.yml")
		require.NoError(t, err)
		defer os.Remove(tempFile.Name())

		_, err = tempFile.WriteString(tc.content)
		if err != nil {
			t.Fatalf("failed to save content: %s", err)
		}

		err = tempFile.Close()
		if err != nil {
			t.Fatalf("failed to close file: %s", err)
		}

		var cfg AppConfig

		err = cfg.ParseFromFile(tempFile.Name())
		if tc.wantErr != nil {
			// unwrap error to exclude path to config file
			require.EqualError(t, errors.Unwrap(err), tc.wantErr.Error())
		} else {
			require.NoError(t, err)
			require.Equal(t, tc.expected, cfg)
		}
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) { testFunc(t, tc) })
	}
}

func ptr[T any](x T) *T {
	return &x
}

//nolint:maintidx
func TestGeneratorConfigYAMLParse(t *testing.T) {
	type testCase struct {
		name     string
		content  string
		wantErr  error
		expected GenerationConfig
	}

	defaultModel := &Model{
		Name:             "test",
		RowsCount:        1,
		RowsPerFile:      1,
		GenerateFrom:     0,
		GenerateToPtr:    nil,
		GenerateTo:       1,
		ModelDir:         "test",
		Columns:          make([]*Column, 0),
		PartitionColumns: make([]*PartitionColumn, 0),
	}

	firstFkUUIDReferencedColumn := &Column{
		Name:   "uuid",
		Type:   "uuid",
		Ranges: []*Params{{ColumnType: "uuid", RangePercentage: 1}},
	}

	secondFkIntFullReferencedColumn := &Column{
		Name: "int_full",
		Type: "integer",
		Ranges: []*Params{{
			ColumnType: "integer",
			IntegerParams: &ColumnIntegerParams{
				BitWidth: 16,
				FromPtr:  ptr(int64(10)),
				From:     10,
				ToPtr:    ptr(int64(1000)),
				To:       1000,
			},
			NullPercentage:     0.1,
			DistinctPercentage: 0.5,
			Ordered:            true,
			RangePercentage:    1,
		}},
	}

	defaultIntegerParams := &ColumnIntegerParams{
		BitWidth: 32,
		FromPtr:  nil,
		ToPtr:    nil,
		From:     math.MinInt32,
		To:       math.MaxInt32,
	}

	defaultFloatParams := &ColumnFloatParams{
		BitWidth: 32,
		FromPtr:  nil,
		ToPtr:    nil,
		From:     -math.MaxFloat32,
		To:       math.MaxFloat32,
	}

	defaultStringParams := &ColumnStringParams{
		MinLength: 1,
		MaxLength: 32,
		Locale:    "en",
	}

	defaultDateTimeParams := &ColumnDateTimeParams{
		From: time.Date(1900, 1, 1, 0, 0, 0, 0, time.UTC),
		To:   time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC),
	}

	testCases := []testCase{
		{
			name:    "EmptyConfig",
			content: "{}",
			wantErr: errors.New("failed to parse generator config: no model to generate"),
			expected: GenerationConfig{
				WorkersCount: runtime.NumCPU() * DefaultWorkersPerCPU,
				BatchSize:    1000,
				RandomSeed:   0,
				Models:       map[string]*Model{},
				OutputConfig: &OutputConfig{
					Type: "csv",
					CSVParams: &CSVConfig{
						FloatPrecision: 2,
						DatetimeFormat: "2006-01-02T15:04:05Z07:00",
						Delimiter:      ",",
					},
				},
			},
		},
		{
			name: "Models",
			content: `
workers_count: 4
batch_size: 10
random_seed: 1488
output:
  checkpoint_interval: 1s
  dir: test_output
models:
  test:
    rows_count: 123
    generate_from: 10
    generate_to: 15
    columns:
      - name: fk_1
        foreign_key: test.uuid
      - name: int_full
        type: integer
        type_params:
          bit_width: 16
          from: 10
          to: 1000
        null_percentage: 0.1
        distinct_percentage: 0.5
        ordered: true
      - name: int_default
        type: integer
      - name: int_enum
        type: integer
        values:
          - 222
          - null
          - 111
      - name: float_full
        type: float
        type_params:
          bit_width: 64
          from: 10.75
          to: 20.1
        ordered: true
      - name: float_default
        type: float
      - name: float_enum
        type: float
        values:
          - 0.007
          - null
          - 1.5
      - name: str_full
        type: string
        type_params:
          min_length: 5
          max_length: 10
          locale: "ru"
          logical_type: "last_name"
          without_large_letters: true
          without_small_letters: true
          without_numbers: true
          without_special_chars: true
        distinct_count: 10
      - name: str_template
        type: string
        type_params:
          template: "AA aa 00 ##"
      - name: str_default
        type: string
      - name: str_enum
        type: string
        values:
          - "test_2"
          - null
          - "test_1"
      - name: dt_full
        type: datetime
        type_params:
          from: 2000-01-01T00:00:00Z
          to: 2001-12-31T23:59:59.999999999Z
      - name: dt_default
        type: datetime
      - name: dt_enum
        type: datetime
        values:
          - 2001-12-31T23:59:59.999999999Z
          - null
          - 2000-01-01T00:00:00Z
      - name: uuid
        type: uuid
      - name: fk_2
        foreign_key: test.int_full
      - name: ranges
        type: integer
        ranges:
          - type_params:
              bit_width: 16
              from: 1
              to: 11
            distinct_count: 10
            ordered: true
          - type_params:
              bit_width: 32
              from: 100
              to: 1100
            distinct_percentage: 1
            null_percentage: 0.2
            ordered: true
            range_percentage: 0.5
          - values:
              - 999
              - null
            null_percentage: 0.2
      - name: ranges_default_filling_bug
        type: string
        ranges:
          - type_params: {}
          - type_params: {}
          - type_params: {}
          - type_params: {}
          - type_params: {}
          - type_params: {}
          - type_params: {}
`,
			expected: GenerationConfig{
				WorkersCount:   4,
				BatchSize:      10,
				RandomSeed:     1488,
				RealRandomSeed: 1488,
				Models: map[string]*Model{
					"test": {
						Name:          "test",
						RowsCount:     123,
						RowsPerFile:   123,
						GenerateFrom:  10,
						GenerateToPtr: ptr(uint64(15)),
						GenerateTo:    15,
						ModelDir:      "test",
						Columns: []*Column{
							{
								Name:             "fk_1",
								ForeignKey:       "test.uuid",
								Params:           &Params{},
								ForeignKeyColumn: firstFkUUIDReferencedColumn,
							},
							secondFkIntFullReferencedColumn,
							{
								Name: "int_default",
								Type: "integer",
								Ranges: []*Params{
									{
										ColumnType:      "integer",
										IntegerParams:   defaultIntegerParams,
										RangePercentage: 1,
									},
								},
							},
							{
								Name: "int_enum",
								Type: "integer",
								Ranges: []*Params{{
									ColumnType:      "integer",
									IntegerParams:   defaultIntegerParams,
									Values:          []any{nil, int32(111), int32(222)},
									RangePercentage: 1,
								}},
							},
							{
								Name: "float_full",
								Type: "float",
								Ranges: []*Params{{
									ColumnType: "float",
									FloatParams: &ColumnFloatParams{
										BitWidth: 64,
										FromPtr:  ptr(10.75),
										From:     10.75,
										ToPtr:    ptr(20.1),
										To:       20.1,
									},
									Ordered:         true,
									RangePercentage: 1,
								}},
							},
							{
								Name: "float_default",
								Type: "float",
								Ranges: []*Params{{
									ColumnType:      "float",
									FloatParams:     defaultFloatParams,
									RangePercentage: 1,
								}},
							},
							{
								Name: "float_enum",
								Type: "float",
								Ranges: []*Params{{
									ColumnType:      "float",
									FloatParams:     defaultFloatParams,
									Values:          []any{nil, float32(0.007), float32(1.5)},
									RangePercentage: 1,
								}},
							},
							{
								Name: "str_full",
								Type: "string",
								Ranges: []*Params{{
									ColumnType: "string",
									StringParams: &ColumnStringParams{
										MinLength:           5,
										MaxLength:           10,
										Locale:              "ru",
										LogicalType:         "last_name",
										WithoutLargeLetters: true,
										WithoutSmallLetters: true,
										WithoutNumbers:      true,
										WithoutSpecialChars: true,
									},
									DistinctCount:   10,
									RangePercentage: 1,
								}},
							},
							{
								Name: "str_template",
								Type: "string",
								Ranges: []*Params{{
									ColumnType: "string",
									StringParams: &ColumnStringParams{
										MinLength: 1,
										MaxLength: 32,
										Locale:    "en",
										Template:  "AA aa 00 ##",
									},
									RangePercentage: 1,
								}},
							},
							{
								Name: "str_default",
								Type: "string",
								Ranges: []*Params{{
									ColumnType:      "string",
									StringParams:    defaultStringParams,
									RangePercentage: 1,
								}},
							},
							{
								Name: "str_enum",
								Type: "string",
								Ranges: []*Params{{
									ColumnType:      "string",
									StringParams:    defaultStringParams,
									Values:          []any{nil, "test_1", "test_2"},
									RangePercentage: 1,
								},
								},
							},
							{
								Name: "dt_full",
								Type: "datetime",
								Ranges: []*Params{{
									ColumnType: "datetime",
									DateTimeParams: &ColumnDateTimeParams{
										From: time.Date(2000, 1, 1, 0, 0, 0, 0, time.UTC),
										To:   time.Date(2001, 12, 31, 23, 59, 59, 1e9-1, time.UTC),
									},
									RangePercentage: 1,
								}},
							},
							{
								Name: "dt_default",
								Type: "datetime",
								Ranges: []*Params{{
									ColumnType:      "datetime",
									DateTimeParams:  defaultDateTimeParams,
									RangePercentage: 1,
								}},
							},
							{
								Name: "dt_enum",
								Type: "datetime",
								Ranges: []*Params{{
									ColumnType:     "datetime",
									DateTimeParams: defaultDateTimeParams,
									Values: []any{
										nil,
										time.Date(2000, 1, 1, 0, 0, 0, 0, time.UTC),
										time.Date(2001, 12, 31, 23, 59, 59, 999999999, time.UTC),
									},
									RangePercentage: 1,
								}},
							},
							firstFkUUIDReferencedColumn,
							{
								Name:             "fk_2",
								Params:           &Params{},
								ForeignKey:       "test.int_full",
								ForeignKeyColumn: secondFkIntFullReferencedColumn,
							},
							{
								Name: "ranges",
								Type: "integer",
								Ranges: []*Params{
									{
										ColumnType: "integer",
										IntegerParams: &ColumnIntegerParams{
											BitWidth: 16,
											FromPtr:  ptr(int64(1)),
											From:     1,
											ToPtr:    ptr(int64(11)),
											To:       11,
										},
										DistinctCount:   10,
										Ordered:         true,
										RangePercentage: 0.25,
									},
									{
										ColumnType: "integer",
										IntegerParams: &ColumnIntegerParams{
											BitWidth: 32,
											FromPtr:  ptr(int64(100)),
											From:     100,
											ToPtr:    ptr(int64(1100)),
											To:       1100,
										},
										DistinctPercentage: 1,
										NullPercentage:     0.2,
										Ordered:            true,
										RangePercentage:    0.5,
									},
									{
										ColumnType:      "integer",
										IntegerParams:   defaultIntegerParams,
										Values:          []any{nil, int32(999)},
										RangePercentage: 0.25,
										NullPercentage:  0.2,
									},
								},
							},
							{
								Name: "ranges_default_filling_bug",
								Type: "string",
								Ranges: []*Params{
									{
										ColumnType:   "string",
										StringParams: defaultStringParams,
										// this value differs to value from last range
										RangePercentage: 0.14285714285714285,
									},
									{
										ColumnType:   "string",
										StringParams: defaultStringParams,
										// this value differs to value from last range
										RangePercentage: 0.14285714285714285,
									},
									{
										ColumnType:   "string",
										StringParams: defaultStringParams,
										// this value differs to value from last range
										RangePercentage: 0.14285714285714285,
									},
									{
										ColumnType:   "string",
										StringParams: defaultStringParams,
										// this value differs to value from last range
										RangePercentage: 0.14285714285714285,
									},
									{
										ColumnType:   "string",
										StringParams: defaultStringParams,
										// this value differs to value from last range
										RangePercentage: 0.14285714285714285,
									},
									{
										ColumnType:   "string",
										StringParams: defaultStringParams,
										// this value differs to value from last range
										RangePercentage: 0.14285714285714285,
									},
									{
										ColumnType:   "string",
										StringParams: defaultStringParams,
										// this last value differs from values of other ranges
										RangePercentage: 0.14285714285714302,
									},
								},
							},
						},
						PartitionColumns: make([]*PartitionColumn, 0),
					},
				},
				OutputConfig: &OutputConfig{
					Type:               "csv",
					Dir:                "test_output",
					CheckpointInterval: time.Second,
					CSVParams: &CSVConfig{
						FloatPrecision: 2,
						DatetimeFormat: "2006-01-02T15:04:05Z07:00",
						Delimiter:      ",",
					},
				},
			},
		},
		{
			name: "CsvFullConfig",
			content: `
random_seed: 1
output:
  type: csv
  params:
    datetime_format: "2006-01-02"
    float_precision: 1
    delimiter: ";"
models:
    test:
        rows_count: 1
`,
			expected: GenerationConfig{
				WorkersCount:   runtime.NumCPU() * DefaultWorkersPerCPU,
				BatchSize:      1000,
				RandomSeed:     1,
				RealRandomSeed: 1,
				Models:         map[string]*Model{"test": defaultModel},
				OutputConfig: &OutputConfig{
					Type:               "csv",
					Dir:                DefaultOutputDir,
					CheckpointInterval: 5 * time.Second,
					CSVParams: &CSVConfig{
						FloatPrecision: 1,
						DatetimeFormat: "2006-01-02",
						Delimiter:      ";",
					},
				},
			},
		},
		{
			name: "DevNullConfig",
			content: `
output:
  type: devnull
random_seed: 1
models:
    test:
        rows_count: 1
`,
			expected: GenerationConfig{
				WorkersCount:   runtime.NumCPU() * DefaultWorkersPerCPU,
				BatchSize:      1000,
				RandomSeed:     1,
				RealRandomSeed: 1,
				Models:         map[string]*Model{"test": defaultModel},
				OutputConfig: &OutputConfig{
					Type:               "devnull",
					Dir:                DefaultOutputDir,
					CheckpointInterval: 5 * time.Second,
					DevNullParams:      &DevNullConfig{},
				},
			},
		},
		{
			name: "HTTPFullConfig",
			content: `
random_seed: 1
output:
  type: http
  params:
    endpoint: http://localhost:8080
    timeout: 10s
    batch_size: 100
    workers_count: 5
    headers:
      test_header: 1
    format_template: "{ table_name: {{ .Table }}, rows_count: {{ len .Rows }}, rows: {{ .RowsJson }} }"
models:
  test:
    rows_count: 1
`,
			expected: GenerationConfig{
				WorkersCount:   runtime.NumCPU() * DefaultWorkersPerCPU,
				BatchSize:      1000,
				RandomSeed:     1,
				RealRandomSeed: 1,
				Models:         map[string]*Model{"test": defaultModel},
				OutputConfig: &OutputConfig{
					Type:               "http",
					Dir:                DefaultOutputDir,
					CheckpointInterval: 5 * time.Second,
					HTTPParams: &HTTPParams{
						Endpoint:     "http://localhost:8080",
						Timeout:      10 * time.Second,
						BatchSize:    100,
						WorkersCount: 5,
						Headers: map[string]string{
							"test_header": "1",
						},
						FormatTemplate: "{ table_name: {{ .Table }}, rows_count: {{ len .Rows }}, rows: {{ .RowsJson }} }",
					},
				},
			},
		},
		{
			name: "HTTPDefaultConfig",
			content: `
random_seed: 1
output:
  type: http
  params:
    endpoint: http://localhost:8080
models:
    test:
        rows_count: 1
`,
			expected: GenerationConfig{
				WorkersCount:   runtime.NumCPU() * DefaultWorkersPerCPU,
				BatchSize:      1000,
				RandomSeed:     1,
				RealRandomSeed: 1,
				Models:         map[string]*Model{"test": defaultModel},
				OutputConfig: &OutputConfig{
					Type:               "http",
					Dir:                DefaultOutputDir,
					CheckpointInterval: 5 * time.Second,
					HTTPParams: &HTTPParams{
						Endpoint:       "http://localhost:8080",
						Timeout:        time.Minute,
						BatchSize:      1000,
						WorkersCount:   1,
						Headers:        map[string]string{},
						FormatTemplate: defaultFormatTemplate,
					},
				},
			},
		},
		{
			name: "TCSFullConfig",
			content: `
random_seed: 1
output:
  type: tcs
  params:
    endpoint: http://tcs.ru:9000
    timeout: 30s
    batch_size: 10
    workers_count: 10
    format_template: "{{ .Table }}"
models:
    test:
        rows_count: 1
`,
			expected: GenerationConfig{
				WorkersCount:   runtime.NumCPU() * DefaultWorkersPerCPU,
				BatchSize:      1000,
				RandomSeed:     1,
				RealRandomSeed: 1,
				Models:         map[string]*Model{"test": defaultModel},
				OutputConfig: &OutputConfig{
					Type:               "tcs",
					Dir:                DefaultOutputDir,
					CheckpointInterval: 5 * time.Second,
					TCSParams: &TCSConfig{
						HTTPParams: HTTPParams{
							Endpoint:     "http://tcs.ru:9000",
							Timeout:      30 * time.Second,
							BatchSize:    10,
							WorkersCount: 10,
							Headers: map[string]string{
								tcsTimeoutHeader: "30000",
							},
							FormatTemplate: defaultFormatTemplate,
						},
					},
				},
			},
		},
		{
			name: "TCSDefaultConfig",
			content: `
random_seed: 1
output:
  type: tcs
  params:
    endpoint: tcs
models:
  test:
    rows_count: 1
`,
			expected: GenerationConfig{
				WorkersCount:   runtime.NumCPU() * DefaultWorkersPerCPU,
				BatchSize:      1000,
				RandomSeed:     1,
				RealRandomSeed: 1,
				Models:         map[string]*Model{"test": defaultModel},
				OutputConfig: &OutputConfig{
					Type:               "tcs",
					Dir:                DefaultOutputDir,
					CheckpointInterval: 5 * time.Second,
					TCSParams: &TCSConfig{
						HTTPParams: HTTPParams{
							Endpoint:     "tcs",
							Timeout:      time.Minute,
							BatchSize:    1000,
							WorkersCount: 1,
							Headers: map[string]string{
								tcsTimeoutHeader: "60000",
							},
							FormatTemplate: defaultFormatTemplate,
						},
					},
				},
			},
		},
		{
			name: "Parquet full config",
			content: `
random_seed: 1
output:
  type: parquet
  params:
    datetime_format: micros
    float_precision: 3
    compression_codec: GZIP
models:
    test:
        rows_count: 1
`,
			expected: GenerationConfig{
				WorkersCount:   runtime.NumCPU() * DefaultWorkersPerCPU,
				BatchSize:      1000,
				RandomSeed:     1,
				RealRandomSeed: 1,
				Models:         map[string]*Model{"test": defaultModel},
				OutputConfig: &OutputConfig{
					Type:               "parquet",
					Dir:                DefaultOutputDir,
					CheckpointInterval: 5 * time.Second,
					ParquetParams: &ParquetConfig{
						FloatPrecision:   3,
						DateTimeFormat:   ParquetDateTimeMicrosFormat,
						CompressionCodec: "GZIP",
					},
				},
			},
		},
		{
			name: "Parquet default",
			content: `
random_seed: 1
output:
  type: parquet
models:
    test:
        rows_count: 1
`,
			expected: GenerationConfig{
				WorkersCount:   runtime.NumCPU() * DefaultWorkersPerCPU,
				BatchSize:      1000,
				RandomSeed:     1,
				RealRandomSeed: 1,
				Models:         map[string]*Model{"test": defaultModel},
				OutputConfig: &OutputConfig{
					Type:               "parquet",
					Dir:                DefaultOutputDir,
					CheckpointInterval: 5 * time.Second,
					ParquetParams: &ParquetConfig{
						FloatPrecision:   2,
						DateTimeFormat:   ParquetDateTimeMillisFormat,
						CompressionCodec: "UNCOMPRESSED",
					},
				},
			},
		},
		{
			name: "Parquet column with default vars",
			content: `
random_seed: 1
output:
  type: parquet
models:
  user:
    rows_count: 100
    rows_per_file: 50
    columns:
      - name: id
        type: integer
        parquet:
          encoding: RLE_DICTIONARY
`,
			expected: GenerationConfig{
				WorkersCount:   runtime.NumCPU() * DefaultWorkersPerCPU,
				BatchSize:      1000,
				RandomSeed:     1,
				RealRandomSeed: 1,
				Models: map[string]*Model{
					"user": {
						Name:          "user",
						RowsCount:     100,
						RowsPerFile:   50,
						GenerateToPtr: nil,
						GenerateTo:    100,
						ModelDir:      "user",
						Columns: []*Column{
							{
								Name: "id",
								Type: "integer",
								Ranges: []*Params{{
									ColumnType: "integer",
									IntegerParams: &ColumnIntegerParams{
										BitWidth: 32,
										FromPtr:  nil,
										From:     math.MinInt32,
										ToPtr:    nil,
										To:       math.MaxInt32,
									},
									RangePercentage: 1,
								}},
								ParquetParams: &ColumnParquetParams{
									Encoding: "RLE_DICTIONARY",
								},
							},
						},
						PartitionColumns: make([]*PartitionColumn, 0),
					},
				},
				OutputConfig: &OutputConfig{
					Type:               "parquet",
					Dir:                DefaultOutputDir,
					CheckpointInterval: 5 * time.Second,
					ParquetParams: &ParquetConfig{
						FloatPrecision:   2,
						DateTimeFormat:   ParquetDateTimeMillisFormat,
						CompressionCodec: "UNCOMPRESSED",
					},
				},
			},
		},
		{
			name: "Parquet with partitions",
			content: `
random_seed: 1
output:
 type: parquet
models:
  user:
    rows_count: 100
    columns:
      - name: uuid_field
        type: uuid
      - name: float_field
        type: float
      - name: int_field
        type: integer
    partition_columns:
        - name: float_field
          write_to_output: false
        - name: uuid_field
          write_to_output: true
`,
			expected: GenerationConfig{
				WorkersCount:   runtime.NumCPU() * DefaultWorkersPerCPU,
				BatchSize:      1000,
				RandomSeed:     1,
				RealRandomSeed: 1,
				Models: map[string]*Model{
					"user": {
						Name:          "user",
						RowsCount:     100,
						RowsPerFile:   100,
						GenerateToPtr: nil,
						GenerateTo:    100,
						ModelDir:      "user",
						Columns: []*Column{
							{
								Name: "uuid_field",
								Type: "uuid",
								Ranges: []*Params{
									{
										ColumnType:      "uuid",
										RangePercentage: 1,
									},
								},
							},
							{
								Name: "int_field",
								Type: "integer",
								Ranges: []*Params{
									{
										ColumnType: "integer",
										IntegerParams: &ColumnIntegerParams{
											BitWidth: 32,
											FromPtr:  nil,
											From:     math.MinInt32,
											ToPtr:    nil,
											To:       math.MaxInt32,
										},
										RangePercentage: 1,
									},
								},
							},
							// NOT WRITEABLE PARTITION COLUMNS MUST BE AT THE END
							{
								Name: "float_field",
								Type: "float",
								Ranges: []*Params{
									{
										ColumnType: "float",
										FloatParams: &ColumnFloatParams{
											BitWidth: 32,
											FromPtr:  nil,
											From:     -math.MaxFloat32,
											ToPtr:    nil,
											To:       math.MaxFloat32,
										},
										RangePercentage: 1,
									},
								},
							},
						},
						PartitionColumns: []*PartitionColumn{
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
				},
				OutputConfig: &OutputConfig{
					Type:               "parquet",
					Dir:                DefaultOutputDir,
					CheckpointInterval: 5 * time.Second,
					ParquetParams: &ParquetConfig{
						FloatPrecision:   2,
						DateTimeFormat:   ParquetDateTimeMillisFormat,
						CompressionCodec: "UNCOMPRESSED",
					},
				},
			},
		},
		{
			name: "Partitions by non-existent columns",
			content: `
random_seed: 1
output:
 type: parquet
models:
  user:
    rows_count: 100
    columns:
      - name: uuid_field
        type: uuid
      - name: float_field
        type: float
    partition_columns:
        - name: float_field
          write_to_output: false
        - name: non_existing
          write_to_output: true
`,
			wantErr: errors.New(
				`failed to validate generator config:
models[user]:
- partition_columns[non_existing] does not exist`,
			),
		},
		{
			name: "Errors list",
			content: `
workers_count: -1
batch_size: 10
random_seed: 1488
output:
  type: parquet
  params:
    compression_codec: non-existent-codec
    float_precision: -1
    datetime_format: non-existent-datetime-format
  checkpoint_interval: -1s
models_to_ignore:
   - non-existent-column
models:
  first:
    rows_count: 1
    generate_from: 3
    generate_to: 2
    columns:
      - name: unknown_type
        type: unknown_type
      - name: invalid_type_params
        type: integer
        type_params:
          bit_width: 10
      - name: invalid_range_percentage
        type: string
        ranges:
          - type_params:
            range_percentage: 2
    partition_columns:
      - write_to_output: false
`,
			wantErr: errors.New(
				`failed to validate generator config:
- workers count should be grater than 0, got -1
models[first]:
- generate_from must be less than or equal to rows_count: 3
- generate_to must be less or equal to rows_count: 2
- generate_from must be less or equal to generate_to: 3
columns[unknown_type]:
- unknown type "unknown_type"
columns[invalid_type_params]:
ranges[0]:
integer params:
- unsupported integer bit width: 10
columns[invalid_range_percentage]:
ranges[0]:
- range percentage should be between 0 and 1, got 2
- invalid range percentage should be between 0 and 1: got 2
- sum of range percentages should be between 0 and 1: got 2
- partition_columns[] does not exist
partition_columns[]:
- name for partition column is required
- unknown model to ignore "non-existent-column"
- all models are marked as ignored
output config:
- checkpoint_interval must be greater than zero, got -1s
parquet params:
- unknown compression codec non-existent-codec, supported [UNCOMPRESSED SNAPPY GZIP LZ4 LZ4RAW LZO ZSTD BROTLI]
- float precision should be grater than 0, got -1
- unknown datetime format non-existent-datetime-format, supported [millis micros]`,
			),
		},
	}

	testFunc := func(t *testing.T, tc testCase) {
		t.Helper()

		tempFile, err := os.CreateTemp(t.TempDir(), "sdvg-config-*.yml")
		require.NoError(t, err)
		defer os.Remove(tempFile.Name())

		_, err = tempFile.WriteString(tc.content)
		if err != nil {
			t.Fatalf("failed to save content: %s", err)
		}

		err = tempFile.Close()
		if err != nil {
			t.Fatalf("failed to close file: %s", err)
		}

		var cfg GenerationConfig

		err = cfg.ParseFromFile(tempFile.Name())

		if tc.wantErr != nil {
			require.EqualError(t, err, tc.wantErr.Error())

			return
		}

		// skip output params map check
		tc.expected.OutputConfig.Params = cfg.OutputConfig.Params

		for modelName := range tc.expected.Models {
			expectedModel := tc.expected.Models[modelName]
			gotModel := cfg.Models[modelName]

			// skip ColumnsTopologicalOrder check
			expectedModel.ColumnsTopologicalOrder = gotModel.ColumnsTopologicalOrder

			for columnName := range expectedModel.Columns {
				expectedColumn := expectedModel.Columns[columnName]
				gotColumn := gotModel.Columns[columnName]

				// skip type params map check
				for idx, expectedRange := range expectedColumn.Ranges {
					expectedRange.TypeParams = gotColumn.Ranges[idx].TypeParams
				}
			}
		}

		require.Equal(t, tc.expected, cfg)
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) { testFunc(t, tc) })
	}
}
