package serve

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"slices"
	"strings"
	"time"

	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
	"github.com/tarantool/sdvg/internal/generator/cli/utils"
	"github.com/tarantool/sdvg/internal/generator/models"
	"github.com/tarantool/sdvg/internal/generator/output/general"
	"github.com/tarantool/sdvg/internal/generator/usecase"
)

func setupRoutes(opts handlerOptions, e *echo.Echo) {
	e.GET("/status/:taskID", toEchoHandler(opts, handleStatus), rejectRequestWithBody)

	post := e.Group("", rejectRequestWithMissingLength, middleware.BodyLimit("1M"))
	post.POST("/generate", toEchoHandler(opts, handleGenerate))
	post.POST("/generate-config", toEchoHandler(opts, handleGenerateConfig))
	post.POST("/validate-config", toEchoHandler(opts, handleValidate))
}

// handleGenerate handler for endpoint 'generate-data'.
func handleGenerate(opts handlerOptions, c echo.Context) error {
	body, err := getRequestBody(c)
	if err != nil {
		return sendResponse(
			c,
			"json",
			http.StatusInternalServerError,
			response{
				Message: "Unable to read request body",
				Error:   err.Error(),
			},
		)
	}

	var generationConfig models.GenerationConfig

	err = generationConfig.ParseFromJSON(body)
	if err != nil {
		return sendResponse(
			c,
			"json",
			http.StatusBadRequest,
			response{
				Message: "Generation config is not valid",
				Error:   err.Error(),
			},
		)
	}

	generationConfig.OutputConfig.Dir = models.DefaultOutputDir

	out := general.NewOutput(&generationConfig, false, true)

	taskID, err := opts.useCase.CreateTask(
		c.Request().Context(), usecase.TaskConfig{
			GenerationConfig: &generationConfig,
			Output:           out,
			HTTPDelivery:     true,
		},
	)
	if err != nil {
		return sendResponse(
			c,
			"json",
			http.StatusInternalServerError,
			response{
				Message: "Failed to start generation",
				Error:   err.Error(),
			},
		)
	}

	return sendResponse(
		c,
		"string",
		http.StatusOK,
		taskID,
	)
}

// handleValidate handler for endpoint 'validate-config'.
func handleValidate(_ handlerOptions, c echo.Context) error {
	body, err := getRequestBody(c)
	if err != nil {
		return sendResponse(
			c,
			"json",
			http.StatusInternalServerError,
			response{
				Message: "Unable to read request body",
				Error:   err.Error(),
			},
		)
	}

	var generationConfig models.GenerationConfig

	err = generationConfig.ParseFromJSON(body)
	if err != nil {
		return sendResponse(
			c,
			"json",
			http.StatusBadRequest,
			response{
				Message: "Generation config is not valid",
				Error:   err.Error(),
			},
		)
	}

	return sendResponse(
		c,
		"json",
		http.StatusOK,
		response{
			Message: "Generation config is valid",
		},
	)
}

// handleStatus handler for endpoint 'status'.
func handleStatus(opts handlerOptions, c echo.Context) error {
	taskID := c.Param("taskID")

	finished, err := opts.useCase.GetResult(taskID)
	if err != nil {
		return sendResponse(
			c,
			"json",
			http.StatusInternalServerError,
			response{
				Message: "Failed to retrieve generation result",
				Error:   err.Error(),
			},
		)
	}

	if finished {
		return sendResponse(
			c,
			"json",
			http.StatusOK,
			response{
				Message: "Generation completed successfully",
			},
		)
	}

	progresses, err := opts.useCase.GetProgress(taskID)
	if err != nil {
		return sendResponse(
			c,
			"json",
			http.StatusInternalServerError,
			response{
				Message: "Failed to retrieve generation progress",
				Error:   err.Error(),
			},
		)
	}

	progressByKey := make(map[string]uint64, len(progresses))

	for key, progress := range progresses {
		progressByKey[key] = utils.GetPercentage(progress.Total, progress.Done)
	}

	return sendResponse(
		c,
		"json",
		http.StatusOK,
		progressByKey,
	)
}

// handleGenerateConfig handler for endpoint 'generate-config'.
func handleGenerateConfig(opts handlerOptions, c echo.Context) error {
	const timeout = 5 * time.Second

	var request generateConfigRequest

	err := c.Bind(&request)
	if err != nil {
		return sendResponse(
			c,
			"json",
			http.StatusBadRequest,
			response{
				Message: "Invalid request body",
				Error:   err.Error(),
			},
		)
	}

	format := strings.ToLower(request.Format)

	if !slices.Contains([]string{"json", "yaml", "yml"}, format) {
		return sendResponse(
			c,
			"json",
			http.StatusBadRequest,
			response{
				Message: "Unsupported format",
			},
		)
	}

	slog.Debug("got generation config format: " + format)

	description := request.Description

	if request.DescriptionType != "" {
		description = strings.Join([]string{
			fmt.Sprintf("**%s**", request.DescriptionType), description}, "/n",
		)
	}

	slog.Debug("got description to create config:\n " + description)

	timeoutCtx, cancel := context.WithTimeout(c.Request().Context(), timeout)
	defer cancel()

	err = opts.openAI.Ping(timeoutCtx)
	if err != nil {
		return sendResponse(
			c,
			"json",
			http.StatusServiceUnavailable,
			response{
				Message: "OpenAI is not available",
				Error:   err.Error(),
			},
		)
	}

	config, err := opts.openAI.GenerateConfig(c.Request().Context(), format, description)
	if err != nil {
		return sendResponse(
			c,
			"json",
			http.StatusInternalServerError,
			response{
				Message: "Unable to generate config",
				Error:   err.Error(),
			},
		)
	}

	return sendResponse(
		c,
		"string",
		http.StatusOK,
		config,
	)
}
