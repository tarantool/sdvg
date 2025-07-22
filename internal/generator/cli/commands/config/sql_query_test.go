//nolint:dupl
package config

import (
	"os"
	"testing"

	"github.com/pkg/errors"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	openaiMock "github.com/tarantool/sdvg/internal/generator/cli/openai/mock"
	"github.com/tarantool/sdvg/internal/generator/cli/options"
	rendererMock "github.com/tarantool/sdvg/internal/generator/cli/render/mock"
	"github.com/tarantool/sdvg/internal/generator/cli/streams"
)

func TestNewSQLQueryCommand(t *testing.T) {
	var (
		extraTempFile  *os.File
		resultTempFile *os.File
		err            error
	)

	type testCase struct {
		name          string
		content       string
		expectedError bool
		mockFunc      func(r *rendererMock.Renderer, openAI *openaiMock.OpenAIService)
	}

	testCases := []testCase{
		{
			name:          "Successful",
			content:       "sql",
			expectedError: false,
			mockFunc: func(r *rendererMock.Renderer, openAI *openaiMock.OpenAIService) {
				r.
					On("InputMenu", mock.Anything, "Enter path to save generation config", mock.Anything).
					Return(resultTempFile.Name(), nil)
				r.
					On("InputMenu", mock.Anything, "Enter path to file containing SQL query", mock.Anything).
					Return(extraTempFile.Name(), nil)
				r.
					On("WithSpinner", mock.Anything, mock.Anything).
					Return()

				openAI.
					On("Ping", mock.Anything).
					Return(nil)
				openAI.
					On("GenerateConfig", mock.Anything, mock.Anything, mock.Anything).
					Return(`{"models":{"test":{"rows_count":1}}}`, nil)
			},
		},
		{
			name:          "Failure",
			expectedError: true,
			mockFunc: func(r *rendererMock.Renderer, _ *openaiMock.OpenAIService) {
				r.
					On("InputMenu", mock.Anything, mock.Anything, mock.Anything).
					Return("", errors.New(""))
			},
		},
	}

	testFunc := func(t *testing.T, tc testCase) {
		t.Helper()

		resultTempFile, err = os.CreateTemp(t.TempDir(), "result-*.json")
		require.NoError(t, err)
		defer os.Remove(resultTempFile.Name())

		extraTempFile, err = os.CreateTemp(t.TempDir(), "extra-*.txt")
		require.NoError(t, err)
		defer os.Remove(extraTempFile.Name())

		_, err = extraTempFile.WriteString(tc.content)
		if err != nil {
			t.Fatalf("failed to save content: %s", err)
		}

		err = extraTempFile.Close()
		if err != nil {
			t.Fatalf("failed to close file: %s", err)
		}

		r := rendererMock.NewRenderer(t)
		openAI := openaiMock.NewOpenAIService(t)
		tc.mockFunc(r, openAI)

		cliOpts := &options.CliOptions{}
		cliOpts.SetRenderer(r)
		cliOpts.SetOpenAI(openAI)
		cliOpts.SetOut(streams.NewOut(os.Stdout))

		opts := &generateConfigOptions{}

		cmd := NewSQLQueryCommand(cliOpts, opts)
		cmd.SetArgs([]string{})

		err = cmd.Execute()

		require.Equal(t, tc.expectedError, err != nil)
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) { testFunc(t, tc) })
	}
}
