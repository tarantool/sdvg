package general

import (
	"context"
	"testing"

	"github.com/pkg/errors"
	"github.com/sashabaranov/go-openai"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	openaiMock "sdvg/internal/openai/mock"
)

func TestPing(t *testing.T) {
	type testCase struct {
		name     string
		wantErr  bool
		mockFunc func(api *openaiMock.API)
	}

	testCases := []testCase{
		{
			name:    "Success",
			wantErr: false,
			mockFunc: func(api *openaiMock.API) {
				api.
					On("Models", mock.Anything).
					Return(nil, nil)
			},
		},
		{
			name:    "Failure",
			wantErr: true,
			mockFunc: func(api *openaiMock.API) {
				api.
					On("Models", mock.Anything).
					Return(nil, errors.New("test error"))
			},
		},
	}

	testFunc := func(t *testing.T, tc testCase) {
		t.Helper()

		api := openaiMock.NewAPI(t)
		tc.mockFunc(api)

		service := NewOpenAIService(api)

		err := service.Ping(context.Background())

		require.Equal(t, tc.wantErr, err != nil)
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) { testFunc(t, tc) })
	}
}

func TestGenerateConfig(t *testing.T) {
	type testCase struct {
		name     string
		format   string
		expected string
		wantErr  bool
		mockFunc func(api *openaiMock.API)
	}

	testCases := []testCase{
		{
			name:    "Failure",
			wantErr: true,
			mockFunc: func(api *openaiMock.API) {
				api.
					On("SendRequest", mock.Anything, mock.Anything).
					Return(openai.ChatCompletionResponse{}, errors.New("test error"))
			},
		},
		{
			name:     "Success YAML format",
			format:   "yaml",
			wantErr:  false,
			expected: "data",
			mockFunc: func(api *openaiMock.API) {
				api.
					On("SendRequest", mock.Anything, mock.Anything).
					Return(openai.ChatCompletionResponse{
						Choices: []openai.ChatCompletionChoice{
							{
								Message: openai.ChatCompletionMessage{
									Content: `
Some text 
` + "```yaml" + `
data
` + "```"},
							},
						},
					},
						nil,
					)
			},
		},
		{
			name:     "Success JSON format",
			format:   "json",
			wantErr:  false,
			expected: "data",
			mockFunc: func(api *openaiMock.API) {
				api.
					On("SendRequest", mock.Anything, mock.Anything).
					Return(openai.ChatCompletionResponse{
						Choices: []openai.ChatCompletionChoice{
							{
								Message: openai.ChatCompletionMessage{
									Content: `
Some text 
` + "```json" + `
data
` + "```"},
							},
						},
					},
						nil,
					)
			},
		},
	}

	testFunc := func(t *testing.T, tc testCase) {
		t.Helper()

		api := openaiMock.NewAPI(t)
		api.
			On("BuildRequest", mock.Anything, mock.Anything, mock.Anything).
			Return(openai.ChatCompletionRequest{})

		tc.mockFunc(api)

		service := NewOpenAIService(api)

		actual, err := service.GenerateConfig(context.Background(), tc.format, "")

		require.Equal(t, tc.wantErr, err != nil)
		require.Equal(t, tc.expected, actual)
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) { testFunc(t, tc) })
	}
}
