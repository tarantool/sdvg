package validate

import (
	"bytes"
	"context"
	"os"
	"strings"
	"testing"

	"github.com/pkg/errors"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"sdvg/internal/generator/cli/options"
	rendererMock "sdvg/internal/generator/cli/render/mock"
	"sdvg/internal/generator/cli/streams"
)

func TestGetGenerationConfigPath(t *testing.T) {
	type testCase struct {
		name          string
		args          []string
		expectedPath  string
		expectedError bool
		mockFunc      func(r *rendererMock.Renderer)
	}

	testCases := []testCase{
		{
			name:          "Config path from args",
			args:          []string{"path.txt"},
			expectedPath:  "path.txt",
			expectedError: false,
			mockFunc:      func(_ *rendererMock.Renderer) {},
		},
		{
			name:          "Config path from input",
			args:          []string{},
			expectedPath:  "path.txt",
			expectedError: false,
			mockFunc: func(r *rendererMock.Renderer) {
				r.
					On("InputMenu", mock.Anything, mock.Anything, mock.Anything).
					Return("path.txt", nil)
			},
		},
		{
			name:          "Error input menu",
			args:          []string{},
			expectedPath:  "",
			expectedError: true,
			mockFunc: func(r *rendererMock.Renderer) {
				r.
					On("InputMenu", mock.Anything, mock.Anything, mock.Anything).
					Return("", errors.New(""))
			},
		},
	}

	testFunc := func(t *testing.T, tc testCase) {
		t.Helper()

		r := rendererMock.NewRenderer(t)
		tc.mockFunc(r)

		opts := &validateOptions{
			renderer: r,
		}

		err := getGenerationConfigFilePath(context.Background(), opts, tc.args)

		require.Equal(t, tc.expectedError, err != nil)
		require.Equal(t, tc.expectedPath, opts.generationConfigPath)
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) { testFunc(t, tc) })
	}
}

func TestRunValidate(t *testing.T) {
	type testCase struct {
		name            string
		content         string
		expectedError   bool
		expectedMessage string
	}

	testCases := []testCase{
		{
			name:            "Successful validation",
			content:         `{"models":{"test":{"rows_count":1}}}`,
			expectedError:   false,
			expectedMessage: "Generation config is valid",
		},
		{
			name:            "Failure validation",
			content:         "{}",
			expectedError:   true,
			expectedMessage: "",
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

		opts := &validateOptions{
			generationConfigPath: tempFile.Name(),
		}

		out := new(bytes.Buffer)

		err = runValidate(out, opts)

		require.Equal(t, tc.expectedError, err != nil)
		require.Equal(t, strings.TrimSpace(tc.expectedMessage), strings.TrimSpace(out.String()))
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) { testFunc(t, tc) })
	}
}

func TestNewValidateConfigCommand(t *testing.T) {
	var (
		tempFile *os.File
		err      error
	)

	type testCase struct {
		name          string
		content       string
		expectedError bool
		mockFunc      func(r *rendererMock.Renderer)
	}

	testCases := []testCase{
		{
			name:          "Successful validation",
			content:       `{"models":{"test":{"rows_count":1}}}`,
			expectedError: false,
			mockFunc: func(r *rendererMock.Renderer) {
				r.On("InputMenu", mock.Anything, mock.Anything, mock.Anything).
					Return(tempFile.Name(), nil)
			},
		},
		{
			name:          "Failed to get path",
			content:       "",
			expectedError: true,
			mockFunc: func(r *rendererMock.Renderer) {
				r.On("InputMenu", mock.Anything, mock.Anything, mock.Anything).
					Return("", errors.New(""))
			},
		},
		{
			name:          "Failure validation",
			content:       "{}",
			expectedError: true,
			mockFunc: func(r *rendererMock.Renderer) {
				r.On("InputMenu", mock.Anything, mock.Anything, mock.Anything).
					Return(tempFile.Name(), nil)
			},
		},
	}

	testFunc := func(t *testing.T, tc testCase) {
		t.Helper()

		tempFile, err = os.CreateTemp(t.TempDir(), "sdvg-config-*.yml")
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

		r := rendererMock.NewRenderer(t)
		tc.mockFunc(r)

		cliOpts := &options.CliOptions{}
		cliOpts.SetRenderer(r)
		cliOpts.SetOut(streams.NewOut(os.Stdout))

		cmd := NewValidateConfigCommand(cliOpts)
		cmd.SetArgs([]string{})

		err = cmd.Execute()

		require.Equal(t, tc.expectedError, err != nil)
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) { testFunc(t, tc) })
	}
}
