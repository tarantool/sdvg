package common

import (
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

func TestParseData(t *testing.T) {
	type testData struct {
		Name string `json:"name"`
		Age  int    `json:"age"`
	}

	type testCase struct {
		name     string
		data     any
		expected *testData
		hasError bool
	}

	testCases := []testCase{
		{
			name: "Valid data",
			data: map[string]any{
				"name": "John",
			},
			expected: &testData{Name: "John", Age: 0},
			hasError: false,
		},
		{
			name:     "Invalid data type",
			data:     "invalid",
			expected: &testData{},
			hasError: true,
		},
	}

	testFunc := func(t *testing.T, tc testCase) {
		t.Helper()

		result, err := AnyToStruct[testData](tc.data)
		if tc.hasError {
			require.Error(t, err)
		} else {
			require.NoError(t, err)
			require.Equal(t, tc.expected, result)
		}
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) { testFunc(t, tc) })
	}
}

func TestGetKey(t *testing.T) {
	type testCase struct {
		name       string
		modelName  string
		columnName string
		expected   string
	}

	testCases := []testCase{
		{
			name:       "Basic test",
			modelName:  "User",
			columnName: "ID",
			expected:   "User.ID",
		},
		{
			name:       "Empty model and column name",
			modelName:  "",
			columnName: "",
			expected:   "?.?",
		},
		{
			name:       "Empty model name",
			modelName:  "",
			columnName: "ID",
			expected:   "?.ID",
		},
		{
			name:       "Empty column name",
			modelName:  "User",
			columnName: "",
			expected:   "User.?",
		},
	}

	testFunc := func(t *testing.T, tc testCase) {
		t.Helper()

		result := GetKey(tc.modelName, tc.columnName)
		require.Equal(t, tc.expected, result)
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) { testFunc(t, tc) })
	}
}

func TestTrim(t *testing.T) {
	type testCase struct {
		name     string
		prefix   string
		suffix   string
		content  string
		expected string
	}

	testCases := []testCase{
		{
			name:     "Successful trim",
			prefix:   "'",
			suffix:   "'",
			content:  "'test'",
			expected: "test",
		},
		{
			name:     "Successful trim with repeat prefix or suffix",
			prefix:   "'",
			suffix:   "'",
			content:  "'test''",
			expected: "test",
		},
		{
			name:     "Only prefix",
			prefix:   "'",
			suffix:   "|",
			content:  "'test'",
			expected: "'test'",
		},
		{
			name:     "Only suffix",
			prefix:   "|",
			suffix:   "'",
			content:  "'test'",
			expected: "'test'",
		},
	}

	testFunc := func(t *testing.T, tc testCase) {
		t.Helper()

		actual := Trim(tc.content, tc.prefix, tc.suffix)

		require.Equal(t, tc.expected, actual)
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) { testFunc(t, tc) })
	}
}

func TestGetIndicesInOrder(t *testing.T) {
	type testCase struct {
		name         string
		original     []string
		specialOrder []string
		expected     []int
	}

	testCases := []testCase{
		{
			name:         "full reorder",
			original:     []string{"I", "am", "hungry", "We", "must", "order", "pizza"},
			specialOrder: []string{"hungry", "I", "am", "pizza", "We", "must", "order"}, // in Yoda notation
			expected:     []int{2, 0, 1, 6, 3, 4, 5},
		},
		{
			name:         "full wrong elements",
			original:     []string{"I", "am", "hungry", "We", "must", "order", "pizza"},
			specialOrder: []string{"Why", "are", "you", "hey", "?"},
			expected:     []int{-1, -1, -1, -1, -1},
		},
		{
			name:         "partly wrong elements",
			original:     []string{"I", "am", "hungry", "We", "must", "order", "pizza"},
			specialOrder: []string{"I", "don't", "want", "to", "order", "pizza"},
			expected:     []int{0, -1, -1, -1, 5, 6},
		},
		{
			name:         "empty",
			original:     []string{"I", "am", "hungry", "We", "must", "order", "pizza"},
			specialOrder: []string{},
			expected:     []int{},
		},
	}

	testFunc := func(t *testing.T, tc testCase) {
		t.Helper()

		actual := GetIndicesInOrder(tc.original, tc.specialOrder)

		require.Equal(t, tc.expected, actual)
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) { testFunc(t, tc) })
	}
}

func TestShiftElementsToEnd(t *testing.T) {
	type testCase struct {
		name                 string
		original             []int
		orderedObjectsToMove []int
		expected             []int
	}

	testCases := []testCase{
		{
			name:                 "all shifted values are presented in original",
			original:             []int{1, 2, 3, 4, 5},
			orderedObjectsToMove: []int{3, 2, 1},
			expected:             []int{4, 5, 3, 2, 1},
		},
		{
			name:                 "shifted values are presented in original partly",
			original:             []int{1, 2, 3, 4, 5},
			orderedObjectsToMove: []int{100, 2, 1},
			expected:             []int{3, 4, 5, 2, 1},
		},
		{
			name:                 "shifted values are not presented in original",
			original:             []int{1, 2, 3, 4, 5},
			orderedObjectsToMove: []int{100, 200, 300},
			expected:             []int{1, 2, 3, 4, 5},
		},
		{
			name:                 "empty original",
			original:             []int{},
			orderedObjectsToMove: []int{100, 200, 300},
			expected:             []int{},
		},
		{
			name:                 "empty shifted values",
			original:             []int{1, 2, 3, 4, 5},
			orderedObjectsToMove: []int{},
			expected:             []int{1, 2, 3, 4, 5},
		},
	}

	testFunc := func(t *testing.T, tc testCase) {
		t.Helper()

		actual := ShiftElementsToEnd(
			tc.original,
			tc.orderedObjectsToMove,
			func(e int) int { return e },
		)

		require.Equal(t, tc.expected, actual)
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) { testFunc(t, tc) })
	}
}

func TestSortSlice(t *testing.T) {
	type testCase struct {
		name    string
		input   []any
		want    []any
		cmpFunc CmpFunc
	}

	firstUUID, err := uuid.Parse("99aa4717-e6a0-4adc-b2c8-ba505e7ffc00")
	require.NoError(t, err)

	secondUUID, err := uuid.Parse("7867d43c-91cc-481a-820b-3eb7c2dede86")
	require.NoError(t, err)

	testCases := []testCase{
		{
			name:  "both nil",
			input: []any{nil, nil},
			want:  []any{nil, nil},
		},
		{
			name:  "nil first",
			input: []any{nil, 42},
			want:  []any{nil, 42},
		},
		{
			name:  "nil last",
			input: []any{100, nil},
			want:  []any{nil, 100},
		},
		{
			name:    "mixed nil",
			input:   []any{"b", nil, "a"},
			want:    []any{nil, "a", "b"},
			cmpFunc: CmpString,
		},
		{
			name:    "int simple",
			input:   []any{3, 1, 2},
			want:    []any{1, 2, 3},
			cmpFunc: CmpInt,
		},
		{
			name:    "mixed int types",
			input:   []any{int32(100), int64(50), int16(75)},
			want:    []any{int64(50), int16(75), int32(100)},
			cmpFunc: CmpInt,
		},
		{
			name:    "negative numbers",
			input:   []any{-1, 0, -2},
			want:    []any{-2, -1, 0},
			cmpFunc: CmpInt,
		},
		{
			name:    "float64 simple",
			input:   []any{3.14, 1.1, 2.2},
			want:    []any{1.1, 2.2, 3.14},
			cmpFunc: CmpFloat,
		},
		{
			name:    "mixed float types",
			input:   []any{float32(3.0), 3.14},
			want:    []any{float32(3.0), 3.14},
			cmpFunc: CmpFloat,
		},
		{
			name:    "string simple",
			input:   []any{"c", "a", "b"},
			want:    []any{"a", "b", "c"},
			cmpFunc: CmpString,
		},
		{
			name:    "case sensitive",
			input:   []any{"A", "b", "a"},
			want:    []any{"A", "a", "b"},
			cmpFunc: CmpString,
		},
		{
			name:    "UUID",
			input:   []any{firstUUID, secondUUID},
			want:    []any{secondUUID, firstUUID},
			cmpFunc: CmpUUID,
		},
		{
			name: "time simple",
			input: []any{
				time.Date(2001, 12, 31, 23, 59, 59, 999999999, time.UTC),
				time.Date(2000, 1, 1, 0, 0, 0, 0, time.UTC),
			},
			want: []any{
				time.Date(2000, 1, 1, 0, 0, 0, 0, time.UTC),
				time.Date(2001, 12, 31, 23, 59, 59, 999999999, time.UTC),
			},
			cmpFunc: CmpTime,
		},
	}

	testFunc := func(t *testing.T, tc testCase) {
		t.Helper()

		SortSlice(tc.input, tc.cmpFunc)
		require.Equal(t, tc.want, tc.input)
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) { testFunc(t, tc) })
	}
}

func TestConvertToInt(t *testing.T) {
	type testCase struct {
		name       string
		input      any
		targetType reflect.Type
		expected   any
		expectErr  bool
	}

	testCases := []testCase{
		{"int to int8", 42, reflect.TypeFor[int8](), int8(42), false},
		{"int8 to int16", int8(42), reflect.TypeFor[int16](), int16(42), false},
		{"int16 to int32", int16(42), reflect.TypeFor[int32](), int32(42), false},
		{"int32 to int64", int16(42), reflect.TypeFor[int64](), int64(42), false},

		{"float64 to int (valid)", 42.0, reflect.TypeFor[int](), 42, false},
		{"float64 to int (invalid)", 42.5, reflect.TypeFor[int](), nil, true},

		{"string to int", "42", reflect.TypeFor[int](), 42, false},
		{"string to int (invalid)", "hello", reflect.TypeFor[int](), nil, true},

		{"unsupported type", true, reflect.TypeFor[int](), nil, true},
	}

	testFunc := func(t *testing.T, tc testCase) {
		t.Helper()

		result, err := ConvertToInt(tc.input, tc.targetType)
		require.Equal(t, tc.expectErr, err != nil)
		require.Equal(t, tc.expected, result)
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) { testFunc(t, tc) })
	}
}

func TestConvertToFloat(t *testing.T) {
	type testCase struct {
		name       string
		input      any
		targetType reflect.Type
		expected   any
		expectErr  bool
	}

	testCases := []testCase{
		{"int to float32", 42, reflect.TypeFor[float32](), float32(42), false},
		{"int8 to float64", int8(42), reflect.TypeFor[float64](), float64(42), false},
		{"float32 to float64", float32(42.5), reflect.TypeFor[float64](), 42.5, false},

		{"string to float32", "42.5", reflect.TypeFor[float32](), float32(42.5), false},
		{"string to float64", "3.1415", reflect.TypeFor[float64](), 3.1415, false},
		{"string to float (invalid)", "hello", reflect.TypeFor[float64](), nil, true},

		{"bool to float (unsupported)", true, reflect.TypeFor[float64](), nil, true},
	}

	testFunc := func(t *testing.T, tc testCase) {
		t.Helper()

		result, err := ConvertToFloat(tc.input, tc.targetType)
		require.Equal(t, tc.expectErr, err != nil)
		require.Equal(t, tc.expected, result)
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) { testFunc(t, tc) })
	}
}

func TestConvertToString(t *testing.T) {
	type testCase struct {
		name       string
		input      any
		targetType reflect.Type
		expected   any
		expectErr  bool
	}

	testCases := []testCase{
		{"string to string", "hello", reflect.TypeFor[string](), "hello", false},
		{"int to string", 42, reflect.TypeFor[string](), "42", false},
		{"float64 to string", 3.1415, reflect.TypeFor[string](), "3.1415", false},
		{"bool to string", true, reflect.TypeFor[string](), "true", false},
		{"struct to string", struct{ Name string }{"test"}, reflect.TypeFor[string](), "{test}", false},
	}

	testFunc := func(t *testing.T, tc testCase) {
		t.Helper()

		result, err := ConvertToString(tc.input, tc.targetType)
		require.Equal(t, tc.expectErr, err != nil)
		require.Equal(t, tc.expected, result)
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) { testFunc(t, tc) })
	}
}

func TestConvertToTime(t *testing.T) {
	type testCase struct {
		name       string
		input      any
		targetType reflect.Type
		expected   any
		expectErr  bool
	}

	refTime := time.Date(2024, time.March, 26, 12, 0, 0, 0, time.UTC)
	refTimeStr := refTime.Format(time.RFC3339)

	testCases := []testCase{
		{
			"time.Time to time.Time",
			refTime,
			reflect.TypeFor[time.Time](),
			refTime,
			false,
		},
		{
			"RFC3339 string to time.Time",
			refTimeStr,
			reflect.TypeFor[time.Time](),
			refTime,
			false,
		},
		{
			"invalid string to time.Time",
			"invalid-time",
			reflect.TypeFor[time.Time](),
			nil,
			true,
		},
		{"int64 timestamp to time.Time",
			refTime.Unix(),
			reflect.TypeFor[time.Time](),
			time.Unix(refTime.Unix(), 0),
			false,
		},
		{
			"float64 timestamp to time.Time",
			float64(refTime.Unix()),
			reflect.TypeFor[time.Time](),
			time.Unix(refTime.Unix(), 0),
			false,
		},

		{
			"unsupported type",
			true,
			reflect.TypeFor[time.Time](),
			nil,
			true,
		},
	}

	testFunc := func(t *testing.T, tc testCase) {
		t.Helper()

		result, err := ConvertToTime(tc.input, tc.targetType)
		require.Equal(t, tc.expectErr, err != nil)
		require.Equal(t, tc.expected, result)
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) { testFunc(t, tc) })
	}
}

func TestConvertToUUID(t *testing.T) {
	type testCase struct {
		name       string
		input      any
		targetType reflect.Type
		expected   any
		expectErr  bool
	}

	validUUID := uuid.New()
	validUUIDStr := validUUID.String()
	invalidUUIDStr := "invalid-uuid"

	testCases := []testCase{
		{"UUID to UUID", validUUID, reflect.TypeFor[uuid.UUID](), validUUID, false},
		{"valid string to UUID", validUUIDStr, reflect.TypeFor[uuid.UUID](), validUUID, false},
		{"invalid string to UUID", invalidUUIDStr, reflect.TypeFor[uuid.UUID](), nil, true},
		{"unsupported type", 42, reflect.TypeFor[uuid.UUID](), nil, true},
	}

	testFunc := func(t *testing.T, tc testCase) {
		t.Helper()

		result, err := ConvertToUUID(tc.input, tc.targetType)
		require.Equal(t, tc.expectErr, err != nil)
		require.Equal(t, tc.expected, result)
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) { testFunc(t, tc) })
	}
}

func createTempFiles(t *testing.T, dir string, files []string, dirs []string) {
	t.Helper()

	for _, name := range files {
		path := filepath.Join(dir, name)
		_, err := os.Create(path)
		require.NoError(t, err)
	}

	for _, name := range dirs {
		path := filepath.Join(dir, name)
		err := os.Mkdir(path, os.ModePerm)
		require.NoError(t, err)
	}
}

func TestWalkWithFilter(t *testing.T) {
	type variant int

	const (
		dirExist variant = iota
		dirNotExist
		isNotDir
	)

	type testCase struct {
		name       string
		variant    variant
		filterFunc func(entry os.DirEntry) bool
		files      []string
		dirs       []string
		expected   []string
		wantErr    bool
	}

	testCases := []testCase{
		{
			name:    "Find files with extension 'txt'",
			variant: dirExist,
			filterFunc: func(e os.DirEntry) bool {
				return e.IsDir() == false && filepath.Ext(e.Name()) == ".txt"
			},
			files:    []string{"a.txt", "b.txt", "c.go", "d"},
			dirs:     []string{"a", "b"},
			expected: []string{"a.txt", "b.txt"},
			wantErr:  false,
		},
		{
			name:    "Find files with prefix '001'",
			variant: dirExist,
			filterFunc: func(e os.DirEntry) bool {
				return !e.IsDir() && strings.HasPrefix(e.Name(), "001")
			},
			files:    []string{"001_a.txt", "001_b.txt", "002_c.go", "d"},
			dirs:     []string{"001_a", "001_bb"},
			expected: []string{"001_a.txt", "001_b.txt"},
			wantErr:  false,
		},
		{
			name:    "Find dirs with prefix '100'",
			variant: dirExist,
			filterFunc: func(e os.DirEntry) bool {
				return e.IsDir() && strings.HasPrefix(e.Name(), "100")
			},
			files:    []string{"001_a.txt", "001_b.txt", "002_c.go", "d"},
			dirs:     []string{"100_a", "100_b"},
			expected: []string{"100_a", "100_b"},
			wantErr:  false,
		},
		{
			name:       "Directory doesn't exist",
			variant:    dirNotExist,
			filterFunc: nil,
			files:      nil,
			dirs:       nil,
			expected:   nil,
			wantErr:    true,
		},
		{
			name:       "Dir is not directory",
			variant:    isNotDir,
			filterFunc: nil,
			files:      nil,
			dirs:       nil,
			expected:   nil,
			wantErr:    true,
		},
	}

	testFunc := func(t *testing.T, tc testCase) {
		t.Helper()

		var dir string

		switch tc.variant {
		case dirExist:
			dir = t.TempDir()
			createTempFiles(t, dir, tc.files, tc.dirs)
		case isNotDir:
			file, err := os.Create("test")
			require.NoError(t, err)
			defer os.Remove(file.Name())

			dir = file.Name()
		case dirNotExist:
		}

		actual, err := WalkWithFilter(dir, tc.filterFunc)
		require.Equal(t, tc.wantErr, err != nil)

		require.Equal(t, len(tc.expected), len(actual))
		require.ElementsMatch(t, tc.expected, actual)
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) { testFunc(t, tc) })
	}
}

func TestExtractValuesFromTemplate(t *testing.T) {
	type testCase struct {
		name     string
		template string
		expected []string
	}

	testCases := []testCase{
		{
			name:     "Empty template",
			template: "",
			expected: []string{},
		},
		{
			name:     "Valid template",
			template: "{{ foo }}.{{boo}}",
			expected: []string{"foo", "boo"},
		},
		{
			name:     "Template with filters",
			template: "{{ foo | upper | lower }}",
			expected: []string{"foo"},
		},
		{
			name:     "Template with functions",
			template: "{{ upper('foo') | lower }}@{{ boo }}",
			expected: []string{"boo"},
		},
		{
			name:     "Invalid template",
			template: "{_{ foo }}",
			expected: []string{},
		},
	}

	testFunc := func(t *testing.T, tc testCase) {
		t.Helper()

		actual := ExtractValuesFromTemplate(tc.template)
		require.Equal(t, tc.expected, actual)
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) { testFunc(t, tc) })
	}
}

func TestTopologicalSort(t *testing.T) {
	type node struct {
		name string
		deps []string
	}

	type testCase struct {
		name             string
		items            []node
		wantErr          bool
		wantDependencies bool
		expected         []string
	}

	testCases := []testCase{
		{
			name:             "Empty items",
			items:            []node{},
			wantErr:          false,
			wantDependencies: false,
			expected:         []string{},
		},
		{
			name: "Items with dependencies",
			items: []node{
				{name: "1", deps: []string{"3"}},
				{name: "2", deps: []string{"4"}},
				{name: "3", deps: []string{"2"}},
				{name: "4", deps: []string{}},
			},
			wantErr:          false,
			wantDependencies: true,
			expected:         []string{"4", "2", "3", "1"},
		},
		{
			name: "Items without dependencies",
			items: []node{
				{name: "1", deps: []string{}},
				{name: "2", deps: []string{}},
				{name: "3", deps: []string{}},
			},
			wantErr:          false,
			wantDependencies: false,
			expected:         []string{"1", "2", "3"},
		},
		{
			name: "Items with cycle dependencies",
			items: []node{
				{name: "1", deps: []string{"2"}},
				{name: "2", deps: []string{"1"}},
			},
			wantErr:          true,
			wantDependencies: false,
			expected:         nil,
		},
	}

	testFunc := func(t *testing.T, tc testCase) {
		t.Helper()

		actual, hasDependencies, err := TopologicalSort(tc.items, func(node node) (string, []string) {
			return node.name, node.deps
		})

		require.Equal(t, tc.wantErr, err != nil)
		require.Equal(t, tc.wantDependencies, hasDependencies)
		require.Equal(t, tc.expected, actual)
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) { testFunc(t, tc) })
	}
}
