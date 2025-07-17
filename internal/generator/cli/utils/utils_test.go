package utils

import (
	"strconv"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestValidateFileFormat(t *testing.T) {
	type testCase struct {
		name                 string
		formats              []string
		content              string
		expectedError        bool
		expectedErrorMessage string
	}

	testCases := []testCase{
		{
			name:          "All formats allowed",
			formats:       []string{},
			content:       "file.yaml",
			expectedError: false,
		},
		{
			name:          "YAML format allowed",
			formats:       []string{".yaml"},
			content:       "file.yaml",
			expectedError: false,
		},
		{
			name:                 "JSON format allowed",
			formats:              []string{".json"},
			content:              "file.yaml",
			expectedError:        true,
			expectedErrorMessage: "invalid file extension, supported: [.json]",
		},
	}

	testFunc := func(t *testing.T, tc testCase) {
		t.Helper()

		fn := ValidateFileFormat(tc.formats...)

		err := fn(tc.content)

		require.Equal(t, tc.expectedError, err != nil)

		if tc.expectedError {
			require.ErrorContains(t, err, tc.expectedErrorMessage)
		}
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) { testFunc(t, tc) })
	}
}

func TestValidateEmptyString(t *testing.T) {
	type testCase struct {
		name                 string
		content              string
		expectedError        bool
		expectedErrorMessage string
	}

	testCases := []testCase{
		{
			name:                 "Empty string",
			content:              "",
			expectedError:        true,
			expectedErrorMessage: "string should not be empty",
		},
		{
			name:                 "Empty string with spaces",
			content:              "        ",
			expectedError:        true,
			expectedErrorMessage: "string should not be empty",
		},
		{
			name:          "Not empty string",
			content:       "not_empty",
			expectedError: false,
		},
	}

	testFunc := func(t *testing.T, tc testCase) {
		t.Helper()

		fn := ValidateEmptyString()

		err := fn(tc.content)

		require.Equal(t, tc.expectedError, err != nil)

		if tc.expectedError {
			require.ErrorContains(t, err, tc.expectedErrorMessage)
		}
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) { testFunc(t, tc) })
	}
}

func TestMap(t *testing.T) {
	type testCase struct {
		name     string
		input    []int
		expected []string
	}

	testCases := []testCase{
		{
			name:     "Convert integers to strings",
			input:    []int{1, 2, 3},
			expected: []string{"1", "2", "3"},
		},
		{
			name:     "Empty slice",
			input:    []int{},
			expected: []string{},
		},
	}

	testFunc := func(t *testing.T, tc testCase) {
		t.Helper()

		result := Map(tc.input, strconv.Itoa)
		require.Equal(t, tc.expected, result)
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) { testFunc(t, tc) })
	}
}

func TestGetPercentage(t *testing.T) {
	type testCase struct {
		name         string
		total        uint64
		currentValue uint64
		expected     uint64
	}

	testCases := []testCase{
		{
			name:         "Normal case",
			total:        200,
			currentValue: 50,
			expected:     25,
		},
		{
			name:         "Division by zero",
			total:        0,
			currentValue: 50,
			expected:     0,
		},
	}

	testFunc := func(t *testing.T, tc testCase) {
		t.Helper()

		result := GetPercentage(tc.total, tc.currentValue)
		require.Equal(t, tc.expected, result)
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) { testFunc(t, tc) })
	}
}
