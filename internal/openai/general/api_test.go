package general

import (
	"context"
	"testing"

	"github.com/pkg/errors"
	"github.com/sashabaranov/go-openai"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	openaiMocks "github.com/tarantool/sdvg/internal/openai/mock"
)

func TestGetBaseURL(t *testing.T) {
	t.Helper()

	const baseURL = "https://api.openai.com"

	openaiMock := openaiMocks.NewAPI(t)
	openaiMock.On("GetBaseURL").Return(baseURL)

	require.Equal(t, baseURL, openaiMock.GetBaseURL())
}

func TestModels(t *testing.T) {
	type testCase struct {
		name     string
		expected []openai.Model
		wantErr  bool
		mockFunc func(api *openaiMocks.API)
	}

	testCases := []testCase{
		{
			name: "Success",
			expected: []openai.Model{
				{ID: "gpt-4"},
				{ID: "gpt-3.5-turbo"},
			},
			wantErr: false,
			mockFunc: func(api *openaiMocks.API) {
				api.
					On("Models", mock.Anything).
					Return(
						[]openai.Model{
							{ID: "gpt-4"},
							{ID: "gpt-3.5-turbo"},
						}, nil)
			},
		},
		{
			name:     "Failure",
			expected: nil,
			wantErr:  true,
			mockFunc: func(api *openaiMocks.API) {
				api.
					On("Models", mock.Anything).
					Return(nil, errors.New("test error"))
			},
		},
	}

	testFunc := func(t *testing.T, tc testCase) {
		t.Helper()

		openaiMock := openaiMocks.NewAPI(t)
		tc.mockFunc(openaiMock)

		models, err := openaiMock.Models(context.Background())

		require.Equal(t, tc.wantErr, err != nil)
		require.Equal(t, tc.expected, models)
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) { testFunc(t, tc) })
	}
}

func TestSendRequest(t *testing.T) {
	type testCase struct {
		name     string
		expected openai.ChatCompletionResponse
		wantErr  bool
		mockFunc func(api *openaiMocks.API)
	}

	testCases := []testCase{
		{
			name:     "Success",
			expected: openai.ChatCompletionResponse{ID: "response-id"},
			wantErr:  false,
			mockFunc: func(api *openaiMocks.API) {
				api.
					On("SendRequest", mock.Anything, mock.Anything).
					Return(openai.ChatCompletionResponse{ID: "response-id"}, nil)
			},
		},
		{
			name:     "Failure",
			expected: openai.ChatCompletionResponse{},
			wantErr:  true,
			mockFunc: func(api *openaiMocks.API) {
				api.
					On("SendRequest", mock.Anything, mock.Anything).
					Return(openai.ChatCompletionResponse{}, errors.New("test error"))
			},
		},
	}

	testFunc := func(t *testing.T, tc testCase) {
		t.Helper()

		openaiMock := openaiMocks.NewAPI(t)
		tc.mockFunc(openaiMock)

		response, err := openaiMock.SendRequest(context.Background(), openai.ChatCompletionRequest{})

		require.Equal(t, tc.wantErr, err != nil)
		require.Equal(t, tc.expected, response)
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) { testFunc(t, tc) })
	}
}

func TestBuildRequest(t *testing.T) {
	openaiMock := openaiMocks.NewAPI(t)

	systemPrompts := []string{"System message"}
	userPrompts := []string{"User message"}
	assistantPrompt := []string{"Assistant message"}

	expectedRequest := openai.ChatCompletionRequest{
		Messages: []openai.ChatCompletionMessage{
			{Role: openai.ChatMessageRoleSystem, Content: "System message"},
			{Role: openai.ChatMessageRoleUser, Content: "User message"},
			{Role: openai.ChatMessageRoleAssistant, Content: "Assistant message"},
		},
	}

	openaiMock.
		On("BuildRequest", systemPrompts, userPrompts, assistantPrompt).
		Return(expectedRequest)

	result := openaiMock.BuildRequest(systemPrompts, userPrompts, assistantPrompt)

	require.Equal(t, expectedRequest, result)
}
