package general

import (
	"context"
	"fmt"
	"strings"

	"github.com/pkg/errors"
	"github.com/sashabaranov/go-openai"

	openaiService "sdvg/internal/generator/cli/openai"
	"sdvg/internal/generator/cli/openai/prompts"
	"sdvg/internal/generator/common"
	openaiAPI "sdvg/internal/openai"
)

// OpenAIService type is implementation of service for OpenAI.
type OpenAIService struct {
	openaiAPI openaiAPI.API
}

// NewOpenAIService creates OpenAIService object.
func NewOpenAIService(openaiAPI openaiAPI.API) openaiService.Service {
	return &OpenAIService{
		openaiAPI: openaiAPI,
	}
}

func (s *OpenAIService) Ping(ctx context.Context) error {
	_, err := s.openaiAPI.Models(ctx)
	if err != nil {
		return errors.New("openai api unreachable")
	}

	return nil
}

// GenerateConfig generates generation config based on its description.
func (s *OpenAIService) GenerateConfig(
	ctx context.Context, format, message string) (string, error) {
	request := s.openaiAPI.BuildRequest(
		[]string{
			strings.Join([]string{
				prompts.Prompts.GenerationConfigPrompts.System,
				prompts.Prompts.GenerationConfigPrompts.Format,
				prompts.Prompts.GenerationConfigPrompts.DefaultValues,
				prompts.Prompts.GenerationConfigPrompts.Rules,
				prompts.Prompts.GenerationConfigPrompts.DescriptionExample,
				prompts.Prompts.GenerationConfigPrompts.SQLQueryExample,
				prompts.Prompts.GenerationConfigPrompts.SampleDataExample,
			}, "\n"),
		},
		[]string{
			fmt.Sprintf(prompts.Prompts.GenerationConfigPrompts.UserMessage, format, message),
		},
		nil,
	)

	config, err := s.sendRequest(ctx, format, request)
	if err != nil {
		return "", err
	}

	return config, nil
}

func (s *OpenAIService) RegenerateConfig(
	ctx context.Context, format, oldConfig, errMessage string, contextMessages ...string) (string, error) {
	request := s.openaiAPI.BuildRequest(
		[]string{
			strings.Join([]string{
				prompts.Prompts.GenerationConfigPrompts.System,
				prompts.Prompts.GenerationConfigPrompts.Format,
				prompts.Prompts.GenerationConfigPrompts.DefaultValues,
				prompts.Prompts.GenerationConfigPrompts.Rules,
				prompts.Prompts.GenerationConfigPrompts.DescriptionExample,
				prompts.Prompts.GenerationConfigPrompts.SQLQueryExample,
				prompts.Prompts.GenerationConfigPrompts.SampleDataExample,
			}, "\n"),
		},
		[]string{
			fmt.Sprintf(prompts.Prompts.GenerationConfigPrompts.RetryMessage, oldConfig, errMessage),
		},
		contextMessages,
	)

	config, err := s.sendRequest(ctx, format, request)
	if err != nil {
		return "", err
	}

	return config, nil
}

func (s *OpenAIService) sendRequest(
	ctx context.Context, format string, request openai.ChatCompletionRequest) (string, error) {
	response, err := s.openaiAPI.SendRequest(ctx, request)
	if err != nil {
		return "", errors.Errorf("failed to receive response from openai: %v", err)
	}

	configuration := response.Choices[0].Message.Content
	configuration = common.Trim(configuration, "```"+format, "```")

	return strings.TrimSpace(configuration), nil
}
