package prompt

import (
	"bytes"
	"context"
	"os"
	"strings"
	"testing"

	"github.com/pkg/errors"
	"github.com/stretchr/testify/require"

	"sdvg/internal/generator/cli/render/assets"
	streams "sdvg/internal/generator/cli/streams"
)

func TestLogo(t *testing.T) {
	expected := assets.LogoText
	out := new(bytes.Buffer)

	renderer := NewRenderer(nil, streams.NewOut(out), false)

	renderer.Logo()

	require.Equal(t, expected, out.String())
}

func TestSelectionMenu(t *testing.T) {
	type testCase struct {
		name            string
		input           string
		expectedError   bool
		items           []string
		expectedItem    string
		expectedMessage string
	}

	testCases := []testCase{
		{
			name:          "Successful",
			input:         "1",
			expectedError: false,
			items:         []string{"item1", "item2"},
			expectedItem:  "item1",
			expectedMessage: `
Test select
1. item1
2. item2
Write a number: 1
Selected: item1
`,
		},
		{
			name:          "Successful retry",
			input:         "3\n1",
			expectedError: false,
			items:         []string{"item1", "item2"},
			expectedItem:  "item1",
			expectedMessage: `
Test select
1. item1
2. item2
Write a number: 3
invalid input, please try again
Write a number: 1
Selected: item1
`,
		},
		{
			name:          "Invalid input",
			input:         "",
			expectedError: true,
			items:         []string{"item1", "item2"},
			expectedItem:  "",
			expectedMessage: `
Test select
1. item1
2. item2
Write a number:
`,
		},
	}

	testFunc := func(t *testing.T, tc testCase) {
		t.Helper()

		in := strings.NewReader(tc.input)
		out := new(bytes.Buffer)

		renderer := NewRenderer(streams.NewIn(in), streams.NewOut(out), false)

		item, err := renderer.SelectionMenu(context.Background(), "Test select", tc.items)

		require.Equal(t, tc.expectedError, err != nil)
		require.Equal(t, tc.expectedItem, item)
		require.Equal(t, strings.TrimSpace(tc.expectedMessage), strings.TrimSpace(out.String()))
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) { testFunc(t, tc) })
	}
}

func TestInputMenu(t *testing.T) {
	type testCase struct {
		name            string
		input           string
		expectedError   bool
		expectedInput   string
		expectedMessage string
	}

	testCases := []testCase{
		{
			name:            "Successful",
			input:           "input",
			expectedError:   false,
			expectedInput:   "input",
			expectedMessage: `Test input: input`,
		},
		{
			name:          "Successful retry",
			input:         "\ninput",
			expectedError: false,
			expectedInput: "input",
			expectedMessage: `
Test input: 
string should not be empty
Test input: input
`,
		},
		{
			name:            "Invalid input",
			input:           "",
			expectedError:   true,
			expectedInput:   "",
			expectedMessage: `Test input:`,
		},
	}

	testFunc := func(t *testing.T, tc testCase) {
		t.Helper()

		in := strings.NewReader(tc.input)
		out := new(bytes.Buffer)

		renderer := NewRenderer(streams.NewIn(in), streams.NewOut(out), false)

		input, err := renderer.InputMenu(context.Background(), "Test input", func(s string) error {
			if strings.TrimSpace(s) == "" {
				return errors.New("string should not be empty")
			}

			return nil
		})

		require.Equal(t, tc.expectedError, err != nil)
		require.Equal(t, tc.expectedInput, input)
		require.Equal(t, strings.TrimSpace(tc.expectedMessage), strings.TrimSpace(out.String()))
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) { testFunc(t, tc) })
	}
}

func TestTextMenu(t *testing.T) {
	type testCase struct {
		name            string
		input           string
		expectedError   bool
		expectedInput   string
		expectedMessage string
	}

	testCases := []testCase{
		{
			name:            "Successful",
			input:           "test\ntext\nmenu",
			expectedError:   false,
			expectedInput:   "test\ntext\nmenu",
			expectedMessage: "Test text menu\ntest\ntext\nmenu",
		},
		{
			name:            "Invalid input",
			input:           "",
			expectedError:   true,
			expectedInput:   "",
			expectedMessage: "Test text menu",
		},
	}

	testFunc := func(t *testing.T, tc testCase) {
		t.Helper()

		in := strings.NewReader(tc.input)
		out := new(bytes.Buffer)

		renderer := NewRenderer(streams.NewIn(in), streams.NewOut(out), false)

		input, err := renderer.TextMenu(context.Background(), "Test text menu")

		require.Equal(t, tc.expectedError, err != nil)
		require.Equal(t, tc.expectedInput, input)
		require.Equal(t, strings.TrimSpace(tc.expectedMessage), strings.TrimSpace(out.String()))
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) { testFunc(t, tc) })
	}
}

func TestEditor(t *testing.T) {
	type testCase struct {
		name          string
		editor        string
		expectedError bool
	}

	testCases := []testCase{
		{
			name:          "wrong editor",
			editor:        "wrong-editor",
			expectedError: true,
		},
		{
			name:          "good editor",
			editor:        "nano",
			expectedError: false,
		},
	}

	testFunc := func(t *testing.T, tc testCase) {
		t.Helper()

		in := strings.NewReader("lol-input")
		out := new(bytes.Buffer)

		t.Setenv(editorEnvVar, tc.editor)

		renderer := NewRenderer(streams.NewIn(in), streams.NewOut(out), false)

		_, err := renderer.getEditor()

		require.Equal(t, tc.expectedError, err != nil)
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) { testFunc(t, tc) })
	}
}

func TestWithSpinner(t *testing.T) {
	t.Helper()

	expected := "Test spinner"
	out := new(bytes.Buffer)

	renderer := NewRenderer(nil, streams.NewOut(out), false)

	renderer.WithSpinner(expected, func() {})

	require.Equal(t, strings.TrimSpace(expected), strings.TrimSpace(out.String()))
}

func TestReadFile(t *testing.T) {
	type testCase struct {
		name            string
		content         string
		expectedError   bool
		emptyPath       bool
		expectedContent string
	}

	testCases := []testCase{
		{
			name:            "Successful",
			content:         "#line1\nline2\n#line3",
			expectedError:   false,
			emptyPath:       false,
			expectedContent: "line2",
		},
		{
			name:            "Failed to read",
			content:         "",
			expectedError:   true,
			emptyPath:       true,
			expectedContent: "",
		},
	}

	testFunc := func(t *testing.T, tc testCase) {
		t.Helper()

		tempFile, err := os.CreateTemp(t.TempDir(), "temp-*.txt")
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

		renderer := NewRenderer(nil, nil, false)

		path := tempFile.Name()
		if tc.emptyPath {
			path = ""
		}

		actual, err := renderer.readFile(path)

		require.Equal(t, tc.expectedError, err != nil)
		require.Equal(t, tc.expectedContent, actual)
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) { testFunc(t, tc) })
	}
}

func TestReadLine(t *testing.T) {
	testCases := []readLinesTestCase{
		{
			name:            "Successful",
			content:         "line1\nline2",
			expectedError:   false,
			closeReader:     false,
			expectedContent: "line1",
		},
		{
			name:            "Failed to read",
			content:         "test",
			expectedError:   true,
			closeReader:     true,
			expectedContent: "",
		},
		{
			name:            "EOF",
			content:         "",
			expectedError:   true,
			closeReader:     false,
			expectedContent: "",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) { readLinesTestFunc(t, tc, SingleLine) })
	}
}

func TestReadMultiline(t *testing.T) {
	testCases := []readLinesTestCase{
		{
			name:            "Successful",
			content:         "line1\nline2",
			expectedError:   false,
			closeReader:     false,
			expectedContent: "line1\nline2",
		},
		{
			name:            "Failed to read",
			content:         "test",
			expectedError:   true,
			closeReader:     true,
			expectedContent: "",
		},
		{
			name:            "EOF",
			content:         "",
			expectedError:   true,
			closeReader:     false,
			expectedContent: "",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) { readLinesTestFunc(t, tc, MultiLine) })
	}
}

const (
	SingleLine = iota
	MultiLine
)

type readLinesTestCase struct {
	name            string
	content         string
	expectedError   bool
	closeReader     bool
	expectedContent string
}

func readLinesTestFunc(t *testing.T, tc readLinesTestCase, mode int) {
	t.Helper()

	tempFile, err := os.CreateTemp(t.TempDir(), "temp-*.txt")
	require.NoError(t, err)
	defer os.Remove(tempFile.Name())

	_, err = tempFile.WriteString(tc.content)
	if err != nil {
		t.Fatalf("failed to save content: %s", err)
	}

	err = tempFile.Sync()
	require.NoError(t, err)

	_, err = tempFile.Seek(0, 0)
	require.NoError(t, err)

	if tc.closeReader {
		err = tempFile.Close()
		if err != nil {
			t.Fatalf("failed to close file: %s", err)
		}
	}

	renderer := NewRenderer(streams.NewIn(tempFile), nil, false)

	var actual string

	switch mode {
	case SingleLine:
		actual, err = renderer.readLine()
	case MultiLine:
		actual, err = renderer.readMultiline()
	}

	require.Equal(t, tc.expectedError, err != nil)
	require.Equal(t, tc.expectedContent, actual)

	if !tc.closeReader {
		err = tempFile.Close()
		if err != nil {
			t.Fatalf("failed to close file: %s", err)
		}
	}
}
