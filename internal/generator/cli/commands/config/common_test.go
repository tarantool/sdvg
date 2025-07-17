package config

import (
	"context"
	"os"
	"testing"

	"github.com/pkg/errors"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	openaiMock "sdvg/internal/generator/cli/openai/mock"
	rendererMock "sdvg/internal/generator/cli/render/mock"
)

func TestGetPathToSaveGenerationConfig(t *testing.T) {
	type testCase struct {
		name          string
		path          string
		expectedPath  string
		expectedError bool
		mockFunc      func(r *rendererMock.Renderer)
	}

	testCases := []testCase{
		{
			name:          "Config path from options",
			path:          "file.txt",
			expectedPath:  "file.txt",
			expectedError: false,
			mockFunc:      func(_ *rendererMock.Renderer) {},
		},
		{
			name:          "Config path from input",
			path:          "",
			expectedPath:  "file.txt",
			expectedError: false,
			mockFunc: func(r *rendererMock.Renderer) {
				r.
					On("InputMenu", mock.Anything, mock.Anything, mock.Anything).
					Return("file.txt", nil)
			},
		},
		{
			name:          "Error input menu",
			path:          "",
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

		opts := &generateConfigOptions{
			generationConfigSavePath: tc.path,
			renderer:                 r,
		}

		err := getPathToSaveGenerationConfig(context.Background(), opts)

		require.Equal(t, tc.expectedError, err != nil)
		require.Equal(t, tc.expectedPath, opts.generationConfigSavePath)
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) { testFunc(t, tc) })
	}
}

func TestGetPathToExtraFile(t *testing.T) {
	type testCase struct {
		name          string
		path          string
		expectedPath  string
		expectedError bool
		mockFunc      func(r *rendererMock.Renderer)
	}

	testCases := []testCase{
		{
			name:          "Config path from options",
			path:          "file.txt",
			expectedPath:  "file.txt",
			expectedError: false,
			mockFunc:      func(_ *rendererMock.Renderer) {},
		},
		{
			name:          "Config path from input",
			path:          "",
			expectedPath:  "file.txt",
			expectedError: false,
			mockFunc: func(r *rendererMock.Renderer) {
				r.
					On("InputMenu", mock.Anything, mock.Anything, mock.Anything).
					Return("file.txt", nil)
			},
		},
		{
			name:          "Error input menu",
			path:          "",
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

		opts := &generateConfigOptions{
			extraFilePath: tc.path,
			renderer:      r,
		}

		err := getPathToExtraFile(context.Background(), opts, "")

		require.Equal(t, tc.expectedError, err != nil)
		require.Equal(t, tc.expectedPath, opts.extraFilePath)
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) { testFunc(t, tc) })
	}
}

func TestGetDescriptionRequest(t *testing.T) {
	type testCase struct {
		name          string
		expectedValue string
		expectedError bool
		mockFunc      func(r *rendererMock.Renderer)
	}

	testCases := []testCase{
		{
			name:          "Successful",
			expectedValue: "Словесное описание\ndescription",
			expectedError: false,
			mockFunc: func(r *rendererMock.Renderer) {
				r.
					On("TextMenu", mock.Anything, mock.Anything).
					Return("description", nil)
			},
		},
		{
			name:          "Failure",
			expectedValue: "",
			expectedError: true,
			mockFunc: func(r *rendererMock.Renderer) {
				r.
					On("TextMenu", mock.Anything, mock.Anything).
					Return("", errors.New(""))
			},
		},
	}

	testFunc := func(t *testing.T, tc testCase) {
		t.Helper()

		r := rendererMock.NewRenderer(t)

		tc.mockFunc(r)

		opts := &generateConfigOptions{
			renderer: r,
		}

		actual, err := getDescriptionRequest(context.Background(), opts)

		require.Equal(t, tc.expectedError, err != nil)
		require.Equal(t, tc.expectedValue, actual)
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) { testFunc(t, tc) })
	}
}

func TestGetSQLOrSampleRequest(t *testing.T) {
	type testCase struct {
		name          string
		mode          generationMode
		extraInput    bool
		emptyPath     bool
		content       string
		expectedValue string
		expectedError bool
		mockFunc      func(r *rendererMock.Renderer)
	}

	testCases := []testCase{
		{
			name:          "Successful SQL request",
			mode:          sqlQuery,
			extraInput:    false,
			emptyPath:     false,
			content:       "sql",
			expectedValue: "SQL запрос\nsql",
			expectedError: false,
			mockFunc:      func(_ *rendererMock.Renderer) {},
		},
		{
			name:          "Successful data samples request",
			mode:          dataSample,
			extraInput:    false,
			emptyPath:     false,
			content:       "data",
			expectedValue: "Пример данных\ndata",
			expectedError: false,
			mockFunc:      func(_ *rendererMock.Renderer) {},
		},
		{
			name:          "With extra input",
			mode:          sqlQuery,
			extraInput:    true,
			emptyPath:     false,
			content:       "sql",
			expectedValue: "SQL запрос\nsql\nУточняющая информация\ninfo",
			expectedError: false,
			mockFunc: func(r *rendererMock.Renderer) {
				r.
					On("TextMenu", mock.Anything, mock.Anything).
					Return("info", nil)
			},
		},
		{
			name:          "Failed to get extra file path",
			mode:          sqlQuery,
			extraInput:    false,
			emptyPath:     true,
			content:       "",
			expectedValue: "",
			expectedError: true,
			mockFunc: func(r *rendererMock.Renderer) {
				r.
					On("InputMenu", mock.Anything, mock.Anything, mock.Anything).
					Return("", errors.New(""))
			},
		},
		{
			name:          "Failed to read extra file",
			mode:          sqlQuery,
			extraInput:    false,
			emptyPath:     true,
			content:       "",
			expectedError: true,
			mockFunc: func(r *rendererMock.Renderer) {
				r.
					On("InputMenu", mock.Anything, mock.Anything, mock.Anything).
					Return("", nil)
			},
		},
		{
			name:          "Failed to read extra input",
			mode:          sqlQuery,
			extraInput:    true,
			emptyPath:     false,
			content:       "sql",
			expectedError: true,
			mockFunc: func(r *rendererMock.Renderer) {
				r.
					On("TextMenu", mock.Anything, mock.Anything).
					Return("", errors.New(""))
			},
		},
	}

	testFunc := func(t *testing.T, tc testCase) {
		t.Helper()

		tempFile, err := os.CreateTemp(t.TempDir(), "extra-*.txt")
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

		path := tempFile.Name()

		if tc.emptyPath {
			path = ""
		}

		opts := &generateConfigOptions{
			renderer:      r,
			extraFilePath: path,
			extraInput:    tc.extraInput,
		}

		actual, err := getSQLOrSampleRequest(context.Background(), opts, tc.mode)

		require.Equal(t, tc.expectedError, err != nil)
		require.Equal(t, tc.expectedValue, actual)
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) { testFunc(t, tc) })
	}
}

func TestCheckAssessToOpenAI(t *testing.T) {
	type testCase struct {
		name          string
		expectedError bool
		mockFunc      func(openAI *openaiMock.OpenAIService)
	}

	testCases := []testCase{
		{
			name:          "Available",
			expectedError: false,
			mockFunc: func(openAI *openaiMock.OpenAIService) {
				openAI.
					On("Ping", mock.Anything).
					Return(nil)
			},
		},
		{
			name:          "Not available",
			expectedError: true,
			mockFunc: func(openAI *openaiMock.OpenAIService) {
				openAI.
					On("Ping", mock.Anything).
					Return(errors.New(""))
			},
		},
	}

	testFunc := func(t *testing.T, tc testCase) {
		t.Helper()

		r := rendererMock.NewRenderer(t)

		r.
			On("WithSpinner", mock.Anything, mock.Anything, mock.Anything).
			Return()

		openAI := openaiMock.NewOpenAIService(t)

		tc.mockFunc(openAI)

		opts := &generateConfigOptions{
			renderer: r,
			openAI:   openAI,
		}

		err := checkAccessToOpenAI(context.Background(), opts)

		require.Equal(t, tc.expectedError, err != nil)
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) { testFunc(t, tc) })
	}
}

func TestTryGenerate(t *testing.T) {
	type testCase struct {
		name          string
		expectedError bool
		mockFunc      func(openAI *openaiMock.OpenAIService)
	}

	testCases := []testCase{
		{
			name:          "Successful",
			expectedError: false,
			mockFunc: func(openAI *openaiMock.OpenAIService) {
				openAI.
					On("GenerateConfig", mock.Anything, mock.Anything, mock.Anything).
					Return(`{"models":{"test":{"rows_count":1}}}`, nil)
			},
		},
		{
			name:          "Failure generate",
			expectedError: true,
			mockFunc: func(openAI *openaiMock.OpenAIService) {
				openAI.
					On("GenerateConfig", mock.Anything, mock.Anything, mock.Anything).
					Return("", errors.New(""))
			},
		},
		{
			name:          "Failure regenerate",
			expectedError: true,
			mockFunc: func(openAI *openaiMock.OpenAIService) {
				openAI.
					On("GenerateConfig", mock.Anything, mock.Anything, mock.Anything).
					Return("{}", nil)
				openAI.
					On("RegenerateConfig", mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything).
					Return("", errors.New(""))
			},
		},
	}

	testFunc := func(t *testing.T, tc testCase) {
		t.Helper()

		r := rendererMock.NewRenderer(t)

		r.
			On("WithSpinner", mock.Anything, mock.Anything, mock.Anything).
			Return()

		openAI := openaiMock.NewOpenAIService(t)

		tc.mockFunc(openAI)

		opts := &generateConfigOptions{
			renderer: r,
			openAI:   openAI,
		}

		_, err := tryGenerate(context.Background(), opts, "", "json")

		require.Equal(t, tc.expectedError, err != nil)
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) { testFunc(t, tc) })
	}
}

func TestGenerate(t *testing.T) {
	type testCase struct {
		name                 string
		content              string
		mode                 generationMode
		emptyPathToSave      bool
		emptyPathToExtraFile bool
		expectedError        bool
		mockFunc             func(r *rendererMock.Renderer, openAI *openaiMock.OpenAIService)
	}

	testCases := []testCase{
		{
			name:          "Successful",
			mode:          description,
			expectedError: false,
			mockFunc: func(r *rendererMock.Renderer, openAI *openaiMock.OpenAIService) {
				r.
					On("WithSpinner", mock.Anything, mock.Anything).
					Return()
				r.
					On("TextMenu", mock.Anything, mock.Anything).
					Return("description", nil)

				openAI.
					On("Ping", mock.Anything).
					Return(nil)
				openAI.
					On("GenerateConfig", mock.Anything, mock.Anything, mock.Anything).
					Return(`{"models":{"test":{"rows_count":1}}}`, nil)
			},
		},
		{
			name:            "Failed to get path to save",
			mode:            description,
			emptyPathToSave: true,
			expectedError:   true,
			mockFunc: func(r *rendererMock.Renderer, _ *openaiMock.OpenAIService) {
				r.
					On("InputMenu", mock.Anything, mock.Anything, mock.Anything).
					Return("", errors.New(""))
			},
		},
		{
			name:                 "Failed to get path to extra file",
			mode:                 sqlQuery,
			emptyPathToExtraFile: true,
			expectedError:        true,
			mockFunc: func(r *rendererMock.Renderer, _ *openaiMock.OpenAIService) {
				r.
					On("InputMenu", mock.Anything, mock.Anything, mock.Anything).
					Return("", errors.New(""))
			},
		},
		{
			name:          "Failed to get description request",
			mode:          description,
			expectedError: true,
			mockFunc: func(r *rendererMock.Renderer, _ *openaiMock.OpenAIService) {
				r.
					On("TextMenu", mock.Anything, mock.Anything).
					Return("", errors.New(""))
			},
		},
		{
			name:          "Failed to get sql request",
			mode:          sqlQuery,
			expectedError: true,
			mockFunc:      func(_ *rendererMock.Renderer, _ *openaiMock.OpenAIService) {},
		},
		{
			name:          "Failed to check access to Open AI",
			mode:          description,
			expectedError: true,
			mockFunc: func(r *rendererMock.Renderer, openAI *openaiMock.OpenAIService) {
				r.
					On("TextMenu", mock.Anything, mock.Anything).
					Return("", nil)
				r.
					On("WithSpinner", mock.Anything, mock.Anything).
					Return()

				openAI.
					On("Ping", mock.Anything).
					Return(errors.New(""))
			},
		},
		{
			name:          "Failed to generate",
			mode:          description,
			expectedError: true,
			mockFunc: func(r *rendererMock.Renderer, openAI *openaiMock.OpenAIService) {
				r.
					On("TextMenu", mock.Anything, mock.Anything).
					Return("", nil)
				r.
					On("WithSpinner", mock.Anything, mock.Anything).
					Return()

				openAI.
					On("Ping", mock.Anything).
					Return(nil)
				openAI.
					On("GenerateConfig", mock.Anything, mock.Anything, mock.Anything).
					Return("", errors.New(""))
			},
		},
	}

	testFunc := func(t *testing.T, tc testCase) {
		t.Helper()

		resultTempFile, err := os.CreateTemp(t.TempDir(), "result-*.json")
		require.NoError(t, err)
		defer os.Remove(resultTempFile.Name())

		extraTempFile, err := os.CreateTemp(t.TempDir(), "extra-file-*.txt")
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

		pathToSave := resultTempFile.Name()
		if tc.emptyPathToSave {
			pathToSave = ""
		}

		pathToExtraFile := extraTempFile.Name()
		if tc.emptyPathToExtraFile {
			pathToExtraFile = ""
		}

		opts := &generateConfigOptions{
			renderer:                 r,
			openAI:                   openAI,
			generationConfigSavePath: pathToSave,
			extraFilePath:            pathToExtraFile,
		}

		err = generate(context.Background(), opts, tc.mode)

		require.Equal(t, tc.expectedError, err != nil)
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) { testFunc(t, tc) })
	}
}

func TestReadFile(t *testing.T) {
	type testCase struct {
		name            string
		content         string
		bufferSize      int
		emptyPath       bool
		expectedError   bool
		expectedContent string
	}

	testCases := []testCase{
		{
			name:            "Successful",
			content:         "TestMessage",
			bufferSize:      len("TestMessage"),
			emptyPath:       false,
			expectedError:   false,
			expectedContent: "TestMessage",
		},
		{
			name:            "Successful with buffer size",
			content:         "TestMessage",
			bufferSize:      len("Test"),
			emptyPath:       false,
			expectedError:   false,
			expectedContent: "Test",
		},
		{
			name:            "Failed to open file",
			content:         "",
			bufferSize:      0,
			emptyPath:       true,
			expectedError:   true,
			expectedContent: "",
		},
		{
			name:            "Failed to read file",
			content:         "",
			bufferSize:      1,
			emptyPath:       false,
			expectedError:   true,
			expectedContent: "",
		},
	}

	testFunc := func(t *testing.T, tc testCase) {
		t.Helper()

		tempFile, err := os.CreateTemp(t.TempDir(), "sdvg-config-*.txt")
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

		path := tempFile.Name()
		if tc.emptyPath {
			path = ""
		}

		actual, err := readFile(path, tc.bufferSize)

		require.Equal(t, tc.expectedError, err != nil)
		require.Equal(t, tc.expectedContent, actual)
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) { testFunc(t, tc) })
	}
}
