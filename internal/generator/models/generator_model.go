package models

import (
	"fmt"
	"math"
	"reflect"
	"slices"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/pkg/errors"
	"github.com/tarantool/sdvg/internal/generator/common"
)

const (
	FirstNameType = "first_name"
	LastNameType  = "last_name"
	PhoneType     = "phone"
	TextType      = "text"
)

// Model type is used to describe model of generated data.
type Model struct {
	Name          string
	RowsCount     uint64  `backup:"true"      json:"rows_count"    yaml:"rows_count"`
	GenerateFrom  uint64  `backup:"true"      json:"generate_from" yaml:"generate_from"`
	GenerateToPtr *uint64 `json:"generate_to" yaml:"generate_to"`
	GenerateTo    uint64  `json:"-"           yaml:"-"`
	RowsPerFile   uint64  `backup:"true"      json:"rows_per_file" yaml:"rows_per_file"`
	ModelDir      string  `backup:"true"      json:"model_dir"     yaml:"model_dir"`
	// The columns from the partitioning key with PartitionColumn.WriteToOutput == false, must be at the end of slice.
	Columns          []*Column          `backup:"true" json:"columns"           yaml:"columns"`
	PartitionColumns []*PartitionColumn `backup:"true" json:"partition_columns" yaml:"partition_columns"`
}

// PartitionColumn type is used to describe partition parameters for column.
type PartitionColumn struct {
	Name          string `backup:"true" json:"name"            yaml:"name"`
	WriteToOutput bool   `backup:"true" json:"write_to_output" yaml:"write_to_output"`
}

func (pc *PartitionColumn) FillDefaults() {}

func (pc *PartitionColumn) Validate() []error {
	if pc.Name == "" {
		return []error{errors.Errorf("name for partition column is required")}
	}

	return nil
}

func (m *Model) Parse() error {
	if m.GenerateToPtr != nil {
		m.GenerateTo = *m.GenerateToPtr
	}

	if m.Columns == nil {
		m.Columns = make([]*Column, 0)
	}

	if m.PartitionColumns == nil {
		m.PartitionColumns = make([]*PartitionColumn, 0)
	}

	for _, column := range m.Columns {
		err := column.Parse()
		if err != nil {
			return errors.WithMessagef(err, "columns[%s]", column.Name)
		}
	}

	nonWriteableColumns := make([]string, 0, len(m.PartitionColumns))

	for _, column := range m.PartitionColumns {
		if !column.WriteToOutput {
			nonWriteableColumns = append(nonWriteableColumns, column.Name)
		}
	}

	m.shiftColumnsToEnd(nonWriteableColumns)

	return nil
}

func (m *Model) FillDefaults() {
	if m.RowsPerFile == 0 {
		m.RowsPerFile = m.RowsCount
	}

	if m.GenerateToPtr == nil {
		m.GenerateTo = m.RowsCount
	}

	if m.ModelDir == "" {
		m.ModelDir = m.Name
	}

	for _, column := range m.Columns {
		column.FillDefaults()
	}

	for _, column := range m.PartitionColumns {
		column.FillDefaults()
	}
}

//nolint:cyclop
func (m *Model) Validate() []error {
	var errs []error

	if m.RowsCount <= 0 {
		errs = append(errs, errors.Errorf("rows_count must be greater than zero: %v", m.RowsCount))
	}

	if m.GenerateFrom > m.RowsCount {
		errs = append(errs, errors.Errorf("generate_from must be less than or equal to rows_count: %v", m.GenerateFrom))
	}

	if m.GenerateTo > m.RowsCount {
		errs = append(errs, errors.Errorf("generate_to must be less or equal to rows_count: %v", m.GenerateTo))
	}

	if m.GenerateFrom > m.GenerateTo {
		errs = append(errs, errors.Errorf("generate_from must be less or equal to generate_to: %v", m.GenerateFrom))
	}

	columnsMap := make(map[string]struct{})

	for _, column := range m.Columns {
		if _, ok := columnsMap[column.Name]; ok {
			errs = append(errs, errors.Errorf("forbidden to have columns with same name %q", column.Name))
		}

		columnsMap[column.Name] = struct{}{}

		if columnErrs := column.Validate(); len(columnErrs) != 0 {
			errs = append(errs, errors.Errorf("columns[%s]:", column.Name))
			errs = append(errs, columnErrs...)
		}
	}

	for _, partitionColumn := range m.PartitionColumns {
		if _, ok := columnsMap[partitionColumn.Name]; !ok {
			errs = append(errs, errors.Errorf("partition_columns[%s] does not exist", partitionColumn.Name))
		}

		if partitionColumnErrs := partitionColumn.Validate(); len(partitionColumnErrs) != 0 {
			errs = append(errs, errors.Errorf("partition_columns[%s]:", partitionColumn.Name))
			errs = append(errs, partitionColumnErrs...)
		}
	}

	return errs
}

// shiftColumnsToEnd - rearranges the columns in the scheme
// by shifting the columns with columnsNames to the end in same order.
func (m *Model) shiftColumnsToEnd(columnsNames []string) {
	m.Columns = common.ShiftElementsToEnd(
		m.Columns,
		columnsNames,
		func(column *Column) string {
			return column.Name
		},
	)
}

// Column type is used to describe column of one structure.
type Column struct {
	Name             string               `backup:"true"  json:"name"              yaml:"name"`
	Type             string               `backup:"true"  json:"type"              yaml:"type"`
	Params           *Params              `json:",inline" yaml:",inline"` // it moved to Ranges after parsing
	Ranges           []*Params            `backup:"true"  json:"ranges"            yaml:"ranges"`
	ForeignKey       string               `backup:"true"  json:"foreign_key"       yaml:"foreign_key"`
	ForeignKeyColumn *Column              `json:"-"       yaml:"-"`
	ForeignKeyOrder  bool                 `backup:"true"  json:"foreign_key_order" yaml:"foreign_key_order"`
	ParquetParams    *ColumnParquetParams `backup:"true"  json:"parquet"           yaml:"parquet"`
}

func (c *Column) String() string {
	var rangesStr string
	for i, r := range c.Ranges {
		//nolint:lll
		rangesStr += fmt.Sprintf(
			"\n  Range[%d]: {Values:%+v, TypeParams:%+v, IntegerParams:%+v, FloatParams:%+v, StringParams:%+v, DateTimeParams:%+v, NullPercentage:%+v, DistinctPercentage:%+v, DistinctCount:%+v, RangePercentage:%+v, Ordered:%+v}",
			i, r.Values, r.TypeParams, r.IntegerParams, r.FloatParams, r.StringParams, r.DateTimeParams, r.NullPercentage, r.DistinctPercentage, r.DistinctCount, r.RangePercentage, r.Ordered,
		)
	}

	return fmt.Sprintf(
		"&{Name:%+v, Type:%+v, ForeignKey:%+v, ForeignKeyColumn:%+v, Params:%+v, ParquetParams:%+v, Ranges:%s}",
		c.Name, c.Type, c.ForeignKey, c.ForeignKeyColumn, c.Params, c.ParquetParams, rangesStr,
	)
}

func (c *Column) Parse() error {
	if c.Params == nil && c.Ranges == nil {
		c.Params = &Params{}
	}

	if c.ForeignKey != "" {
		return nil
	}

	if c.Params != nil {
		if c.Ranges != nil {
			return errors.New("forbidden to set both global type params and ranges")
		}

		c.Ranges = append(c.Ranges, c.Params)
		c.Params = nil
	}

	for i, r := range c.Ranges {
		r.ColumnType = c.Type
		if err := r.Parse(); err != nil {
			return errors.WithMessagef(err, "ranges[%d]", i)
		}
	}

	if err := FieldParse(c.ParquetParams); err != nil {
		return errors.WithMessage(err, "parquet params")
	}

	return nil
}

func (c *Column) FillDefaults() {
	var (
		rangePercentageSum      float64
		rangesWithOutPercentage int
	)

	for _, r := range c.Ranges {
		r.FillDefaults()

		if r.RangePercentage > 0 {
			rangePercentageSum += r.RangePercentage
		} else {
			rangesWithOutPercentage++
		}
	}

	if rangesWithOutPercentage > 0 {
		avgRangePercentage := (1 - rangePercentageSum) / float64(rangesWithOutPercentage)

		for i, r := range c.Ranges {
			if r.RangePercentage == 0 {
				if i == len(c.Ranges)-1 {
					r.RangePercentage = 1 - rangePercentageSum
				} else {
					r.RangePercentage = avgRangePercentage
					rangePercentageSum += avgRangePercentage
				}
			}
		}
	}

	FieldFillDefaults(c.ParquetParams)
}

func (c *Column) Validate() []error {
	var (
		rangePercentageSum float64
		errs               []error
	)

	if c.ForeignKey != "" {
		if common.Any(
			c.Type != "",
			c.Ranges != nil,
			c.ParquetParams != nil,
		) {
			errs = append(errs, errors.New("forbidden to use foreign key with any of other params"))
		}

		return errs
	}

	if !slices.Contains([]string{"integer", "float", "string", "datetime", "uuid"}, c.Type) {
		errs = append(errs, errors.Errorf("unknown type %q", c.Type))
	}

	for i, r := range c.Ranges {
		if rangeErrs := r.Validate(); len(rangeErrs) != 0 {
			errs = append(errs, errors.Errorf("ranges[%d]:", i))
			errs = append(errs, rangeErrs...)
		}

		if r.RangePercentage < 0 || r.RangePercentage > 1 {
			errs = append(errs, errors.Errorf("invalid range percentage should be between 0 and 1: got %v", r.RangePercentage))
		}

		rangePercentageSum += r.RangePercentage
	}

	if rangePercentageSum != 1 {
		errs = append(errs, errors.Errorf("sum of range percentages should be between 0 and 1: got %v", rangePercentageSum))
	}

	if parquetErrs := FieldValidate(c.ParquetParams); len(parquetErrs) != 0 {
		errs = append(errs, errors.New("parquet params:"))
		errs = append(errs, parquetErrs...)
	}

	return errs
}

type Params struct {
	ColumnType string `json:"-" yaml:"-"`
	//nolint:lll
	TypeParams         any                   `backup:"true" json:"type_params"         yaml:"type_params"` // only for config parsing
	IntegerParams      *ColumnIntegerParams  `json:"-"      yaml:"-"`
	FloatParams        *ColumnFloatParams    `json:"-"      yaml:"-"`
	StringParams       *ColumnStringParams   `json:"-"      yaml:"-"`
	DateTimeParams     *ColumnDateTimeParams `json:"-"      yaml:"-"`
	Values             []any                 `backup:"true" json:"values"              yaml:"values"`
	NullPercentage     float64               `backup:"true" json:"null_percentage"     yaml:"null_percentage"`
	DistinctPercentage float64               `backup:"true" json:"distinct_percentage" yaml:"distinct_percentage"`
	DistinctCount      uint64                `backup:"true" json:"distinct_count"      yaml:"distinct_count"`
	RangePercentage    float64               `backup:"true" json:"range_percentage"    yaml:"range_percentage"`
	Ordered            bool                  `backup:"true" json:"ordered"             yaml:"ordered"`
}

func (p *Params) Parse() error {
	var err error

	switch p.ColumnType {
	case "integer":
		p.IntegerParams, err = common.AnyToStruct[ColumnIntegerParams](p.TypeParams)
		p.TypeParams = p.IntegerParams
	case "float":
		p.FloatParams, err = common.AnyToStruct[ColumnFloatParams](p.TypeParams)
		p.TypeParams = p.FloatParams
	case "string":
		p.StringParams, err = common.AnyToStruct[ColumnStringParams](p.TypeParams)
		p.TypeParams = p.StringParams
	case "datetime":
		p.DateTimeParams, err = common.AnyToStruct[ColumnDateTimeParams](p.TypeParams)
		p.TypeParams = p.DateTimeParams
	}

	if err != nil {
		return errors.WithMessagef(err, "%s params", p.ColumnType)
	}

	if err = FieldParse(p.IntegerParams); err != nil {
		return errors.WithMessage(err, "integer params")
	}

	if err = FieldParse(p.FloatParams); err != nil {
		return errors.WithMessage(err, "float params")
	}

	if err = FieldParse(p.StringParams); err != nil {
		return errors.WithMessage(err, "string params")
	}

	if err = FieldParse(p.DateTimeParams); err != nil {
		return errors.WithMessage(err, "datetime params")
	}

	return nil
}

func (p *Params) FillDefaults() {
	FieldFillDefaults(p.IntegerParams)

	FieldFillDefaults(p.FloatParams)

	FieldFillDefaults(p.StringParams)

	FieldFillDefaults(p.DateTimeParams)
}

//nolint:cyclop
func (p *Params) Validate() []error {
	var errs []error

	if p.RangePercentage < 0 || p.RangePercentage > 1 {
		errs = append(errs, errors.Errorf("range percentage should be between 0 and 1, got %v", p.RangePercentage))
	}

	if p.NullPercentage < 0 || p.NullPercentage > 1 {
		errs = append(errs, errors.Errorf("null percentage should be between 0 and 1, got %v", p.NullPercentage))
	}

	if p.DistinctPercentage < 0 || p.DistinctPercentage > 1 {
		errs = append(errs, errors.Errorf("distinct percentage should be between 0 and 1, got %v", p.DistinctPercentage))
	}

	if p.Values != nil {
		if common.Any(
			p.DistinctPercentage != 0,
			p.DistinctCount != 0,
		) {
			errs = append(errs, errors.New("forbidden to use enum value with distinct params"))
		}
	}

	if p.DistinctPercentage != 0 && p.DistinctCount != 0 {
		errs = append(errs,
			errors.Errorf("forbidden to use distinct percentage (%v) and distinct count (%v) at the same time",
				p.DistinctPercentage, p.DistinctCount,
			))
	}

	if integerParamsErrs := FieldValidate(p.IntegerParams); len(integerParamsErrs) != 0 {
		errs = append(errs, errors.New("integer params:"))
		errs = append(errs, integerParamsErrs...)
	}

	if floatParamsErrs := FieldValidate(p.FloatParams); len(floatParamsErrs) != 0 {
		errs = append(errs, errors.New("float params:"))
		errs = append(errs, floatParamsErrs...)
	}

	if stringParamsErrs := FieldValidate(p.StringParams); len(stringParamsErrs) != 0 {
		errs = append(errs, errors.New("string params:"))
		errs = append(errs, stringParamsErrs...)
	}

	if datetimeParamsErrs := FieldValidate(p.DateTimeParams); len(datetimeParamsErrs) != 0 {
		errs = append(errs, errors.New("datetime params:"))
		errs = append(errs, datetimeParamsErrs...)
	}

	if p.StringParams != nil && p.StringParams.Template != "" {
		if common.Any(
			p.Ordered,
			p.DistinctPercentage != 0,
			p.DistinctCount != 0,
		) {
			errs = append(errs, errors.New("forbidden to use string template with distinct params or ordered"))
		}
	}

	// must be called only after parsing, filling defaults and validation of TypeParams.
	if p.Values != nil {
		if err := p.PostProcess(); err != nil {
			errs = append(errs, errors.WithMessage(err, "enum values"))
		}
	}

	return errs
}

func (p *Params) PostProcess() error {
	targetType, cmpFunc, err := p.determineTargetType()
	if err != nil {
		return err
	}

	err = p.convertValuesToTargetType(targetType)
	if err != nil {
		return err
	}

	common.SortSlice(p.Values, cmpFunc)

	return nil
}

//nolint:mnd,cyclop
func (p *Params) determineTargetType() (reflect.Type, common.CmpFunc, error) {
	var (
		targetType reflect.Type
		cmpFunc    common.CmpFunc
	)

	switch p.ColumnType {
	case "integer":
		cmpFunc = common.CmpInt

		switch p.IntegerParams.BitWidth {
		case 8:
			targetType = reflect.TypeFor[int8]()
		case 16:
			targetType = reflect.TypeFor[int16]()
		case 32:
			targetType = reflect.TypeFor[int32]()
		case 64:
			targetType = reflect.TypeFor[int64]()
		default:
			targetType = reflect.TypeFor[int]()
		}
	case "float":
		cmpFunc = common.CmpFloat

		switch p.FloatParams.BitWidth {
		case 32:
			targetType = reflect.TypeFor[float32]()
		case 64:
			targetType = reflect.TypeFor[float64]()
		default:
			targetType = reflect.TypeFor[float64]()
		}
	case "string":
		cmpFunc = common.CmpString
		targetType = reflect.TypeFor[string]()
	case "uuid":
		cmpFunc = common.CmpUUID
		targetType = reflect.TypeFor[uuid.UUID]()
	case "datetime":
		cmpFunc = common.CmpTime
		targetType = reflect.TypeFor[time.Time]()
	default:
		return nil, nil, errors.Errorf("unsupported type %q", p.ColumnType)
	}

	return targetType, cmpFunc, nil
}

//nolint:cyclop
func (p *Params) convertValuesToTargetType(targetType reflect.Type) error {
	var (
		convertedValue any
		err            error
	)

	for i, value := range p.Values {
		if value == nil {
			continue
		}

		switch targetType.Kind() {
		case reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
			convertedValue, err = common.ConvertToInt(value, targetType)
		case reflect.Float32, reflect.Float64:
			convertedValue, err = common.ConvertToFloat(value, targetType)
		case reflect.String:
			convertedValue, err = common.ConvertToString(value, targetType)
		case reflect.Array:
			convertedValue, err = common.ConvertToUUID(value, targetType)
		case reflect.Struct:
			switch targetType {
			case reflect.TypeFor[time.Time]():
				convertedValue, err = common.ConvertToTime(value, targetType)
			default:
				return errors.Errorf("unsupported type %s", targetType)
			}
		default:
			return errors.Errorf("unsupported type %s", targetType)
		}

		if err != nil {
			return err
		}

		p.Values[i] = convertedValue
	}

	return nil
}

// Verify interface compliance in compile time.
var _ Field = (*ColumnIntegerParams)(nil)

// ColumnIntegerParams type is used to describe params for integer fields.
type ColumnIntegerParams struct {
	BitWidth int    `backup:"true" json:"bit_width" yaml:"bit_width"`
	FromPtr  *int64 `backup:"true" json:"from"      yaml:"from"`
	From     int64  `json:"-"      yaml:"-"`
	ToPtr    *int64 `backup:"true" json:"to"        yaml:"to"`
	To       int64  `json:"-"      yaml:"-"`
}

func (p *ColumnIntegerParams) Parse() error {
	if p.FromPtr != nil {
		p.From = *p.FromPtr
	}

	if p.ToPtr != nil {
		p.To = *p.ToPtr
	}

	return nil
}

func (p *ColumnIntegerParams) FillDefaults() {
	if p.BitWidth == 0 {
		p.BitWidth = 32
	}

	if p.FromPtr == nil {
		p.From = -1 << (p.BitWidth - 1)
	}

	if p.ToPtr == nil {
		p.To = 1<<(p.BitWidth-1) - 1
	}
}

func (p *ColumnIntegerParams) Validate() []error {
	var errs []error

	if !slices.Contains([]int{8, 16, 32, 64}, p.BitWidth) {
		errs = append(errs, errors.Errorf("unsupported integer bit width: %d", p.BitWidth))
	}

	if p.From > p.To {
		errs = append(errs, errors.Errorf(
			"'from' field (%v) should be less than or equal to 'to' field (%v)", p.From, p.To,
		))
	}

	return errs
}

// Verify interface compliance in compile time.
var _ Field = (*ColumnFloatParams)(nil)

// ColumnFloatParams type is used to describe params for float fields.
type ColumnFloatParams struct {
	BitWidth int      `backup:"true" json:"bit_width" yaml:"bit_width"`
	FromPtr  *float64 `backup:"true" json:"from"      yaml:"from"`
	From     float64  `json:"-"      yaml:"-"`
	ToPtr    *float64 `backup:"true" json:"to"        yaml:"to"`
	To       float64  `json:"-"      yaml:"-"`
}

func (p *ColumnFloatParams) Parse() error {
	if p.FromPtr != nil {
		p.From = *p.FromPtr
	}

	if p.ToPtr != nil {
		p.To = *p.ToPtr
	}

	return nil
}

//nolint:mnd
func (p *ColumnFloatParams) FillDefaults() {
	if p.BitWidth == 0 {
		p.BitWidth = 32
	}

	var (
		minValue float64
		maxValue float64
	)

	if p.BitWidth == 32 {
		minValue = -math.MaxFloat32
		maxValue = math.MaxFloat32
	} else {
		minValue = -math.MaxFloat64
		maxValue = math.MaxFloat64
	}

	if p.FromPtr == nil {
		p.From = minValue
	}

	if p.ToPtr == nil {
		p.To = maxValue
	}
}

func (p *ColumnFloatParams) Validate() []error {
	var errs []error

	if !slices.Contains([]int{32, 64}, p.BitWidth) {
		errs = append(errs, errors.Errorf("unsupported float bit width: %d", p.BitWidth))
	}

	if p.From > p.To {
		errs = append(errs, errors.Errorf("'from' field (%v) should be less than or equal to 'to' field (%v)", p.From, p.To))
	}

	return errs
}

// Verify interface compliance in compile time.
var _ Field = (*ColumnStringParams)(nil)

// ColumnStringParams type is used to describe params for string fields.
type ColumnStringParams struct {
	MinLength           int    `backup:"true" json:"min_length"            yaml:"min_length"`
	MaxLength           int    `backup:"true" json:"max_length"            yaml:"max_length"`
	Locale              string `backup:"true" json:"locale"                yaml:"locale"`
	LogicalType         string `backup:"true" json:"logical_type"          yaml:"logical_type"`
	Template            string `backup:"true" json:"template"              yaml:"template"`
	Pattern             string `backup:"true" json:"pattern"               yaml:"pattern"`
	WithoutLargeLetters bool   `backup:"true" json:"without_large_letters" yaml:"without_large_letters"`
	WithoutSmallLetters bool   `backup:"true" json:"without_small_letters" yaml:"without_small_letters"`
	WithoutNumbers      bool   `backup:"true" json:"without_numbers"       yaml:"without_numbers"`
	WithoutSpecialChars bool   `backup:"true" json:"without_special_chars" yaml:"without_special_chars"`
}

func (p *ColumnStringParams) Parse() error { return nil }

func (p *ColumnStringParams) FillDefaults() {
	if p.MinLength == 0 {
		p.MinLength = 1
	}

	if p.MaxLength == 0 {
		p.MaxLength = 32
	}

	if p.Locale == "" {
		p.Locale = "en"
	}

	p.Locale = strings.ToLower(p.Locale)

	p.LogicalType = strings.ToLower(p.LogicalType)
}

func (p *ColumnStringParams) Validate() []error {
	var errs []error

	if p.Template != "" && p.Pattern != "" {
		errs = append(errs, errors.Errorf("forbidden to use template and pattern at the same time"))
	}

	if p.MinLength > p.MaxLength {
		errs = append(errs, errors.Errorf(
			"min length (%v) should be less than or equal to max length (%v)",
			p.MinLength, p.MaxLength,
		))
	}

	if !slices.Contains([]string{"ru", "en"}, p.Locale) {
		errs = append(errs, errors.Errorf("unknown locale: %s", p.Locale))
	}

	if !slices.Contains([]string{"", FirstNameType, LastNameType, PhoneType, TextType}, p.LogicalType) {
		errs = append(errs, errors.Errorf("unknown logical type: %s", p.LogicalType))
	}

	return errs
}

// Verify interface compliance in compile time.
var _ Field = (*ColumnDateTimeParams)(nil)

// ColumnDateTimeParams type is used to describe params for DateTime fields.
type ColumnDateTimeParams struct {
	From time.Time `backup:"true" json:"from" yaml:"from"`
	To   time.Time `backup:"true" json:"to"   yaml:"to"`
}

func (p *ColumnDateTimeParams) Parse() error { return nil }

func (p *ColumnDateTimeParams) FillDefaults() {
	if p.From.IsZero() {
		p.From = time.Date(1900, 1, 1, 0, 0, 0, 0, time.UTC)
	}

	if p.To.IsZero() {
		p.To = time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)
	}
}

func (p *ColumnDateTimeParams) Validate() []error {
	var errs []error

	if p.From.After(p.To) {
		errs = append(errs, errors.Errorf("'from' field (%v) should be before 'to' field (%v)", p.From, p.To))
	}

	return errs
}

// Verify interface compliance in compile time.
var _ Field = (*ColumnParquetParams)(nil)

var parquetSupportedEncodings = []string{
	"PLAIN",
	"PLAIN_DICT",
	"RLE",
	"RLE_DICTIONARY",
	"DELTA_BINARY_PACKED",
	"DELTA_BYTE_ARRAY",
	"DELTA_LENGTH_BYTE_ARRAY",
	"BYTE_STREAM_SPLIT",
}

// ColumnParquetParams type is used to describe params for parquet fields.
type ColumnParquetParams struct {
	Encoding string `backup:"true" json:"encoding" yaml:"encoding"`
}

func (p *ColumnParquetParams) Parse() error { return nil }

func (p *ColumnParquetParams) FillDefaults() {
	if p.Encoding == "" {
		p.Encoding = "PLAIN"
	}
}

func (p *ColumnParquetParams) Validate() []error {
	var errs []error

	if !slices.Contains(parquetSupportedEncodings, p.Encoding) {
		errs = append(errs, errors.Errorf("unknown parquet encoding: %q, supported: %s",
			p.Encoding, parquetSupportedEncodings))
	}

	return errs
}
