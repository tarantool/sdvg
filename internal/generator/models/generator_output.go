package models

import (
	"net/url"
	"slices"
	"strconv"
	"time"
	"unicode/utf8"

	"github.com/pkg/errors"
	"github.com/tarantool/sdvg/internal/generator/common"
)

const (
	DefaultOutputDir            = "output"
	DefaultOutputType           = "csv"
	defaultFormatTemplate       = `{ "table_name": "{{ .ModelName }}", "rows": {{ json .Rows }} }`
	tcsTimeoutHeader            = "x-tcs-timeout_ms"
	ParquetDateTimeMillisFormat = "millis"
	ParquetDateTimeMicrosFormat = "micros"

	PartitionFilesLimitDefault = 1000
)

// DataRow type is used to represent any data row that was generated.
type DataRow struct {
	Values []any
}

type OutputConfig struct {
	Type               string         `backup:"true"              json:"type"                yaml:"type"`
	Dir                string         `backup:"true"              json:"dir"                 yaml:"dir"`
	CheckpointInterval time.Duration  `json:"checkpoint_interval" yaml:"checkpoint_interval"`
	CreateModelDir     bool           `backup:"true"              json:"create_model_dir"    yaml:"create_model_dir"`
	Params             any            `backup:"true"              json:"params"              yaml:"params"`
	DevNullParams      *DevNullConfig `json:"-"                   yaml:"-"`
	CSVParams          *CSVConfig     `json:"-"                   yaml:"-"`
	HTTPParams         *HTTPParams    `json:"-"                   yaml:"-"`
	TCSParams          *TCSConfig     `json:"-"                   yaml:"-"`
	ParquetParams      *ParquetConfig `json:"-"                   yaml:"-"`
}

//nolint:cyclop
func (c *OutputConfig) Parse() error {
	var err error

	switch c.Type {
	case "csv", "":
		c.CSVParams, err = common.AnyToStruct[CSVConfig](c.Params)
	case "devnull":
		c.DevNullParams, err = common.AnyToStruct[DevNullConfig](c.Params)
	case "http":
		c.HTTPParams, err = common.AnyToStruct[HTTPParams](c.Params)
	case "tcs":
		c.TCSParams, err = common.AnyToStruct[TCSConfig](c.Params)
	case "parquet":
		c.ParquetParams, err = common.AnyToStruct[ParquetConfig](c.Params)
	}

	if err != nil {
		return errors.WithMessagef(err, "%q output params", c.Type)
	}

	if err = FieldParse(c.CSVParams); err != nil {
		return errors.WithMessage(err, "csv params")
	}

	if err = FieldParse(c.DevNullParams); err != nil {
		return errors.WithMessage(err, "devnull params")
	}

	if err = FieldParse(c.HTTPParams); err != nil {
		return errors.WithMessage(err, "http params")
	}

	if err = FieldParse(c.TCSParams); err != nil {
		return errors.WithMessage(err, "tcs params")
	}

	if err = FieldParse(c.ParquetParams); err != nil {
		return errors.WithMessage(err, "parquet params")
	}

	return nil
}

func (c *OutputConfig) FillDefaults() {
	if c.Type == "" {
		c.Type = DefaultOutputType
	}

	if c.Dir == "" {
		c.Dir = DefaultOutputDir
	}

	if c.CheckpointInterval == 0 {
		c.CheckpointInterval = 5 * time.Second //nolint:mnd
	}

	FieldFillDefaults(c.CSVParams)

	FieldFillDefaults(c.DevNullParams)

	FieldFillDefaults(c.HTTPParams)

	FieldFillDefaults(c.TCSParams)

	FieldFillDefaults(c.ParquetParams)
}

var OutputTypes = []string{"csv", "devnull", "http", "tcs", "parquet"}
var DiskFilesOutputTypes = []string{"csv", "parquet"} // output types that actually create files on disk

func (c *OutputConfig) Validate() []error {
	var errs []error

	if !slices.Contains(OutputTypes, c.Type) {
		errs = append(errs, errors.Errorf("unknown output type: %s", c.Type))
	}

	if c.CheckpointInterval < 0 {
		errs = append(errs, errors.Errorf("checkpoint_interval must be greater than zero, got %v", c.CheckpointInterval))
	}

	if csvParamsErrs := FieldValidate(c.CSVParams); len(csvParamsErrs) != 0 {
		errs = append(errs, errors.New("csv params:"))
		errs = append(errs, csvParamsErrs...)
	}

	if devNullParamsErrs := FieldValidate(c.DevNullParams); len(devNullParamsErrs) != 0 {
		errs = append(errs, errors.New("devnull params:"))
		errs = append(errs, devNullParamsErrs...)
	}

	if httpParamsErrs := FieldValidate(c.HTTPParams); len(httpParamsErrs) != 0 {
		errs = append(errs, errors.New("http params:"))
		errs = append(errs, httpParamsErrs...)
	}

	if TCSParamsErrs := FieldValidate(c.TCSParams); len(TCSParamsErrs) != 0 {
		errs = append(errs, errors.New("TCS params:"))
		errs = append(errs, TCSParamsErrs...)
	}

	if parquetParamsErrs := FieldValidate(c.ParquetParams); len(parquetParamsErrs) != 0 {
		errs = append(errs, errors.New("parquet params:"))
		errs = append(errs, parquetParamsErrs...)
	}

	return errs
}

// Verify interface compliance in compile time.
var _ Field = (*DevNullConfig)(nil)

// DevNullConfig type used to describe output config for devnull implementation.
type DevNullConfig struct {
	Handler func(row *DataRow, modelName string) error `json:"-" yaml:"-"`
}

func (c *DevNullConfig) Parse() error { return nil }

func (c *DevNullConfig) FillDefaults() {}

func (c *DevNullConfig) Validate() []error { return nil }

// Verify interface compliance in compile time.
var _ Field = (*CSVConfig)(nil)

// CSVConfig type used to describe output config for CSV implementation.
type CSVConfig struct {
	FloatPrecision      int    `json:"float_precision" yaml:"float_precision"`
	DatetimeFormat      string `json:"datetime_format" yaml:"datetime_format"`
	Delimiter           string `backup:"true"          json:"delimiter"       yaml:"delimiter"`
	WithoutHeaders      bool   `backup:"true"          json:"without_headers" yaml:"without_headers"`
	PartitionFilesLimit *int   `json:"partition_files_limit" yaml:"partition_files_limit"`
}

func (c *CSVConfig) Parse() error { return nil }

func (c *CSVConfig) FillDefaults() {
	if c.FloatPrecision == 0 {
		c.FloatPrecision = 2
	}

	if c.DatetimeFormat == "" {
		c.DatetimeFormat = "2006-01-02T15:04:05Z07:00"
	}

	if c.Delimiter == "" {
		c.Delimiter = ","
	}

	if c.PartitionFilesLimit == nil {
		c.PartitionFilesLimit = new(int)
		*c.PartitionFilesLimit = 1000
	}
}

func (c *CSVConfig) Validate() []error {
	var errs []error

	if c.FloatPrecision < 0 {
		errs = append(errs, errors.Errorf("float precision should be grater than 0, got %v", c.FloatPrecision))
	}

	if utf8.RuneCountInString(c.Delimiter) != 1 {
		errs = append(errs, errors.Errorf("the delimiter must consist of one character, got %v", c.Delimiter))
	}

	if c.PartitionFilesLimit != nil && *c.PartitionFilesLimit <= 0 {
		errs = append(errs, errors.Errorf("partition files limit should be greater than 0, got: %v", *c.PartitionFilesLimit))
	}

	return errs
}

var _ Field = (*HTTPParams)(nil)

type HTTPParams struct {
	Endpoint       string            `json:"endpoint"        yaml:"endpoint"`
	Timeout        time.Duration     `json:"timeout"         yaml:"timeout"`
	BatchSize      int               `json:"batch_size"      yaml:"batch_size"`
	WorkersCount   int               `json:"workers_count"   yaml:"workers_count"`
	Headers        map[string]string `json:"headers"         yaml:"headers"`
	FormatTemplate string            `json:"format_template" yaml:"format_template"`
}

func (c *HTTPParams) Parse() error { return nil }

func (c *HTTPParams) FillDefaults() {
	if c.Timeout == 0 {
		c.Timeout = time.Minute
	}

	if c.BatchSize == 0 {
		c.BatchSize = 1000
	}

	if c.WorkersCount == 0 {
		c.WorkersCount = 1
	}

	if c.Headers == nil {
		c.Headers = make(map[string]string)
	}

	if c.FormatTemplate == "" {
		c.FormatTemplate = defaultFormatTemplate
	}
}

func (c *HTTPParams) Validate() []error {
	var errs []error

	if _, err := url.Parse(c.Endpoint); err != nil {
		errs = append(errs, errors.New(err.Error()))
	}

	if c.Timeout < 0 {
		errs = append(errs, errors.Errorf("timeout should be grater or equals to 0, got %v", c.Timeout))
	}

	if c.BatchSize <= 0 {
		errs = append(errs, errors.Errorf("batch size should be grater than 0, got %v", c.BatchSize))
	}

	if c.WorkersCount <= 0 {
		errs = append(errs, errors.Errorf("workers count should be grater than 0, got %v", c.WorkersCount))
	}

	return errs
}

// Verify interface compliance in compile time.
var _ Field = (*TCSConfig)(nil)

// TCSConfig type used to describe output config for TCS implementation.
type TCSConfig struct {
	HTTPParams `json:",inline" yaml:",inline"`
}

func (c *TCSConfig) Parse() error { return nil }

func (c *TCSConfig) FillDefaults() {
	c.HTTPParams.FillDefaults()
	c.FormatTemplate = defaultFormatTemplate

	_, ok := c.Headers[tcsTimeoutHeader]
	if !ok {
		c.Headers[tcsTimeoutHeader] = strconv.FormatInt(c.Timeout.Milliseconds(), 10)
	}
}

func (c *TCSConfig) Validate() []error {
	errs := c.HTTPParams.Validate()

	if _, ok := c.Headers[tcsTimeoutHeader]; !ok {
		errs = append(errs, errors.New("tcs timeout header must be specified"))
	}

	return errs
}

// Verify interface compliance in compile time.
var _ Field = (*ParquetConfig)(nil)

// ParquetConfig type used to describe output config for parquet implementation.
type ParquetConfig struct {
	CompressionCodec    string `backup:"true"          json:"compression_codec" yaml:"compression_codec"`
	FloatPrecision      int    `json:"float_precision" yaml:"float_precision"`
	DateTimeFormat      string `json:"datetime_format" yaml:"datetime_format"`
	PartitionFilesLimit *int   `json:"partition_files_limit" yaml:"partition_files_limit"`
}

//nolint:lll
var parquetSupportedCompressionCodecs = []string{"UNCOMPRESSED", "SNAPPY", "GZIP", "LZ4", "LZ4RAW", "LZO", "ZSTD", "BROTLI"}
var parquetSupportedDateTimeFormats = []string{ParquetDateTimeMillisFormat, ParquetDateTimeMicrosFormat}

func (c *ParquetConfig) Parse() error { return nil }

func (c *ParquetConfig) FillDefaults() {
	if c.CompressionCodec == "" {
		c.CompressionCodec = "UNCOMPRESSED"
	}

	if c.FloatPrecision == 0 {
		c.FloatPrecision = 2
	}

	if c.DateTimeFormat == "" {
		c.DateTimeFormat = ParquetDateTimeMillisFormat
	}

	if c.PartitionFilesLimit == nil {
		c.PartitionFilesLimit = new(int)
		*c.PartitionFilesLimit = 1000
	}
}

func (c *ParquetConfig) Validate() []error {
	var errs []error

	if !slices.Contains(parquetSupportedCompressionCodecs, c.CompressionCodec) {
		errs = append(errs, errors.Errorf("unknown compression codec %v, supported %v",
			c.CompressionCodec, parquetSupportedCompressionCodecs))
	}

	if c.FloatPrecision < 0 {
		errs = append(errs, errors.Errorf("float precision should be grater than 0, got %v", c.FloatPrecision))
	}

	if !slices.Contains(parquetSupportedDateTimeFormats, c.DateTimeFormat) {
		errs = append(errs, errors.Errorf("unknown datetime format %v, supported %v",
			c.DateTimeFormat, parquetSupportedDateTimeFormats))
	}

	if c.PartitionFilesLimit != nil && *c.PartitionFilesLimit <= 0 {
		errs = append(errs, errors.Errorf("partition files limit should be greater than 0, got: %v", *c.PartitionFilesLimit))
	}

	return errs
}
