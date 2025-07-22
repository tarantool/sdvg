package general

import (
	"context"
	"path/filepath"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"github.com/tarantool/sdvg/internal/generator/common"
	"github.com/tarantool/sdvg/internal/generator/models"
)

var (
	defaultColumns = []*models.Column{
		{
			Name: "int64",
			Type: "integer",
			Ranges: []*models.Params{
				{IntegerParams: &models.ColumnIntegerParams{BitWidth: 64}},
			},
		},
		{
			Name: "float32",
			Type: "float",
			Ranges: []*models.Params{
				{FloatParams: &models.ColumnFloatParams{BitWidth: 32}},
			},
		},
		{
			Name: "string",
			Type: "string",
		},
		{
			Name: "uuid",
			Type: "uuid",
		},
		{
			Name: "nil",
			Type: "string",
			Ranges: []*models.Params{
				{NullPercentage: 1},
			},
		},
		{
			Name: "datetime",
			Type: "datetime",
		},
	}

	uuidValue      = uuid.New()
	dateTimeValue1 = time.Date(2002, time.February, 27, 14, 0, 0, 0, &time.Location{})
	dateTimeValue2 = time.Date(1995, time.February, 17, 14, 0, 0, 0, &time.Location{})
)

func TestPartitionPaths(t *testing.T) {
	dir := t.TempDir()

	type expectedValues struct {
		PartitionColumnsIndexes []int
		PartitionPaths          map[string]struct{}
		WittenRows              []*models.DataRow
	}

	type testCase struct {
		name     string
		model    *models.Model
		data     []*models.DataRow
		expected *expectedValues
	}

	dataToWrite := []*models.DataRow{
		{Values: []any{int64(66), float32(5), "firstValue", uuidValue, nil, dateTimeValue1}},
		{Values: []any{int64(69), float32(543.13), "secondValue", uuidValue, nil, dateTimeValue2}},
	}

	testCases := []testCase{
		{
			name: "without partition",
			model: &models.Model{
				Name:             "flat_model",
				RowsCount:        2,
				RowsPerFile:      2,
				Columns:          defaultColumns,
				PartitionColumns: make([]*models.PartitionColumn, 0),
			},
			data: dataToWrite,
			expected: &expectedValues{
				PartitionPaths: map[string]struct{}{
					dir: {},
				},
				PartitionColumnsIndexes: make([]int, 0),
				WittenRows:              dataToWrite,
			},
		},
		{
			name: "1 writable partition column",
			model: &models.Model{
				Name:        "writable_partition",
				RowsCount:   2,
				RowsPerFile: 2,
				Columns:     defaultColumns,
				PartitionColumns: []*models.PartitionColumn{
					{
						Name:          "int64",
						WriteToOutput: true,
					},
				},
			},
			data: dataToWrite,
			expected: &expectedValues{
				PartitionPaths: map[string]struct{}{
					filepath.Join(dir, "int64=66"): {},
					filepath.Join(dir, "int64=69"): {},
				},
				PartitionColumnsIndexes: []int{0},
				WittenRows:              dataToWrite,
			},
		},
		{
			name: "1 writable partition column and 1 non writable",
			model: &models.Model{
				Name:        "mixed_partition",
				RowsCount:   2,
				RowsPerFile: 2,
				Columns:     defaultColumns,
				PartitionColumns: []*models.PartitionColumn{
					{
						Name:          "datetime",
						WriteToOutput: false,
					},
					{
						Name:          "string",
						WriteToOutput: true,
					},
				},
			},
			data: dataToWrite,
			expected: &expectedValues{
				PartitionPaths: map[string]struct{}{
					filepath.Join(dir, "datetime=2002-02-27 14:00:00 +0000 UTC", "string=firstValue"):  {},
					filepath.Join(dir, "datetime=1995-02-17 14:00:00 +0000 UTC", "string=secondValue"): {},
				},
				PartitionColumnsIndexes: []int{5, 2},
				WittenRows: []*models.DataRow{
					{Values: []any{int64(66), float32(5), "firstValue", uuidValue, nil}},
					{Values: []any{int64(69), float32(543.13), "secondValue", uuidValue, nil}},
				},
			},
		},
		{
			name: "1 float partition column",
			model: &models.Model{
				Name:        "writable_partition_float",
				RowsCount:   2,
				RowsPerFile: 2,
				Columns:     defaultColumns,
				PartitionColumns: []*models.PartitionColumn{
					{
						Name:          "float32",
						WriteToOutput: true,
					},
				},
			},
			data: dataToWrite,
			expected: &expectedValues{
				PartitionPaths: map[string]struct{}{
					filepath.Join(dir, "float32=5"):      {},
					filepath.Join(dir, "float32=543.13"): {},
				},
				PartitionColumnsIndexes: []int{1},
				WittenRows:              dataToWrite,
			},
		},
		{
			name: "1 nil partition column",
			model: &models.Model{
				Name:        "nil_partition",
				RowsCount:   2,
				RowsPerFile: 2,
				Columns:     defaultColumns,
				PartitionColumns: []*models.PartitionColumn{
					{
						Name:          "nil",
						WriteToOutput: true,
					},
				},
			},
			data: dataToWrite,
			expected: &expectedValues{
				PartitionPaths: map[string]struct{}{
					filepath.Join(dir, "nil=null"): {},
				},
				PartitionColumnsIndexes: []int{4},
				WittenRows:              dataToWrite,
			},
		},
	}

	testFunc := func(t *testing.T, tCase testCase) {
		t.Helper()

		writeMutex := &sync.Mutex{}
		writtenRows := make([]*models.DataRow, 0, len(tCase.data))

		devnullConfig := &models.OutputConfig{
			Type:               "devnull",
			Dir:                dir,
			CheckpointInterval: time.Second,
			DevNullParams: &models.DevNullConfig{
				Handler: func(row *models.DataRow, _ string) error {
					writeMutex.Lock()
					writtenRows = append(writtenRows, row)
					writeMutex.Unlock()

					return nil
				},
			},
		}

		writer, err := newModelWriter(tCase.model, devnullConfig, false)
		require.NoError(t, err)

		err = writer.WriteRows(context.Background(), tCase.data)
		require.NoError(t, err)

		actualValues := &expectedValues{
			PartitionPaths:          common.MakeSet(writer.writerByPartition),
			PartitionColumnsIndexes: writer.partitionColumnsIndexes,
			WittenRows:              writtenRows,
		}

		require.Equal(t, tCase.expected, actualValues)

		require.NoError(t, writer.Teardown())
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) { testFunc(t, tc) })
	}
}
