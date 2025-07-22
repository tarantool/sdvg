package config

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/pkg/errors"
	"github.com/tarantool/sdvg/internal/generator/cli/utils"
	"github.com/tarantool/sdvg/internal/generator/models"
)

// generationMode type is used to describe generation modes.
type generationMode int

const (
	description generationMode = iota
	sqlQuery
	dataSample
)

const (
	yamlFormat = "yaml"
	jsonFormat = "json"
)

// generate generates generation config using passed mode.
func generate(ctx context.Context, opts *generateConfigOptions, mode generationMode) error {
	err := getPathToSaveGenerationConfig(ctx, opts)
	if err != nil {
		return err
	}

	slog.Debug("got a path to save generation config", slog.String("path", opts.generationConfigSavePath))

	format := filepath.Ext(opts.generationConfigSavePath)

	switch format {
	case ".yaml", ".yml":
		format = yamlFormat
	case ".json":
		format = jsonFormat
	}

	var request string

	switch mode {
	case description:
		request, err = getDescriptionRequest(ctx, opts)
	case sqlQuery, dataSample:
		request, err = getSQLOrSampleRequest(ctx, opts, mode)
	}

	if err != nil {
		return err
	}

	slog.Debug("got a request to generate config", slog.String("request", request))

	err = checkAccessToOpenAI(ctx, opts)
	if err != nil {
		return err
	}

	content, err := tryGenerate(ctx, opts, request, format)
	if err != nil {
		return err
	}

	//nolint:mnd
	err = os.WriteFile(opts.generationConfigSavePath, []byte(content), 0644)
	if err != nil {
		return errors.Errorf("failed to write configuration to file: %v", err)
	}

	return nil
}

// getPathToSaveGenerationConfig gets path to save generation config file from generateConfigOptions or user input.
func getPathToSaveGenerationConfig(ctx context.Context, opts *generateConfigOptions) error {
	if opts.generationConfigSavePath == "" {
		filePath, err := opts.renderer.InputMenu(
			ctx,
			"Enter path to save generation config",
			utils.ValidateFileFormat(".yml", ".yaml", ".json"),
		)
		if err != nil {
			return errors.WithMessage(err, "failed to get path to save generation config")
		}

		opts.generationConfigSavePath = filePath
	}

	return nil
}

// getPathToExtraFile gets extra file path from generateConfigOptions or user input.
func getPathToExtraFile(ctx context.Context, opts *generateConfigOptions, title string) error {
	if opts.extraFilePath == "" {
		filePath, err := opts.renderer.InputMenu(
			ctx,
			title,
			utils.ValidateEmptyString(),
		)
		if err != nil {
			return errors.WithMessage(err, "failed to get path to extra file")
		}

		opts.extraFilePath = filePath
	}

	return nil
}

// getDescriptionRequest returns the generation request described by the user.
func getDescriptionRequest(ctx context.Context, opts *generateConfigOptions) (string, error) {
	const title = `# Please provide a detailed description for generating a configuration.
# Include any specific requirements, data types, formats, and any other relevant information
# that will help in creating an accurate configuration for the generator.`

	desc, err := opts.renderer.TextMenu(ctx, title)
	if err != nil {
		return "", errors.WithMessage(err, "failed to get description")
	}

	request := strings.Join([]string{"Словесное описание", desc}, "\n")

	return strings.TrimSpace(request), nil
}

// getSQLOrSampleRequest returns the generation request from file containing sql query or data samples.
func getSQLOrSampleRequest(ctx context.Context, opts *generateConfigOptions, mode generationMode) (string, error) {
	const (
		fileTitle = "# Please provide clarifying information for generating a configuration."
		bufSize   = 3072
	)

	var (
		requestTitle string
		inputTitle   string
	)

	if mode == sqlQuery {
		requestTitle = "SQL запрос"
		inputTitle = "Enter path to file containing SQL query"
	} else {
		requestTitle = "Пример данных"
		inputTitle = "Enter path to file containing data samples"
	}

	err := getPathToExtraFile(ctx, opts, inputTitle)
	if err != nil {
		return "", err
	}

	slog.Info("got a path to extra file", slog.String("path", opts.extraFilePath))

	content, err := readFile(opts.extraFilePath, bufSize)
	if err != nil {
		return "", errors.WithMessage(err, "failed to read extra file")
	}

	if opts.extraInput {
		extraInput, err := opts.renderer.TextMenu(ctx, fileTitle)
		if err != nil {
			return "", errors.WithMessage(err, "failed to get clarifying information")
		}

		content = strings.Join([]string{content, "Уточняющая информация", extraInput}, "\n")
	}

	request := strings.Join([]string{requestTitle, content}, "\n")

	return strings.TrimSpace(request), nil
}

// checkAccessToOpenAI checks access to the Open AI service.
func checkAccessToOpenAI(ctx context.Context, opts *generateConfigOptions) error {
	const timeout = 5 * time.Second

	var err error

	fn := func() {
		timeoutCtx, cancel := context.WithTimeout(ctx, timeout)
		defer cancel()

		err = opts.openAI.Ping(timeoutCtx)
	}

	opts.renderer.WithSpinner("Checking access to OpenAI API...", fn)

	if err != nil {
		return err
	}

	return nil
}

// tryGenerate returns generation config generated by Open AI based on the passed request.
// Performs 3 generation attempts.
func tryGenerate(ctx context.Context, opts *generateConfigOptions, request string, format string) (string, error) {
	const maxAttempts = 3

	var (
		attempt         int
		contextMessages []string
		response        string
		err             error
	)

	fn := func() {
		if attempt == 1 {
			response, err = opts.openAI.GenerateConfig(ctx, format, request)
		} else {
			response, err = opts.openAI.RegenerateConfig(ctx, format, request, err.Error(), contextMessages...)
		}
	}

	for attempt < maxAttempts {
		attempt++

		opts.renderer.WithSpinner(fmt.Sprintf("Generating config, attempt %d...", attempt), fn)

		if err != nil {
			return "", err
		}

		var generationConfig models.GenerationConfig

		if format == yamlFormat {
			err = generationConfig.ParseFromYAML([]byte(response))
		} else {
			err = generationConfig.ParseFromJSON([]byte(response))
		}

		if err == nil {
			break
		}

		contextMessages = append(contextMessages, response)
	}

	slog.Info("generation finished", slog.Any("attempts", attempt))

	if err != nil {
		slog.Warn(
			"generated configuration contains errors, you can fix them yourself or try to generate again,",
			slog.String("error", err.Error()),
		)
	}

	return strings.TrimSpace(response), nil
}

// readFile reads part of file the size of bufferSize.
func readFile(filePath string, bufferSize int) (string, error) {
	f, err := os.OpenFile(filePath, os.O_RDONLY|os.O_SYNC, 0)
	if err != nil {
		return "", errors.New(err.Error())
	}

	buffer := make([]byte, bufferSize)

	n, err := f.Read(buffer)
	if err != nil {
		return "", errors.New(err.Error())
	}

	return string(buffer[:n]), nil
}
