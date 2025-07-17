package openai

import "context"

//go:generate go run github.com/vektra/mockery/v2@v2.51.1 --name=Service --output=mock --outpkg=mock
type Service interface {
	// Ping should send test request to OpenAI API to check service availability.
	Ping(ctx context.Context) error
	// GenerateConfig should generate models.GenerationConfig based on message passed from user.
	GenerateConfig(ctx context.Context, format, message string) (string, error)
	// RegenerateConfig should try regenerate configuration based on error.
	RegenerateConfig(ctx context.Context, format, oldConfig, errMessage string, contextMessages ...string) (string, error)
}
