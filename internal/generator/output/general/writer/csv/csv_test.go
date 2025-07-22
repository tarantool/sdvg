package csv

import (
	"context"
	"encoding/csv"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"github.com/tarantool/sdvg/internal/generator/common"
	"github.com/tarantool/sdvg/internal/generator/models"
)

func TestWriteRow(t *testing.T) {
	testModel := &models.Model{
		Name: "test",
		Columns: []*models.Column{
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
				Name: "datetime",
				Type: "datetime",
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
		},
	}

	uuidValue := uuid.New()
	dateTimeValue := time.Now()

	testData := []*models.DataRow{
		{
			Values: []any{int64(10), float32(5), "firstValue", dateTimeValue, uuidValue, nil},
		},
		{
			Values: []any{int64(1032), float32(543.13), "secondValue", dateTimeValue, uuidValue, nil},
		},
	}

	csvConfig := &models.CSVConfig{
		FloatPrecision: 2,
		DatetimeFormat: "2006-01-02",
		Delimiter:      ",",
	}

	type testCase struct {
		name           string
		model          *models.Model
		rowsCount      uint64
		rowsPerFile    uint64
		withoutHeaders bool
		data           []*models.DataRow
		expectedData   [][]string
	}

	testCases := []testCase{
		{
			name:           "Rows per file equals rows count",
			model:          testModel,
			rowsCount:      2,
			rowsPerFile:    2,
			withoutHeaders: true,
			data:           testData,
			expectedData: [][]string{
				{"10", "5.00", "firstValue", dateTimeValue.Format("2006-01-02"), uuidValue.String(), ""},
				{"1032", "543.13", "secondValue", dateTimeValue.Format("2006-01-02"), uuidValue.String(), ""},
			},
		},
		{
			name:           "Rows per file not equals rows count",
			model:          testModel,
			rowsCount:      2,
			rowsPerFile:    1,
			withoutHeaders: true,
			data:           testData,
			expectedData: [][]string{
				{"10", "5.00", "firstValue", dateTimeValue.Format("2006-01-02"), uuidValue.String(), ""},
				{"1032", "543.13", "secondValue", dateTimeValue.Format("2006-01-02"), uuidValue.String(), ""},
			},
		},
		{
			name:           "With headers",
			model:          testModel,
			rowsCount:      2,
			rowsPerFile:    2,
			withoutHeaders: false,
			data:           testData,
			expectedData: [][]string{
				{"int64", "float32", "string", "datetime", "uuid", "nil"},
				{"10", "5.00", "firstValue", dateTimeValue.Format("2006-01-02"), uuidValue.String(), ""},
				{"1032", "543.13", "secondValue", dateTimeValue.Format("2006-01-02"), uuidValue.String(), ""},
			},
		},
	}

	testFunc := func(t *testing.T, tc testCase) {
		t.Helper()

		tc.model.RowsCount = tc.rowsCount
		tc.model.RowsPerFile = tc.rowsPerFile

		csvConfig.WithoutHeaders = tc.withoutHeaders

		csvWriter := NewWriter(context.Background(), tc.model, csvConfig, "./", false, nil)

		err := csvWriter.Init()
		require.NoError(t, err)

		for _, row := range tc.data {
			err = csvWriter.WriteRow(row)
			require.NoError(t, err)
		}

		err = csvWriter.Teardown()
		require.NoError(t, err)

		filesAmountExpected := getFileNumber(len(tc.data), int(tc.model.RowsPerFile))
		filesRows := make(map[string][][]string, filesAmountExpected)
		fileNamesOrdered := make([]string, 0, len(filesRows))

		for i := range filesAmountExpected {
			fileName := csvWriter.getFileName(uint64(i))
			fileNamesOrdered = append(fileNamesOrdered, fileName)

			file, err := os.OpenFile(fileName, os.O_RDWR, 0644)
			require.NoError(t, err)

			defer func() {
				require.NoError(t, file.Close())
				require.NoError(t, os.Remove(fileName))
			}()

			reader := csv.NewReader(file)
			records, err := reader.ReadAll()
			require.NoError(t, err)

			filesRows[fileName] = records
		}

		require.Len(t, filesRows, filesAmountExpected)

		var i int

		for _, fileName := range fileNamesOrdered {
			for _, row := range filesRows[fileName] {
				require.Equal(t, tc.expectedData[i], row)

				i++
			}
		}
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) { testFunc(t, tc) })
	}
}

func TestParseDataRow(t *testing.T) {
	type testCase struct {
		name     string
		row      *models.DataRow
		expected []string
	}

	testCases := []testCase{
		{
			name: "Basic types",
			row: &models.DataRow{
				Values: []any{"string", 123, 45.678, true, nil},
			},
			expected: []string{"string", "123", "45.68", "true", ""},
		},
		{
			name: "Time and UUID",
			row: &models.DataRow{
				Values: []any{
					time.Date(2023, 10, 1, 0, 0, 0, 0, time.UTC),
					uuid.MustParse("123e4567-e89b-12d3-a456-426614174000"),
				},
			},
			expected: []string{"2023-10-01", "123e4567-e89b-12d3-a456-426614174000"},
		},
	}

	testFunc := func(t *testing.T, tc testCase) {
		t.Helper()

		csvWriter := &Writer{
			config: &models.CSVConfig{
				FloatPrecision: 2,
				DatetimeFormat: "2006-01-02",
			},
		}

		result, err := csvWriter.parseDataRow(tc.row)
		require.NoError(t, err)
		require.Equal(t, tc.expected, result)
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) { testFunc(t, tc) })
	}
}

func TestWriteToCorrectFiles(t *testing.T) {
	type testCase struct {
		name          string
		rowsPerFile   uint64
		rows          []*models.DataRow
		writersCount  int
		expectedFiles []string
		expectedData  map[string][][]string
	}

	config := &models.CSVConfig{
		Delimiter: ",",
	}

	model := &models.Model{
		Name: "test",
		Columns: []*models.Column{
			{
				Name: "id",
				Type: "integer",
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
			name:          "Rows per file equals rows count",
			rowsPerFile:   uint64(len(rows)),
			rows:          rows,
			writersCount:  5,
			expectedFiles: []string{"test_0.csv"},
			expectedData: map[string][][]string{
				"test_0.csv": {{"id"}, {"1"}, {"2"}, {"3"}, {"4"}, {"5"}, {"6"}, {"7"}, {"8"}, {"9"}},
			},
		},
		{
			name:          "Rows per file less than rows count",
			rowsPerFile:   2,
			rows:          rows,
			writersCount:  3,
			expectedFiles: []string{"test_0.csv", "test_1.csv", "test_2.csv", "test_3.csv", "test_4.csv"},
			expectedData: map[string][][]string{
				"test_0.csv": {{"id"}, {"1"}, {"2"}},
				"test_1.csv": {{"id"}, {"3"}, {"4"}},
				"test_2.csv": {{"id"}, {"5"}, {"6"}},
				"test_3.csv": {{"id"}, {"7"}, {"8"}},
				"test_4.csv": {{"id"}, {"9"}},
			},
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

		write := func(from, to int, continueGeneration bool) {
			writer := NewWriter(context.Background(), model, config, dir, continueGeneration, nil)
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

		files, err := common.WalkWithFilter(dir, func(e os.DirEntry) bool {
			return !e.IsDir() && filepath.Ext(e.Name()) == ".csv"
		})
		require.NoError(t, err)

		require.ElementsMatch(t, tc.expectedFiles, files)

		for _, fileName := range files {
			file, err := os.Open(filepath.Join(dir, fileName))
			require.NoError(t, err)

			csvReader := csv.NewReader(file)

			actualData, err := csvReader.ReadAll()
			require.NoError(t, err)

			require.Equal(t, tc.expectedData[fileName], actualData)
		}
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
