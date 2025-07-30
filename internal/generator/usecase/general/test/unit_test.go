package test

import (
	"context"
	"encoding/json"
	"math"
	"slices"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/pkg/errors"
	"github.com/stretchr/testify/require"
	"github.com/tarantool/sdvg/internal/generator/models"
	outputMock "github.com/tarantool/sdvg/internal/generator/output/mock"
	"github.com/tarantool/sdvg/internal/generator/usecase"
	usecaseGeneral "github.com/tarantool/sdvg/internal/generator/usecase/general"
	"github.com/tarantool/sdvg/internal/generator/usecase/general/generator/value"
)

const (
	UnitDefaultRowsCount  = 251
	UnitDefaultColumnName = "test"
	UnitWorkersCount      = 1
	UnitBatchSize         = 2
	UnitRandomSeed        = 1738591926070236604
)

func int64Ptr(i int64) *int64 {
	return &i
}

func float64Ptr(i float64) *float64 {
	return &i
}

func deepColumnCopy(c *models.Column) *models.Column {
	cCopy := *c

	var (
		fk *models.Column
		pp *models.ColumnParquetParams
	)

	if c.ForeignKeyColumn != nil {
		fkCopy := *c.ForeignKeyColumn
		fk = &fkCopy
	}

	if c.ParquetParams != nil {
		ppCopy := *c.ParquetParams
		pp = &ppCopy
	}

	ranges := make([]*models.Params, 0, len(c.Ranges))

	for i := range c.Ranges {
		r := *c.Ranges[i]
		ranges = append(ranges, &r)
	}

	cCopy.ForeignKeyColumn = fk
	cCopy.ParquetParams = pp
	cCopy.Ranges = ranges

	return &cCopy
}

func toString(t *testing.T, anyValue any) string {
	t.Helper()

	val, err := json.Marshal(anyValue)
	if err != nil {
		t.Fatalf("Failed to json marshal of %v: %s", val, err)
	}

	return string(val)
}

func getCfg(t *testing.T, model map[string]*models.Model) models.GenerationConfig {
	t.Helper()

	cfg := models.GenerationConfig{
		WorkersCount: UnitWorkersCount,
		BatchSize:    UnitBatchSize,
		RandomSeed:   UnitRandomSeed,
		Models:       model,
	}

	require.NoError(t, cfg.Parse())
	cfg.FillDefaults()
	require.Empty(t, cfg.Validate())

	return cfg
}

func oneColumnCfg(t *testing.T, column *models.Column) models.GenerationConfig {
	t.Helper()

	return getCfg(t, map[string]*models.Model{
		UnitDefaultColumnName: {
			RowsCount: UnitDefaultRowsCount,
			Columns:   []*models.Column{column},
		},
	})
}

func generateFunc(t *testing.T, cfg models.GenerationConfig) map[string][]*models.DataRow {
	t.Helper()

	mutex := sync.Mutex{}
	handled := make(map[string][]*models.DataRow)

	outputHandler := func(_ context.Context, modelName string, rows []*models.DataRow) error {
		mutex.Lock()
		defer mutex.Unlock()

		handled[modelName] = append(handled[modelName], rows...)

		return nil
	}

	out := outputMock.NewOutput(outputHandler)
	uc := usecaseGeneral.NewUseCase(usecaseGeneral.UseCaseConfig{})

	taskID, err := uc.CreateTask(
		context.Background(),
		usecase.TaskConfig{
			GenerationConfig: &cfg,
			Output:           out,
		},
	)
	if err != nil {
		t.Fatal(err)
	}

	err = uc.WaitResult(taskID)
	if err != nil {
		t.Fatal(err)
	}

	return handled
}

func checkType(t *testing.T, column *models.Column, expectedType any) []*models.DataRow {
	t.Helper()

	handled := generateFunc(t, oneColumnCfg(t, column))[UnitDefaultColumnName]
	require.Len(t, handled, UnitDefaultRowsCount, "column: %+v\n handled: %+v", column, handled)
	require.Len(t, handled[0].Values, 1, "column: %+v\n handled: %+v", column, handled)
	require.IsType(t, expectedType, handled[0].Values[0], "column: %+v\n handled: %+v", column, handled)

	return handled
}

func checkValue(t *testing.T, column *models.Column, expectedValue any) {
	t.Helper()

	handled := generateFunc(t, oneColumnCfg(t, column))[UnitDefaultColumnName]
	require.Len(t, handled, UnitDefaultRowsCount, "column: %+v\n handled: %+v", column, handled)
	require.Len(t, handled[0].Values, 1, "column: %+v\n handled: %+v", column, handled)

	for i := range UnitDefaultRowsCount {
		require.Equal(t, expectedValue, handled[i].Values[0], "column: %+v\n handled: %+v", column, handled)
	}
}

func checkOrdered(t *testing.T, column *models.Column) {
	t.Helper()

	column = deepColumnCopy(column)
	column.Ranges[0].Ordered = true
	handled := generateFunc(t, oneColumnCfg(t, column))[UnitDefaultColumnName]
	require.Len(t, handled, UnitDefaultRowsCount, handled)

	for i := range UnitDefaultRowsCount - 1 {
		require.Len(t, handled[i].Values, 1, "column: %+v\n handled: %+v", column, handled)
		require.Len(t, handled[i+1].Values, 1, "column: %+v\n handled: %+v", column, handled)
		require.LessOrEqual(t, handled[i].Values[0], handled[i+1].Values[0], "column: %+v\n handled: %+v", column, handled)
	}
}

func checkDistinct(t *testing.T, column *models.Column) {
	t.Helper()

	column = deepColumnCopy(column)
	column.Ranges[0].DistinctPercentage = 1
	handled := generateFunc(t, oneColumnCfg(t, column))[UnitDefaultColumnName]
	require.Len(t, handled, UnitDefaultRowsCount, "column: %+v\n handled: %+v", column, handled)

	uniqueMap := make(map[string]bool)

	for i := range UnitDefaultRowsCount {
		require.Len(t, handled[i].Values, 1, "column: %+v\n handled: %+v", column, handled)
		val := toString(t, handled[i].Values[0])
		_, alreadyHas := uniqueMap[val]
		require.False(t, alreadyHas, "value: %+v\nmap: %+v", val, uniqueMap)
		uniqueMap[val] = true
	}
}

func checkValuesCount(t *testing.T, gen value.Generator, expectedValueCount float64) {
	t.Helper()

	require.NoError(t, gen.Prepare())

	valuesCount := gen.ValuesCount()
	require.Equal(t, uint64(expectedValueCount), uint64(valuesCount))
}

func checkForeignKey(t *testing.T, column *models.Column, nullPercentage float64, foreignOrdered bool) {
	t.Helper()

	column = deepColumnCopy(column)
	column.Name = "test"
	column.Ranges[0].NullPercentage = nullPercentage

	cfg := getCfg(t, map[string]*models.Model{
		"orig": {
			RowsCount: UnitDefaultRowsCount,
			Columns:   []*models.Column{column},
		},
		"foreign": {
			RowsCount: UnitDefaultRowsCount * 2,
			Columns: []*models.Column{{
				Name:       "foreign_key",
				ForeignKey: "orig.test",
				Params:     &models.Params{Ordered: foreignOrdered},
			}},
		},
	})
	handled := generateFunc(t, cfg)

	origHandled := handled["orig"]
	require.Len(t, origHandled, UnitDefaultRowsCount, "column: %+v\n handled: %+v", column, origHandled)

	foreignHandled := handled["foreign"]
	require.Len(t, foreignHandled, 2*UnitDefaultRowsCount, "column: %+v\n handled: %+v", column, foreignHandled)

	origMap := make(map[any]bool)

	for i := range UnitDefaultRowsCount {
		require.Len(t, origHandled[i].Values, 1, "column: %+v\n handled: %+v", column, origHandled)

		if origHandled[i].Values[0] == nil {
			// skip nullable values
			continue
		}

		val := toString(t, origHandled[i].Values[0])
		_, alreadyHas := origMap[val]
		require.False(t, alreadyHas, "value: %+v\nmap: %+v", val, origMap)
		origMap[val] = true
	}

	for i := range UnitDefaultRowsCount * 2 {
		require.Len(t, foreignHandled[i].Values, 1, "column: %+v\n handled: %+v", column, foreignHandled)

		if foreignHandled[i].Values[0] == nil {
			// skip nullable values
			continue
		}

		if i > 0 && foreignOrdered {
			lhs := foreignHandled[i].Values[0]
			rhs := foreignHandled[i-1].Values[0]

			if lhs != nil && rhs != nil {
				//nolint:forcetypeassert
				if column.Type == "uuid" {
					lhs = lhs.(uuid.UUID).String()
					rhs = rhs.(uuid.UUID).String()
				}

				require.GreaterOrEqual(t, lhs, rhs)
			}
		}

		val := toString(t, foreignHandled[i].Values[0])
		_, alreadyHas := origMap[val]
		require.True(t, alreadyHas, "value: %+v (#%d)\nmap: %+v", val, i, origMap)
	}
}

func checkForeignKeyCases(t *testing.T, column *models.Column) {
	t.Helper()

	checkForeignKey(t, column, 0, false)
	checkForeignKey(t, column, 0, true)
	checkForeignKey(t, column, 0.3, false)
	checkForeignKey(t, column, 0.3, true)
}

func TestInteger(t *testing.T) {
	checkTypeCases := []struct {
		typeParams *models.ColumnIntegerParams
		expected   any
	}{
		{nil, int32(0)},
		{&models.ColumnIntegerParams{BitWidth: 8}, int8(0)},
		{&models.ColumnIntegerParams{BitWidth: 16}, int16(0)},
		{&models.ColumnIntegerParams{BitWidth: 32}, int32(0)},
		{&models.ColumnIntegerParams{BitWidth: 64}, int64(0)},
		{
			&models.ColumnIntegerParams{BitWidth: 8, FromPtr: int64Ptr(math.MinInt8), ToPtr: int64Ptr(math.MaxInt8)},
			int8(0),
		},
		{
			&models.ColumnIntegerParams{BitWidth: 16, FromPtr: int64Ptr(math.MinInt16), ToPtr: int64Ptr(math.MaxInt16)},
			int16(0),
		},
		{
			&models.ColumnIntegerParams{BitWidth: 32, FromPtr: int64Ptr(math.MinInt32), ToPtr: int64Ptr(math.MaxInt32)},
			int32(0),
		},
		{
			&models.ColumnIntegerParams{BitWidth: 64, FromPtr: int64Ptr(math.MinInt64), ToPtr: int64Ptr(math.MaxInt64)},
			int64(0),
		},
	}

	for _, testCase := range checkTypeCases {
		column := &models.Column{
			Name:   "integers",
			Type:   "integer",
			Ranges: []*models.Params{{TypeParams: testCase.typeParams}},
		}

		checkType(t, column, testCase.expected)
		checkOrdered(t, column)
		checkDistinct(t, column)
		checkForeignKeyCases(t, column)
	}

	checkValueCases := []struct {
		typeParams *models.ColumnIntegerParams
		expected   any
	}{
		{
			&models.ColumnIntegerParams{BitWidth: 8, FromPtr: int64Ptr(math.MinInt8), ToPtr: int64Ptr(math.MinInt8)},
			int8(math.MinInt8),
		},
		{
			&models.ColumnIntegerParams{BitWidth: 16, FromPtr: int64Ptr(math.MinInt16), ToPtr: int64Ptr(math.MinInt16)},
			int16(math.MinInt16),
		},
		{
			&models.ColumnIntegerParams{BitWidth: 32, FromPtr: int64Ptr(math.MinInt32), ToPtr: int64Ptr(math.MinInt32)},
			int32(math.MinInt32),
		},
		{
			&models.ColumnIntegerParams{BitWidth: 64, FromPtr: int64Ptr(math.MinInt64), ToPtr: int64Ptr(math.MinInt64)},
			int64(math.MinInt64),
		},
		{
			&models.ColumnIntegerParams{BitWidth: 8, FromPtr: int64Ptr(math.MaxInt8), ToPtr: int64Ptr(math.MaxInt8)},
			int8(math.MaxInt8),
		},
		{
			&models.ColumnIntegerParams{BitWidth: 16, FromPtr: int64Ptr(math.MaxInt16), ToPtr: int64Ptr(math.MaxInt16)},
			int16(math.MaxInt16),
		},
		{
			&models.ColumnIntegerParams{BitWidth: 32, FromPtr: int64Ptr(math.MaxInt32), ToPtr: int64Ptr(math.MaxInt32)},
			int32(math.MaxInt32),
		},
		{
			&models.ColumnIntegerParams{BitWidth: 64, FromPtr: int64Ptr(math.MaxInt64), ToPtr: int64Ptr(math.MaxInt64)},
			int64(math.MaxInt64),
		},
	}

	for _, testCase := range checkValueCases {
		column := &models.Column{
			Name:   "integers",
			Type:   "integer",
			Ranges: []*models.Params{{TypeParams: testCase.typeParams}},
		}

		checkValue(t, column, testCase.expected)
	}

	checkValuesCountCases := []struct {
		typeParams *models.ColumnIntegerParams
		expected   float64
	}{
		{&models.ColumnIntegerParams{From: 1, To: 5}, 5},
		{&models.ColumnIntegerParams{From: 100, To: 1000}, 901},
		{&models.ColumnIntegerParams{From: 1, To: 1}, 1},
		{&models.ColumnIntegerParams{From: 123, To: 654}, 532},
	}

	for _, testCase := range checkValuesCountCases {
		generator := &value.IntegerGenerator{ColumnIntegerParams: testCase.typeParams}
		checkValuesCount(t, generator, testCase.expected)
	}
}

func TestFloat(t *testing.T) {
	checkTypeCases := []struct {
		typeParams *models.ColumnFloatParams
		expected   any
	}{
		{nil, float32(0)},
		{&models.ColumnFloatParams{BitWidth: 32}, float32(0)},
		{&models.ColumnFloatParams{BitWidth: 64}, float64(0)},
		{
			&models.ColumnFloatParams{BitWidth: 32, FromPtr: float64Ptr(-math.MaxFloat32), ToPtr: float64Ptr(math.MaxFloat32)},
			float32(0),
		},
		{
			&models.ColumnFloatParams{BitWidth: 64, FromPtr: float64Ptr(-math.MaxFloat64), ToPtr: float64Ptr(math.MaxFloat64)},
			float64(0),
		},
	}

	for _, testCase := range checkTypeCases {
		column := &models.Column{
			Name:   "floats",
			Type:   "float",
			Ranges: []*models.Params{{TypeParams: testCase.typeParams}},
		}

		checkType(t, column, testCase.expected)
		checkOrdered(t, column)
		checkDistinct(t, column)
		checkForeignKeyCases(t, column)
	}

	checkValueCases := []struct {
		typeParams *models.ColumnFloatParams
		expected   any
	}{
		{
			&models.ColumnFloatParams{BitWidth: 32, FromPtr: float64Ptr(-math.MaxFloat32), ToPtr: float64Ptr(-math.MaxFloat32)},
			float32(-math.MaxFloat32),
		},
		{
			&models.ColumnFloatParams{BitWidth: 64, FromPtr: float64Ptr(-math.MaxFloat64), ToPtr: float64Ptr(-math.MaxFloat64)},
			-math.MaxFloat64,
		},
		{
			&models.ColumnFloatParams{BitWidth: 32, FromPtr: float64Ptr(math.MaxFloat32), ToPtr: float64Ptr(math.MaxFloat32)},
			float32(math.MaxFloat32),
		},
		{
			&models.ColumnFloatParams{BitWidth: 64, FromPtr: float64Ptr(math.MaxFloat64), ToPtr: float64Ptr(math.MaxFloat64)},
			math.MaxFloat64,
		},
	}

	for _, testCase := range checkValueCases {
		column := &models.Column{
			Name:   "floats",
			Type:   "float",
			Ranges: []*models.Params{{TypeParams: testCase.typeParams}},
		}

		checkValue(t, column, testCase.expected)
	}

	checkValuesCountCases := []struct {
		typeParams *models.ColumnFloatParams
		expected   float64
	}{
		{&models.ColumnFloatParams{From: 1.021, To: 5.554433}, math.Inf(1)},
		{&models.ColumnFloatParams{From: 195.2345, To: 1000}, math.Inf(1)},
		{&models.ColumnFloatParams{From: 0.12345, To: 1}, math.Inf(1)},
		{&models.ColumnFloatParams{From: 123, To: 654}, math.Inf(1)},
	}

	for _, testCase := range checkValuesCountCases {
		generator := &value.FloatGenerator{ColumnFloatParams: testCase.typeParams}
		checkValuesCount(t, generator, testCase.expected)
	}
}

func TestString(t *testing.T) {
	testCases := []struct {
		typeParams *models.ColumnStringParams
		minLen     int
		maxLen     int
	}{
		{&models.ColumnStringParams{}, 1, 32},
		{&models.ColumnStringParams{LogicalType: models.FirstNameType, Locale: "en"}, 1, 32},
		{&models.ColumnStringParams{LogicalType: models.LastNameType, Locale: "en"}, 1, 32},
		{&models.ColumnStringParams{LogicalType: models.PhoneType, Locale: "en"}, 1, 32},
		{&models.ColumnStringParams{LogicalType: models.FirstNameType, Locale: "ru"}, 1, 32},
		{&models.ColumnStringParams{LogicalType: models.LastNameType, Locale: "ru"}, 1, 32},
		{&models.ColumnStringParams{LogicalType: models.PhoneType, Locale: "ru"}, 1, 32},
		{&models.ColumnStringParams{MinLength: 5, MaxLength: 5}, 5, 5},
		{&models.ColumnStringParams{LogicalType: models.FirstNameType, MinLength: 5, MaxLength: 5}, 5, 5},
		{&models.ColumnStringParams{LogicalType: models.LastNameType, MinLength: 4, MaxLength: 7}, 4, 7},
		{&models.ColumnStringParams{LogicalType: models.PhoneType, MinLength: 10, MaxLength: 10}, 10, 10},
		{&models.ColumnStringParams{MinLength: 100, MaxLength: 100}, 100, 100},
		{&models.ColumnStringParams{Pattern: "AAaa00##", Locale: "en"}, 8, 8},
		{&models.ColumnStringParams{Pattern: "AAaa00##", Locale: "ru"}, 8, 8},
		{&models.ColumnStringParams{Pattern: "0123456789012345678901234567890123456789"}, 40, 40},
		{&models.ColumnStringParams{LogicalType: models.TextType, MinLength: 3, MaxLength: 5}, 3, 5},
		{&models.ColumnStringParams{LogicalType: models.TextType, MinLength: 254, MaxLength: 256}, 254, 256},
		{&models.ColumnStringParams{LogicalType: models.TextType, MinLength: 510, MaxLength: 512}, 510, 512},
		{&models.ColumnStringParams{LogicalType: models.TextType, MinLength: 3, MaxLength: 5, Locale: "ru"}, 3, 5},
		{&models.ColumnStringParams{LogicalType: models.TextType, MinLength: 254, MaxLength: 256, Locale: "ru"}, 254, 256},
		{&models.ColumnStringParams{LogicalType: models.TextType, MinLength: 510, MaxLength: 512, Locale: "ru"}, 510, 512},
	}

	for _, testCase := range testCases {
		column := &models.Column{
			Name:   "strings",
			Type:   "string",
			Ranges: []*models.Params{{TypeParams: testCase.typeParams}},
		}

		handled := checkType(t, column, "")
		strValue, ok := handled[0].Values[0].(string)
		require.True(t, ok)

		strValueLen := len([]rune(strValue))
		require.GreaterOrEqual(t, strValueLen, testCase.minLen, handled)
		require.LessOrEqual(t, strValueLen, testCase.maxLen, handled)

		checkOrdered(t, column)
		checkDistinct(t, column)
		checkForeignKeyCases(t, column)
	}

	checkValuesCountCases := []struct {
		typeParams *models.ColumnStringParams
		expected   float64
	}{
		{
			&models.ColumnStringParams{
				MinLength:           1,
				MaxLength:           1,
				Locale:              "en",
				WithoutNumbers:      true,
				WithoutSpecialChars: true,
			},
			52,
		},
		{
			&models.ColumnStringParams{
				MinLength:           1,
				MaxLength:           1,
				Locale:              "ru",
				WithoutNumbers:      true,
				WithoutSpecialChars: true,
			},
			66.0,
		},
		{
			&models.ColumnStringParams{
				MinLength:           3,
				MaxLength:           7,
				Locale:              "en",
				WithoutNumbers:      true,
				WithoutSpecialChars: true,
			},
			1048229968448,
		},
		{
			&models.ColumnStringParams{
				MinLength:           2,
				MaxLength:           9,
				Locale:              "ru",
				WithoutNumbers:      true,
				WithoutSpecialChars: true,
			},
			24128259706319868,
		},
		{
			&models.ColumnStringParams{
				MinLength:           10,
				MaxLength:           24,
				Locale:              "en",
				WithoutLargeLetters: true,
				WithoutSmallLetters: true,
				WithoutSpecialChars: true,
			},
			1111111111111110000000000,
		},
		{
			&models.ColumnStringParams{
				MinLength:           1,
				MaxLength:           8,
				Locale:              "en",
				WithoutLargeLetters: true,
				WithoutSmallLetters: true,
				WithoutNumbers:      true,
			},
			81870575520,
		},
		{
			&models.ColumnStringParams{
				MinLength: 10,
				MaxLength: 15,
				Locale:    "en",
			},
			88394150280794134360488281250,
		},
		{
			&models.ColumnStringParams{
				MinLength: 10,
				MaxLength: 15,
				Locale:    "ru",
			},
			868834460299970670989801640300,
		},
		{
			&models.ColumnStringParams{
				Locale:   "en",
				Template: "{{ .field }}",
			},
			1,
		},
		{
			&models.ColumnStringParams{
				Locale:  "en",
				Pattern: "A00",
			},
			2600,
		},
	}

	for _, testCase := range checkValuesCountCases {
		generator := &value.StringGenerator{ColumnStringParams: testCase.typeParams}
		checkValuesCount(t, generator, testCase.expected)
	}
}

func TestUUID(t *testing.T) {
	column := &models.Column{Name: "uuids", Type: "uuid"}
	checkType(t, column, uuid.UUID{})
	checkDistinct(t, column)
	checkForeignKeyCases(t, column)
	checkValuesCount(t, &value.UUIDGenerator{}, float64(1<<(128-10)-1))
}

func TestDateTime(t *testing.T) {
	minDate := time.Date(1, 1, 1, 0, 0, 0, 1, time.UTC)
	maxDate := time.Date(2099, 12, 31, 23, 59, 59, 0, time.UTC)

	checkTypeCases := []struct {
		typeParams *models.ColumnDateTimeParams
		expected   any
	}{
		{nil, time.Time{}},
		{&models.ColumnDateTimeParams{}, time.Time{}},
		{&models.ColumnDateTimeParams{From: minDate, To: maxDate}, time.Time{}},
	}

	for _, testCase := range checkTypeCases {
		column := &models.Column{
			Name:   "datetimes",
			Type:   "datetime",
			Ranges: []*models.Params{{TypeParams: testCase.typeParams}},
		}

		checkType(t, column, testCase.expected)
		checkOrdered(t, column)
		checkDistinct(t, column)
		checkForeignKeyCases(t, column)
	}

	checkValueCases := []struct {
		typeParams *models.ColumnDateTimeParams
		expected   any
	}{
		{&models.ColumnDateTimeParams{From: minDate, To: minDate}, minDate},
		{&models.ColumnDateTimeParams{From: maxDate, To: maxDate}, maxDate},
	}

	for _, testCase := range checkValueCases {
		column := &models.Column{
			Name:   "datetimes",
			Type:   "datetime",
			Ranges: []*models.Params{{TypeParams: testCase.typeParams}},
		}

		checkValue(t, column, testCase.expected)
	}

	checkValuesCountCases := []struct {
		typeParams *models.ColumnDateTimeParams
		expected   float64
	}{
		{
			&models.ColumnDateTimeParams{
				From: time.Date(2025, 7, 25, 10, 0, 0, 0, time.UTC),
				To:   time.Date(2025, 7, 25, 10, 0, 0, 0, time.UTC),
			},
			1,
		},
		{
			&models.ColumnDateTimeParams{
				From: time.Date(2025, 7, 25, 10, 0, 0, 500_000_000, time.UTC),
				To:   time.Date(2025, 7, 25, 10, 0, 5, 500_000_000, time.UTC),
			},
			6,
		},
		{
			&models.ColumnDateTimeParams{
				From: time.Date(2025, 7, 25, 10, 0, 0, 900_000_000, time.UTC),
				To:   time.Date(2025, 7, 25, 10, 0, 1, 100_000_000, time.UTC),
			},
			400_000_002,
		},
		{
			&models.ColumnDateTimeParams{
				From: time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC),
				To:   time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC),
			},
			31_536_001,
		},
	}

	for _, testCase := range checkValuesCountCases {
		generator := &value.DateTimeGenerator{ColumnDateTimeParams: testCase.typeParams}
		checkValuesCount(t, generator, testCase.expected)
	}
}

func TestIdempotence(t *testing.T) {
	cfg := models.GenerationConfig{
		RandomSeed:   0,
		WorkersCount: 3,
		BatchSize:    2,
		Models: map[string]*models.Model{
			"user": {
				RowsCount: UnitDefaultRowsCount,
				Columns: []*models.Column{
					{
						Name: "id",
						Type: "integer",
						Ranges: []*models.Params{{TypeParams: &models.ColumnIntegerParams{
							ToPtr: int64Ptr(10_000_000),
						},
							DistinctPercentage: 0.5}},
					},
					{
						Name:   "id_2",
						Type:   "integer",
						Params: &models.Params{Ordered: true},
					},
					{
						Name: "char_id",
						Type: "integer",
						Ranges: []*models.Params{{TypeParams: &models.ColumnIntegerParams{
							BitWidth: 8,
						}}},
					},
					{
						Name: "short_id",
						Type: "integer",
						Ranges: []*models.Params{{TypeParams: &models.ColumnIntegerParams{
							BitWidth: 16,
						}}},
					},
					{
						Name: "long_id",
						Type: "integer",
						Ranges: []*models.Params{{TypeParams: &models.ColumnIntegerParams{
							BitWidth: 64,
						}}},
					},
					{
						Name: "str_id",
						Type: "string",
						Ranges: []*models.Params{{
							TypeParams: &models.ColumnStringParams{
								MinLength: 16,
								MaxLength: 32,
							},
							Ordered: true}},
					},
					{
						Name: "ru_phone",
						Type: "string",
						Ranges: []*models.Params{{TypeParams: &models.ColumnStringParams{
							LogicalType: "phone",
							Locale:      "ru",
						}}},
					},
					{
						Name: "first_name_ru",
						Type: "string",
						Ranges: []*models.Params{{TypeParams: &models.ColumnStringParams{
							LogicalType: "first_name",
							Locale:      "ru",
						}}},
					},
					{
						Name: "last_name_ru",
						Type: "string",
						Ranges: []*models.Params{{TypeParams: &models.ColumnStringParams{
							LogicalType: "last_name",
							Locale:      "ru",
						},
							DistinctCount: 5}},
					},
					{
						Name: "first_name_en",
						Type: "string",
						Ranges: []*models.Params{{TypeParams: &models.ColumnStringParams{
							LogicalType: "first_name",
							Locale:      "en",
						}}},
					},
					{
						Name: "passport",
						Type: "string",
						Ranges: []*models.Params{{TypeParams: &models.ColumnStringParams{
							Pattern: "AA 00 000 000",
						},
							NullPercentage: 0.5}},
					},
					{
						Name: "datetime",
						Type: "datetime",
					},
					{
						Name: "uuid",
						Type: "uuid",
					},
					{
						Name:   "enum",
						Type:   "integer",
						Ranges: []*models.Params{{Values: []any{nil, 1}}},
					},
				},
			},
			"token": {
				RowsCount: UnitDefaultRowsCount,
				Columns: []*models.Column{
					{
						Name: "id",
						Type: "integer",
					},
					{
						Name:       "user_id",
						ForeignKey: "user.id",
					},
				},
			},
		},
	}

	require.NoError(t, cfg.Parse())
	cfg.FillDefaults()
	require.Empty(t, cfg.Validate())

	handled1 := generateFunc(t, cfg)
	require.Len(t, handled1, 2, handled1)
	require.Len(t, handled1["user"], UnitDefaultRowsCount, handled1)
	require.Len(t, handled1["token"], UnitDefaultRowsCount, handled1)

	cfg.RandomSeed = 0

	require.NoError(t, cfg.Parse())
	cfg.FillDefaults()
	require.Empty(t, cfg.Validate())

	handled2 := generateFunc(t, cfg)
	require.Len(t, handled2, 2, handled2)
	require.Len(t, handled2["user"], UnitDefaultRowsCount, handled2)
	require.Len(t, handled2["token"], UnitDefaultRowsCount, handled2)

	require.NotEqual(t, handled1, handled2)

	cfg.RandomSeed = 10

	handled1 = generateFunc(t, cfg)
	require.Len(t, handled1, 2, handled1)
	require.Len(t, handled1["user"], UnitDefaultRowsCount, handled1)
	require.Len(t, handled1["token"], UnitDefaultRowsCount, handled1)

	handled2 = generateFunc(t, cfg)
	require.Len(t, handled2, 2, handled2)
	require.Len(t, handled2["user"], UnitDefaultRowsCount, handled2)
	require.Len(t, handled2["token"], UnitDefaultRowsCount, handled2)

	require.Equal(t, handled1, handled2)
}

func TestEnum(t *testing.T) {
	firstUUID, err := uuid.Parse("99aa4717-e6a0-4adc-b2c8-ba505e7ffc00")
	require.NoError(t, err)

	secondUUID, err := uuid.Parse("7867d43c-91cc-481a-820b-3eb7c2dede86")
	require.NoError(t, err)

	testCases := []struct {
		name      string
		dataType  string
		rowsCount uint64
		values    []any
		expected  []any
	}{
		{
			name:      "integer",
			dataType:  "integer",
			rowsCount: 9,
			values:    []any{222, nil, "111"},
			expected:  []any{nil, nil, nil, int32(111), int32(111), int32(111), int32(222), int32(222), int32(222)},
		},
		{
			name:      "float",
			dataType:  "float",
			rowsCount: 4,
			values:    []any{31.23123, nil, "1.4123111111", 5},
			expected:  []any{nil, float32(1.4123111111), float32(5), float32(31.23123)},
		},
		{
			name:      "string",
			dataType:  "string",
			rowsCount: 5,
			values:    []any{"values_1", nil, "values_2", 1, 1.5},
			expected:  []any{nil, "1", "1.5", "values_1", "values_2"},
		},
		{
			name:      "datetime",
			dataType:  "datetime",
			rowsCount: 5,
			values: []any{
				nil,
				time.Date(2000, 1, 1, 0, 0, 0, 0, time.UTC),
				time.Date(2001, 12, 31, 23, 59, 59, 1e9-1, time.UTC),
			},
			expected: []any{
				nil,
				nil,
				time.Date(2000, 1, 1, 0, 0, 0, 0, time.UTC),
				time.Date(2000, 1, 1, 0, 0, 0, 0, time.UTC),
				time.Date(2001, 12, 31, 23, 59, 59, 1e9-1, time.UTC)},
		},
		{
			name:      "uuid",
			dataType:  "uuid",
			rowsCount: 2,
			values: []any{
				"99aa4717-e6a0-4adc-b2c8-ba505e7ffc00",
				"7867d43c-91cc-481a-820b-3eb7c2dede86",
			},
			expected: []any{secondUUID, firstUUID},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			column := &models.Column{
				Name:   "enums",
				Type:   tc.dataType,
				Ranges: []*models.Params{{Values: tc.values}},
			}

			cfg := oneColumnCfg(t, column)
			cfg.Models[UnitDefaultColumnName].RowsCount = tc.rowsCount
			cfg.Models[UnitDefaultColumnName].GenerateTo = tc.rowsCount

			handledDataRows := generateFunc(t, cfg)[UnitDefaultColumnName]
			require.Len(t, handledDataRows, len(tc.expected))

			columnOrdered := &models.Column{
				Name:   "enums",
				Type:   tc.dataType,
				Ranges: []*models.Params{{Values: tc.values, Ordered: true}},
			}

			cfg = oneColumnCfg(t, columnOrdered)
			cfg.Models[UnitDefaultColumnName].RowsCount = tc.rowsCount
			cfg.Models[UnitDefaultColumnName].GenerateTo = tc.rowsCount

			handledDataRows = generateFunc(t, cfg)[UnitDefaultColumnName]
			require.Len(t, handledDataRows, len(tc.expected))

			for i := range handledDataRows {
				val := handledDataRows[i].Values[0]
				require.Equal(t, tc.expected[i], val)
			}
		})
	}
}

func TestIgnore(t *testing.T) {
	cfg := models.GenerationConfig{
		RandomSeed:     0,
		WorkersCount:   3,
		BatchSize:      2,
		ModelsToIgnore: []string{"user"},
		Models: map[string]*models.Model{
			"user": {
				RowsCount: UnitDefaultRowsCount,
				Columns: []*models.Column{
					{
						Name: "id",
						Type: "integer",
					},
				},
			},
			"token": {
				RowsCount: UnitDefaultRowsCount,
				Columns: []*models.Column{
					{
						Name: "id",
						Type: "integer",
					},
					{
						Name:       "user_id",
						ForeignKey: "user.id",
					},
				},
			},
		},
	}

	require.NoError(t, cfg.Parse())
	cfg.FillDefaults()
	require.Empty(t, cfg.Validate())

	handled := generateFunc(t, cfg)
	require.Len(t, handled, 1, handled)
	require.Len(t, handled["token"], UnitDefaultRowsCount, handled)
}

func TestRanges(t *testing.T) {
	testCases := []struct {
		name     string
		dataType string
		ranges   []*models.Params
	}{
		{
			name:     "integer with three ranges, where one is enum",
			dataType: "integer",
			ranges: []*models.Params{
				{
					TypeParams: &models.ColumnIntegerParams{
						BitWidth: 32,
						FromPtr:  int64Ptr(-200),
						ToPtr:    int64Ptr(-100),
					},
				},
				{
					TypeParams: &models.ColumnIntegerParams{
						BitWidth: 64,
						FromPtr:  int64Ptr(300),
						ToPtr:    int64Ptr(400),
					},
				},
				{
					Values: []any{999},
				},
			},
		},
		{
			name:     "string with three ranges, where one has custom range percentage and one is enum",
			dataType: "string",
			ranges: []*models.Params{
				{
					TypeParams: &models.ColumnStringParams{
						MinLength: 3,
						MaxLength: 10,
					},
					RangePercentage: 0.5,
				},
				{
					TypeParams: &models.ColumnStringParams{
						MinLength: 20,
						MaxLength: 30,
					},
				},
				{
					Values: []any{"v1", "v2"},
				},
			},
		},
		{
			name:     "datetime with three ranges, where each has custom range percentage and one is enum",
			dataType: "datetime",
			ranges: []*models.Params{
				{
					TypeParams: &models.ColumnDateTimeParams{
						From: time.Date(100, 1, 1, 0, 0, 0, 1, time.UTC),
						To:   time.Date(300, 1, 1, 0, 0, 0, 1, time.UTC),
					},
					RangePercentage: 0.85,
				},
				{
					TypeParams: &models.ColumnDateTimeParams{
						From: time.Date(1800, 12, 31, 23, 59, 59, 0, time.UTC),
						To:   time.Date(1900, 12, 31, 23, 59, 59, 0, time.UTC),
					},
					RangePercentage: 0.1,
				},
				{
					Values: []any{
						time.Date(2005, 3, 9, 4, 55, 00, 0, time.UTC),
						time.Date(1967, 10, 22, 4, 55, 00, 0, time.UTC),
					},
					RangePercentage: 0.05,
				},
			},
		},
		{
			name:     "float with three ranges, where each has custom range percentage and one is enum",
			dataType: "float",
			ranges: []*models.Params{
				{
					TypeParams: &models.ColumnFloatParams{
						BitWidth: 32,
						FromPtr:  float64Ptr(1.74354),
						ToPtr:    float64Ptr(2.92317),
					},
					RangePercentage: 0.85,
				},
				{
					TypeParams: &models.ColumnFloatParams{
						BitWidth: 64,
						FromPtr:  float64Ptr(3.11231),
						ToPtr:    float64Ptr(4.12312),
					},
					RangePercentage: 0.1,
				},
				{
					Values: []any{
						9.032005,
						22.101967,
					},
					RangePercentage: 0.05,
				},
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			column := &models.Column{Name: "ranges", Type: tc.dataType, Ranges: tc.ranges}

			cfg := oneColumnCfg(t, column)
			cfg.Models[UnitDefaultColumnName].RowsCount = UnitDefaultRowsCount

			handledDataRows := generateFunc(t, cfg)[UnitDefaultColumnName]
			require.Len(t, handledDataRows, UnitDefaultRowsCount)

			expectedValuesAmountPerRange := make(map[int]int, len(tc.ranges))

			for idx, r := range tc.ranges {
				expectedValuesAmountPerRange[idx] = int(math.Ceil(float64(UnitDefaultRowsCount) * r.RangePercentage))
			}

			for i := range handledDataRows {
				val := handledDataRows[i].Values[0]

				rangeIdx, err := mapValueToRange(tc.dataType, val, tc.ranges)
				require.NoError(t, err)

				expectedValuesAmountPerRange[rangeIdx]--
			}

			for idx := range tc.ranges {
				require.GreaterOrEqual(t, expectedValuesAmountPerRange[idx], 0, idx)
			}
		})
	}
}

//nolint:cyclop, gocognit
func mapValueToRange(columnType string, value any, ranges []*models.Params) (int, error) {
	for idx, r := range ranges {
		if r.Values != nil {
			if slices.Contains(r.Values, value) {
				return idx, nil
			}
		}

		switch columnType {
		case "integer":
			switch val := value.(type) {
			case int32:
				if int32(r.IntegerParams.From) <= val && val <= int32(r.IntegerParams.To) {
					return idx, nil
				}
			case int64:
				if r.IntegerParams.From <= val && val <= r.IntegerParams.To {
					return idx, nil
				}
			}
		case "string":
			strValue, ok := value.(string)
			if !ok {
				return -1, errors.Errorf("expected string, failed to cast: %v", value)
			}

			if r.StringParams.MinLength <= len([]rune(strValue)) && len([]rune(strValue)) <= r.StringParams.MaxLength {
				return idx, nil
			}
		case "datetime":
			timeValue, ok := value.(time.Time)
			if !ok {
				return -1, errors.Errorf("expected time.Time, failed to cast: %v", value)
			}

			if timeValue.Sub(r.DateTimeParams.From) > 0 && r.DateTimeParams.To.Sub(timeValue) > 0 {
				return idx, nil
			}
		case "float":
			switch val := value.(type) {
			case float32:
				if float32(r.FloatParams.From) <= val && val <= float32(r.FloatParams.To) {
					return idx, nil
				}
			case float64:
				if r.FloatParams.From <= val && val <= r.FloatParams.To {
					return idx, nil
				}
			}
		}
	}

	return -1, errors.Errorf("range not found for value: %v", value)
}
