package generate

import (
	"context"
	"os"
	"testing"

	"github.com/pkg/errors"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"github.com/tarantool/sdvg/internal/generator/cli/options"
	rendererMock "github.com/tarantool/sdvg/internal/generator/cli/render/mock"
	"github.com/tarantool/sdvg/internal/generator/cli/streams"
	"github.com/tarantool/sdvg/internal/generator/usecase"
	usecaseMock "github.com/tarantool/sdvg/internal/generator/usecase/mock"
)

func TestGetGenerationConfigFilePath(t *testing.T) {
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

		opts := &generateOptions{
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

func TestRunGenerate(t *testing.T) {
	type testCase struct {
		name          string
		content       string
		expectedError bool
		mockFunc      func(uc *usecaseMock.UseCase)
	}

	testCases := []testCase{
		{
			name:          "Successful generation",
			content:       `{"models":{"test":{"rows_count":1}}}`,
			expectedError: false,
			mockFunc: func(uc *usecaseMock.UseCase) {
				uc.
					On("CreateTask", mock.Anything, mock.Anything).
					Return(mock.Anything, nil)
				uc.
					On("GetProgress", mock.Anything).
					Return(map[string]usecase.Progress{"test": {Done: 1, Total: 1}}, nil)
				uc.
					On("WaitResult", mock.Anything).
					Return(nil)
			},
		},
		{
			name:          "Generation config is not valid",
			content:       "{}",
			expectedError: true,
			mockFunc:      func(_ *usecaseMock.UseCase) {},
		},
		{
			name:          "Failed to create task",
			content:       `{"models":{"test":{"rows_count":1}}}`,
			expectedError: true,
			mockFunc: func(uc *usecaseMock.UseCase) {
				uc.
					On("CreateTask", mock.Anything, mock.Anything).
					Return("", errors.New(""))
			},
		},
		{
			name:          "Failed to wait result",
			content:       `{"models":{"test":{"rows_count":1}}}`,
			expectedError: true,
			mockFunc: func(uc *usecaseMock.UseCase) {
				uc.
					On("CreateTask", mock.Anything, mock.Anything).
					Return(mock.Anything, nil)
				uc.
					On("WaitResult", mock.Anything).
					Return(errors.New(""))
			},
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

		uc := usecaseMock.NewUseCase(t)

		tc.mockFunc(uc)

		uc.
			On("GetProgress", mock.Anything).
			Return(map[string]usecase.Progress{}, nil).
			Maybe()

		opts := &generateOptions{
			generationConfigPath: tempFile.Name(),
			useCase:              uc,
		}

		err = runGenerate(context.Background(), opts)

		require.Equal(t, tc.expectedError, err != nil, err)
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) { testFunc(t, tc) })
	}
}

func TestNewGenerateCommand(t *testing.T) {
	var (
		tempFile *os.File
		err      error
	)

	type testCase struct {
		name          string
		content       string
		expectedError bool
		mockFunc      func(r *rendererMock.Renderer, uc *usecaseMock.UseCase)
	}

	testCases := []testCase{
		{
			name:          "Successful generation",
			content:       `{"models":{"test":{"rows_count":1}}}`,
			expectedError: false,
			mockFunc: func(r *rendererMock.Renderer, uc *usecaseMock.UseCase) {
				r.
					On("InputMenu", mock.Anything, mock.Anything, mock.Anything).
					Return(tempFile.Name(), nil)

				uc.
					On("CreateTask", mock.Anything, mock.Anything).
					Return(mock.Anything, nil)
				uc.
					On("GetProgress", mock.Anything).
					Return(nil, errors.New(""))
				uc.
					On("WaitResult", mock.Anything).
					Return(nil)
			},
		},
		{
			name:          "Failed to get path to generation config",
			expectedError: true,
			mockFunc: func(r *rendererMock.Renderer, _ *usecaseMock.UseCase) {
				r.
					On("InputMenu", mock.Anything, mock.Anything, mock.Anything).
					Return("", errors.New(""))
			},
		},
		{
			name:          "Failure generation",
			content:       "{}",
			expectedError: true,
			mockFunc: func(r *rendererMock.Renderer, _ *usecaseMock.UseCase) {
				r.
					On("InputMenu", mock.Anything, mock.Anything, mock.Anything).
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
		uc := usecaseMock.NewUseCase(t)
		tc.mockFunc(r, uc)

		cliOpts := &options.CliOptions{}
		cliOpts.SetRenderer(r)
		cliOpts.SetUseCase(uc)
		cliOpts.SetOut(streams.NewOut(os.Stdout))

		cmd := NewGenerateCommand(cliOpts)
		cmd.SetArgs([]string{"-f"})

		err = cmd.Execute()

		require.Equal(t, tc.expectedError, err != nil)
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) { testFunc(t, tc) })
	}
}
