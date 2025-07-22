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

func TestNewDescriptionCommand(t *testing.T) {
	var (
		tempFile *os.File
		err      error
	)

	type testCase struct {
		name          string
		expectedError bool
		mockFunc      func(r *rendererMock.Renderer, openAI *openaiMock.OpenAIService)
	}

	testCases := []testCase{
		{
			name:          "Successful",
			expectedError: false,
			mockFunc: func(r *rendererMock.Renderer, openAI *openaiMock.OpenAIService) {
				r.
					On("InputMenu", mock.Anything, mock.Anything, mock.Anything).
					Return(tempFile.Name(), nil)
				r.
					On("TextMenu", mock.Anything, mock.Anything).
					Return("description", nil)
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

		tempFile, err = os.CreateTemp(t.TempDir(), "result-*.json")
		require.NoError(t, err)
		defer os.Remove(tempFile.Name())

		r := rendererMock.NewRenderer(t)
		openAI := openaiMock.NewOpenAIService(t)
		tc.mockFunc(r, openAI)

		cliOpts := &options.CliOptions{}
		cliOpts.SetRenderer(r)
		cliOpts.SetOpenAI(openAI)
		cliOpts.SetOut(streams.NewOut(os.Stdout))

		opts := &generateConfigOptions{}

		cmd := NewDescriptionCommand(cliOpts, opts)
		cmd.SetArgs([]string{})

		err = cmd.Execute()

		require.Equal(t, tc.expectedError, err != nil)
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) { testFunc(t, tc) })
	}
}
