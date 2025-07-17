package openai

import (
	"context"

	"github.com/sashabaranov/go-openai"
)

//go:generate go run github.com/vektra/mockery/v2@v2.51.1 --name=API --output=mock --outpkg=mock
type API interface {
	// GetBaseURL should return base URL.
	GetBaseURL() string
	// Models should return available models.
	Models(ctx context.Context) ([]openai.Model, error)
	// SendRequest should send request to OpenAI.
	SendRequest(ctx context.Context, request openai.ChatCompletionRequest) (openai.ChatCompletionResponse, error)
	// BuildRequest should build request form systemPrompt, userPrompt and assistantPrompt.
	BuildRequest(systemPrompts, userPrompts, assistantPrompt []string) openai.ChatCompletionRequest
}
