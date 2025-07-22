package test

import (
	"context"
	"fmt"
	"sync/atomic"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/tarantool/sdvg/internal/generator/models"
	outputMock "github.com/tarantool/sdvg/internal/generator/output/mock"
	"github.com/tarantool/sdvg/internal/generator/usecase"
	usecaseGeneral "github.com/tarantool/sdvg/internal/generator/usecase/general"
)

const (
	BenchDefaultFieldsCount = 8
	BenchWorkersCount       = 64
	BenchBatchSize          = 1000
	BenchRandomSeed         = 1738591926070236604
)

func benchmarkFunc(b *testing.B, columns []*models.Column) {
	b.Helper()

	rowsHandled := atomic.Uint64{}

	uc := usecaseGeneral.NewUseCase(usecaseGeneral.UseCaseConfig{})

	cfg := &models.GenerationConfig{
		WorkersCount: BenchWorkersCount,
		BatchSize:    BenchBatchSize,
		RandomSeed:   BenchRandomSeed,
		Models: map[string]*models.Model{
			"test": {
				RowsCount: uint64(b.N),
				Columns:   columns,
			},
		},
		OutputConfig: &models.OutputConfig{
			Type: "devnull",
		},
	}

	require.NoError(b, cfg.Parse())
	cfg.FillDefaults()
	require.Empty(b, cfg.Validate())

	outputHandler := func(_ context.Context, _ string, rows []*models.DataRow) error {
		rowsHandled.Add(uint64(len(rows)))

		return nil
	}

	out := outputMock.NewOutput(outputHandler)

	b.ResetTimer()

	taskID, err := uc.CreateTask(
		context.Background(),
		usecase.TaskConfig{GenerationConfig: cfg, Output: out},
	)
	if err != nil {
		b.Fatal(err)
	}

	err = uc.WaitResult(taskID)
	if err != nil {
		b.Fatal(err)
	}

	b.StopTimer()

	require.Equal(b, uint64(b.N), rowsHandled.Load())

	b.ReportMetric(float64(b.N)/b.Elapsed().Seconds(), "rows/s")
	b.ReportMetric(float64(b.N)/b.Elapsed().Seconds()*float64(len(columns)), "values/s")
}

func getNamePrefix(ordered bool, ci bool) string {
	name := ""
	if ci {
		name += "CI/"
	}

	if ordered {
		name += "Ordered/"
	} else {
		name += "Random/"
	}

	return name
}

func BenchmarkInteger(b *testing.B) {
	testCases := []struct {
		bitWidth int
		ordered  bool
		ci       bool
	}{
		{8, false, false},
		{16, false, false},
		{32, false, true},
		{32, true, true},
		{64, false, false},
	}

	for _, testCase := range testCases {
		name := fmt.Sprintf(
			"%sbits-%v-cpu",
			getNamePrefix(testCase.ordered, testCase.ci),
			testCase.bitWidth,
		)

		b.Run(name, func(b *testing.B) {
			b.Helper()

			columns := make([]*models.Column, 0, BenchDefaultFieldsCount)
			for c := range BenchDefaultFieldsCount {
				columns = append(
					columns, &models.Column{
						Name: fmt.Sprintf("integer-%d", c), Type: "integer",
						Params: &models.Params{
							TypeParams: &models.ColumnIntegerParams{BitWidth: testCase.bitWidth},
							Ordered:    testCase.ordered,
						},
					},
				)
			}

			b.SetBytes(int64(BenchDefaultFieldsCount * testCase.bitWidth / 8))
			benchmarkFunc(b, columns)
		})
	}
}

func BenchmarkFloat(b *testing.B) {
	testCases := []struct {
		bitWidth int
		ordered  bool
		ci       bool
	}{
		{32, false, true},
		{32, true, false},
		{64, false, false},
	}

	for _, testCase := range testCases {
		name := fmt.Sprintf(
			"%sbits-%v-cpu",
			getNamePrefix(testCase.ordered, testCase.ci),
			testCase.bitWidth,
		)

		b.Run(name, func(b *testing.B) {
			b.Helper()

			columns := make([]*models.Column, 0, BenchDefaultFieldsCount)
			for c := range BenchDefaultFieldsCount {
				columns = append(
					columns, &models.Column{
						Name: fmt.Sprintf("float-%d", c), Type: "float",
						Params: &models.Params{
							TypeParams: &models.ColumnFloatParams{BitWidth: testCase.bitWidth},
							Ordered:    testCase.ordered,
						},
					},
				)
			}

			b.SetBytes(int64(BenchDefaultFieldsCount * testCase.bitWidth / 8))
			benchmarkFunc(b, columns)
		})
	}
}

func BenchmarkString(b *testing.B) {
	testCases := []struct {
		length  int
		ordered bool
		ci      bool
	}{
		{4, false, false},
		{8, false, false},
		{16, false, true},
		{16, true, false},
		{32, false, false},
		{128, false, false},
	}

	for _, testCase := range testCases {
		name := fmt.Sprintf(
			"%slength-%v-cpu",
			getNamePrefix(testCase.ordered, testCase.ci),
			testCase.length,
		)

		b.Run(name, func(b *testing.B) {
			b.Helper()

			columns := make([]*models.Column, 0, BenchDefaultFieldsCount)
			for c := range BenchDefaultFieldsCount {
				columns = append(
					columns, &models.Column{
						Name: fmt.Sprintf("string-%d", c), Type: "string",
						Params: &models.Params{
							TypeParams: &models.ColumnStringParams{
								MinLength: testCase.length, MaxLength: testCase.length,
							},
							Ordered: testCase.ordered,
						},
					},
				)
			}

			b.SetBytes(int64(BenchDefaultFieldsCount * testCase.length))
			benchmarkFunc(b, columns)
		})
	}
}

func BenchmarkText(b *testing.B) {
	testCases := []struct {
		length  int
		ordered bool
		ci      bool
	}{
		{16, false, false},
		{128, false, true},
		{128, true, false},
		{1024, false, false},
	}

	for _, testCase := range testCases {
		name := fmt.Sprintf(
			"%slength-%v-cpu",
			getNamePrefix(testCase.ordered, testCase.ci),
			testCase.length,
		)

		b.Run(name, func(b *testing.B) {
			b.Helper()

			columns := make([]*models.Column, 0, BenchDefaultFieldsCount)
			for c := range BenchDefaultFieldsCount {
				columns = append(
					columns, &models.Column{
						Name: fmt.Sprintf("string-%d", c), Type: "string",
						Params: &models.Params{
							TypeParams: &models.ColumnStringParams{
								MinLength: testCase.length, MaxLength: testCase.length,
							},
							Ordered: testCase.ordered, DistinctPercentage: 0,
						},
					},
				)
			}

			b.SetBytes(int64(BenchDefaultFieldsCount * testCase.length))
			benchmarkFunc(b, columns)
		})
	}
}

func BenchmarkUUID(b *testing.B) {
	testCases := []struct {
		ordered bool
		ci      bool
	}{
		{false, true},
		{true, false},
	}

	for _, testCase := range testCases {
		name := getNamePrefix(testCase.ordered, testCase.ci) + "cpu"

		b.Run(name, func(b *testing.B) {
			b.Helper()

			columns := make([]*models.Column, 0, BenchDefaultFieldsCount)
			for c := range BenchDefaultFieldsCount {
				columns = append(columns, &models.Column{
					Name: fmt.Sprintf("uuid-%d", c), Type: "uuid",
					Params: &models.Params{
						Ordered: testCase.ordered,
					},
				})
			}

			b.SetBytes(int64(BenchDefaultFieldsCount * 16))
			benchmarkFunc(b, columns)
		})
	}
}

func BenchmarkDateTime(b *testing.B) {
	testCases := []struct {
		ordered bool
		ci      bool
	}{
		{false, true},
		{true, false},
	}

	for _, testCase := range testCases {
		name := getNamePrefix(testCase.ordered, testCase.ci) + "cpu"

		b.Run(name, func(b *testing.B) {
			b.Helper()

			columns := make([]*models.Column, 0, BenchDefaultFieldsCount)
			for c := range BenchDefaultFieldsCount {
				columns = append(columns, &models.Column{
					Name: fmt.Sprintf("datetime-%d", c), Type: "datetime",
					Params: &models.Params{
						Ordered: testCase.ordered,
					},
				})
			}

			b.SetBytes(int64(BenchDefaultFieldsCount * 16))
			benchmarkFunc(b, columns)
		})
	}
}

var values = []any{nil, 1, 2}

func BenchmarkEnum(b *testing.B) {
	testCases := []struct {
		ordered bool
		ci      bool
	}{
		{false, true},
		{true, false},
	}

	for _, testCase := range testCases {
		name := getNamePrefix(testCase.ordered, testCase.ci) + "cpu"

		b.Run(name, func(b *testing.B) {
			b.Helper()

			columns := make([]*models.Column, 0, BenchDefaultFieldsCount)
			for c := range BenchDefaultFieldsCount {
				columns = append(columns, &models.Column{
					Name: fmt.Sprintf("enum-%d", c),
					Type: "integer",
					Params: &models.Params{
						Values:  values,
						Ordered: testCase.ordered,
					},
				})
			}

			b.SetBytes(int64(BenchDefaultFieldsCount * 16))
			benchmarkFunc(b, columns)
		})
	}
}

func BenchmarkRanges(b *testing.B) {
	testCases := []struct {
		dataType string
		ranges   []*models.Params
		ci       bool
	}{
		{
			"integer",
			[]*models.Params{
				{
					IntegerParams: &models.ColumnIntegerParams{
						BitWidth: 16,
						FromPtr:  int64Ptr(1),
						ToPtr:    int64Ptr(11),
					},
				},
				{
					IntegerParams: &models.ColumnIntegerParams{
						BitWidth: 32,
						FromPtr:  int64Ptr(100),
						ToPtr:    int64Ptr(1100),
					},
				},
				{
					Values: []any{
						nil,
						999,
					},
				},
			},
			true,
		},
	}

	for _, testCase := range testCases {
		name := getNamePrefix(false, testCase.ci) + "cpu"

		b.Run(name, func(b *testing.B) {
			b.Helper()

			columns := make([]*models.Column, 0, BenchDefaultFieldsCount)
			for c := range BenchDefaultFieldsCount {
				columns = append(columns, &models.Column{
					Name:   fmt.Sprintf("enum-%d", c),
					Type:   testCase.dataType,
					Ranges: testCase.ranges,
				})
			}

			b.SetBytes(int64(BenchDefaultFieldsCount * 16))
			benchmarkFunc(b, columns)
		})
	}
}
