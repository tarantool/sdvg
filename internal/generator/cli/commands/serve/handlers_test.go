package serve

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/labstack/echo/v4"
	"github.com/pkg/errors"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	openaiMock "github.com/tarantool/sdvg/internal/generator/cli/openai/mock"
	"github.com/tarantool/sdvg/internal/generator/usecase"
	usecaseMock "github.com/tarantool/sdvg/internal/generator/usecase/mock"
)

func TestHandleGenerate(t *testing.T) {
	type testCase struct {
		name            string
		expectedCode    int
		expectedMessage string
		mockFunc        func(*usecaseMock.UseCase)
		reqBody         []byte
	}

	testCases := []testCase{
		{
			name:            "Successful task creation",
			expectedCode:    http.StatusOK,
			expectedMessage: "testID",
			mockFunc: func(uc *usecaseMock.UseCase) {
				uc.
					On("CreateTask", mock.Anything, mock.Anything).
					Return("testID", nil)
			},
			reqBody: []byte(`{"models":{"test":{"rows_count":1}}}`),
		},
		{
			name:            "Invalid generation config",
			expectedCode:    http.StatusBadRequest,
			expectedMessage: "Generation config is not valid",
			mockFunc:        func(_ *usecaseMock.UseCase) {},
			reqBody:         []byte("{}"),
		},
		{
			name:            "Failed to start generation",
			expectedCode:    http.StatusInternalServerError,
			expectedMessage: "Failed to start generation",
			mockFunc: func(uc *usecaseMock.UseCase) {
				uc.
					On("CreateTask", mock.Anything, mock.Anything).
					Return("", errors.New(""))
			},
			reqBody: []byte(`{"models":{"test":{"rows_count":1}}}`),
		},
	}

	testFunc := func(t *testing.T, tc testCase) {
		t.Helper()

		uc := usecaseMock.NewUseCase(t)
		tc.mockFunc(uc)

		req := httptest.NewRequest(http.MethodPost, "/generate", bytes.NewReader(tc.reqBody))
		req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)

		res := httptest.NewRecorder()

		e := echo.New()
		context := e.NewContext(req, res)

		opts := handlerOptions{
			useCase: uc,
		}

		err := handleGenerate(opts, context)
		require.NoError(t, err)
		require.Equal(t, tc.expectedCode, res.Code)

		var resp response

		err = json.Unmarshal(res.Body.Bytes(), &resp)
		if err == nil {
			require.Equal(t, tc.expectedMessage, resp.Message)
		} else {
			require.Equal(t, tc.expectedMessage, res.Body.String())
		}
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) { testFunc(t, tc) })
	}
}

func TestHandleValidate(t *testing.T) {
	type testCase struct {
		name            string
		expectedCode    int
		expectedMessage string
		reqBody         []byte
	}

	testCases := []testCase{
		{
			name:            "Successful validation",
			expectedCode:    http.StatusOK,
			expectedMessage: "Generation config is valid",
			reqBody:         []byte(`{"models":{"test":{"rows_count":1}}}`),
		},
		{
			name:            "Failure validation",
			expectedCode:    http.StatusBadRequest,
			expectedMessage: "Generation config is not valid",
			reqBody:         []byte("{}"),
		},
	}

	testFunc := func(t *testing.T, tc testCase) {
		t.Helper()

		req := httptest.NewRequest(http.MethodPost, "/validate-config", bytes.NewReader(tc.reqBody))
		req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)

		res := httptest.NewRecorder()

		e := echo.New()
		context := e.NewContext(req, res)

		opts := handlerOptions{}

		err := handleValidate(opts, context)
		require.NoError(t, err)
		require.Equal(t, tc.expectedCode, res.Code)

		var resp response
		err = json.Unmarshal(res.Body.Bytes(), &resp)

		require.NoError(t, err)
		require.Equal(t, tc.expectedMessage, resp.Message)
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) { testFunc(t, tc) })
	}
}

func TestHandleStatus(t *testing.T) {
	type testCase struct {
		name            string
		expectedCode    int
		expectedMessage string
		mockFunc        func(*usecaseMock.UseCase)
	}

	testCases := []testCase{
		{
			name:            "Successful generation complete",
			expectedCode:    http.StatusOK,
			expectedMessage: "Generation completed successfully",
			mockFunc: func(uc *usecaseMock.UseCase) {
				uc.
					On("GetResult", "testID").
					Return(true, nil)
			},
		},
		{
			name:            "Successful getting generation progress",
			expectedCode:    http.StatusOK,
			expectedMessage: "{\"key1\":50,\"key2\":25}\n",
			mockFunc: func(uc *usecaseMock.UseCase) {
				uc.
					On("GetResult", "testID").
					Return(false, nil)
				uc.
					On("GetProgress", "testID").
					Return(map[string]usecase.Progress{
						"key1": {
							Total: 100,
							Done:  50,
						},
						"key2": {
							Total: 100,
							Done:  25,
						},
					}, nil)
			},
		},
		{
			name:            "Failed getting generation result",
			expectedCode:    http.StatusInternalServerError,
			expectedMessage: "Failed to retrieve generation result",
			mockFunc: func(uc *usecaseMock.UseCase) {
				uc.
					On("GetResult", "testID").
					Return(false, errors.New("test error"))
			},
		},
		{
			name:            "Failed getting generation progress",
			expectedCode:    http.StatusInternalServerError,
			expectedMessage: "Failed to retrieve generation progress",
			mockFunc: func(uc *usecaseMock.UseCase) {
				uc.
					On("GetResult", "testID").
					Return(false, nil)
				uc.
					On("GetProgress", "testID").
					Return(nil, errors.New("test error"))
			},
		},
	}

	testFunc := func(t *testing.T, tc testCase) {
		t.Helper()

		uc := usecaseMock.NewUseCase(t)
		tc.mockFunc(uc)

		req := httptest.NewRequest(http.MethodGet, "/status/testID", nil)
		req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)

		res := httptest.NewRecorder()

		e := echo.New()
		context := e.NewContext(req, res)
		context.SetParamNames("taskID")
		context.SetParamValues("testID")

		opts := handlerOptions{
			useCase: uc,
		}

		err := handleStatus(opts, context)
		require.NoError(t, err)
		require.Equal(t, tc.expectedCode, res.Code)

		var resp response

		err = json.Unmarshal(res.Body.Bytes(), &resp)
		require.NoError(t, err)

		if resp.Message != "" || resp.Error != "" {
			require.Equal(t, tc.expectedMessage, resp.Message)
		} else {
			require.Equal(t, tc.expectedMessage, res.Body.String())
		}
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) { testFunc(t, tc) })
	}
}

func TestHandleGenerateConfig(t *testing.T) {
	type testCase struct {
		name            string
		expectedCode    int
		expectedMessage string
		mockFunc        func(client *openaiMock.OpenAIService)
		reqBody         []byte
	}

	testCases := []testCase{
		{
			name:            "Successful generation",
			expectedCode:    http.StatusOK,
			expectedMessage: `{"models":{"test":{"rows_count":1}}}`,
			mockFunc: func(client *openaiMock.OpenAIService) {
				client.
					On("Ping", mock.Anything).
					Return(nil)

				client.
					On("GenerateConfig", mock.Anything, mock.Anything, mock.Anything).
					Return(`{"models":{"test":{"rows_count":1}}}`, nil)
			},
			reqBody: []byte(`
{
  "format": "yaml",
  "message": "test"
} 
`),
		},
		{
			name:            "Failure generation",
			expectedCode:    http.StatusInternalServerError,
			expectedMessage: "Unable to generate config",
			mockFunc: func(client *openaiMock.OpenAIService) {
				client.
					On("Ping", mock.Anything).
					Return(nil)

				client.
					On("GenerateConfig", mock.Anything, mock.Anything, mock.Anything).
					Return("", errors.New("test error"))
			},
			reqBody: []byte(`
{
  "format": "yaml",
  "message": "test"
}
`),
		},
		{
			name:            "Openai API is not available",
			expectedCode:    http.StatusServiceUnavailable,
			expectedMessage: "OpenAI is not available",
			mockFunc: func(client *openaiMock.OpenAIService) {
				client.
					On("Ping", mock.Anything).
					Return(errors.New("test error"))
			},
			reqBody: []byte(`
{
  "format": "yaml",
  "message": "test"
}
`),
		},
		{
			name:            "Invalid request body",
			expectedCode:    http.StatusBadRequest,
			expectedMessage: "Invalid request body",
			mockFunc:        func(_ *openaiMock.OpenAIService) {},
			reqBody: []byte(`
{
  format: unsupported
  message: test
}
`),
		},
		{
			name:            "Unsupported format",
			expectedCode:    http.StatusBadRequest,
			expectedMessage: "Unsupported format",
			mockFunc:        func(_ *openaiMock.OpenAIService) {},
			reqBody: []byte(`
{
  "format": "unsupported",
  "message": "test"
}
`),
		},
	}

	testFunc := func(t *testing.T, tc testCase) {
		t.Helper()

		openAI := openaiMock.NewOpenAIService(t)
		tc.mockFunc(openAI)

		req := httptest.NewRequest(http.MethodGet, "/generate-config", bytes.NewReader(tc.reqBody))
		req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)

		res := httptest.NewRecorder()

		e := echo.New()
		context := e.NewContext(req, res)

		opts := handlerOptions{
			openAI: openAI,
		}

		err := handleGenerateConfig(opts, context)
		require.NoError(t, err)
		require.Equal(t, tc.expectedCode, res.Code)

		var resp response

		err = json.Unmarshal(res.Body.Bytes(), &resp)
		require.NoError(t, err)

		if resp.Message != "" || resp.Error != "" {
			require.Equal(t, tc.expectedMessage, resp.Message)
		} else {
			require.Equal(t, tc.expectedMessage, res.Body.String())
		}
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) { testFunc(t, tc) })
	}
}
