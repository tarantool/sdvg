package general

import (
	"context"

	"github.com/pkg/errors"
	"github.com/sashabaranov/go-openai"
	"github.com/tarantool/sdvg/internal/generator/models"
	openAI "github.com/tarantool/sdvg/internal/openai"
)

type OpenAIAPI struct {
	model   string
	baseURL string
	client  *openai.Client
}

// NewOpenAIAPI creates OpenAIAPI object.
func NewOpenAIAPI(cfg models.OpenAI) openAI.API {
	config := openai.DefaultConfig(cfg.APIKey)
	config.BaseURL = cfg.BaseURL

	client := openai.NewClientWithConfig(config)

	return &OpenAIAPI{
		model:   cfg.Model,
		baseURL: cfg.BaseURL,
		client:  client,
	}
}

func (a *OpenAIAPI) GetBaseURL() string {
	return a.baseURL
}

func (a *OpenAIAPI) Models(ctx context.Context) ([]openai.Model, error) {
	listModels, err := a.client.ListModels(ctx)
	if err != nil {
		return nil, errors.Errorf("failed to get openai models: %v", err)
	}

	return listModels.Models, nil
}

func (a *OpenAIAPI) SendRequest(
	ctx context.Context, request openai.ChatCompletionRequest) (openai.ChatCompletionResponse, error) {
	response, err := a.client.CreateChatCompletion(ctx, request)
	if err != nil {
		return openai.ChatCompletionResponse{}, errors.Errorf("failed to send request to openai: %v", err)
	}

	return response, nil
}

func (a *OpenAIAPI) BuildRequest(
	systemPrompts, userPrompts, assistantPrompt []string) openai.ChatCompletionRequest {
	messages := make([]openai.ChatCompletionMessage, 0, len(systemPrompts)+len(userPrompts))

	for _, prompt := range systemPrompts {
		messages = append(messages, openai.ChatCompletionMessage{
			Role:    openai.ChatMessageRoleSystem,
			Content: prompt,
		})
	}

	for _, prompt := range userPrompts {
		messages = append(messages, openai.ChatCompletionMessage{
			Role:    openai.ChatMessageRoleUser,
			Content: prompt,
		})
	}

	for _, prompt := range assistantPrompt {
		messages = append(messages, openai.ChatCompletionMessage{
			Role:    openai.ChatMessageRoleAssistant,
			Content: prompt,
		})
	}

	request := openai.ChatCompletionRequest{
		Model:    a.model,
		Messages: messages,
	}

	return request
}
