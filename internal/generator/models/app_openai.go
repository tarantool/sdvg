package models

// OpenAI type used to describe OpenAI config.
type OpenAI struct {
	APIKey  string `json:"api_key"  yaml:"api_key"`
	BaseURL string `json:"base_url" yaml:"base_url"`
	Model   string `json:"model"    yaml:"model"`
}
