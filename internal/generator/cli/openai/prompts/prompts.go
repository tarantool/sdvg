package prompts

import (
	_ "embed"
	"log"

	"gopkg.in/yaml.v3"
)

//go:embed prompt.yml
var generateConfigPrompt []byte
var Prompts OpenAIPrompts

type OpenAIPrompts struct {
	GenerationConfigPrompts GenerationConfigPrompts `yaml:"generation_config"`
}

type GenerationConfigPrompts struct {
	System             string `yaml:"system"`
	Format             string `yaml:"format"`
	DefaultValues      string `yaml:"default_values"`
	Rules              string `yaml:"rules"`
	DescriptionExample string `yaml:"description_example"`
	SQLQueryExample    string `yaml:"sql_query_example"`
	SampleDataExample  string `yaml:"sample_data_example"`
	UserMessage        string `yaml:"user_message"`
	RetryMessage       string `yaml:"retry_message"`
}

func init() {
	err := yaml.Unmarshal(generateConfigPrompt, &Prompts)
	if err != nil {
		log.Fatalf("init - parse locale constants: %s", err)
	}
}
