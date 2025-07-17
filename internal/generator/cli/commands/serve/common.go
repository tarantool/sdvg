package serve

import (
	"io"
	"log/slog"
	"net/http"

	"github.com/labstack/echo/v4"
	"github.com/pkg/errors"

	"sdvg/internal/generator/cli/openai"
	"sdvg/internal/generator/usecase"
)

type handlerOptions struct {
	useCase usecase.UseCase
	openAI  openai.Service
}

type httpHandler func(handlerOptions, echo.Context) error

func toEchoHandler(
	opts handlerOptions,
	handler httpHandler,
) func(echo.Context) error {
	return func(c echo.Context) error {
		return handler(opts, c)
	}
}

// generateConfigRequest type used to describe request to generate config for http client.
type generateConfigRequest struct {
	Format          string `json:"format"`
	DescriptionType string `json:"description_type"`
	Description     string `json:"description"`
}

// response type used to describe response for http client.
type response struct {
	Message string `json:"message,omitempty"`
	Error   string `json:"error,omitempty"`
}

// getRequestBody returns contents of http request body.
func getRequestBody(c echo.Context) ([]byte, error) {
	body, err := io.ReadAll(c.Request().Body)
	if err != nil {
		return nil, errors.Errorf("failed to read request body: %v", err)
	}

	defer func() {
		if err = c.Request().Body.Close(); err != nil {
			slog.Error("unable to close request body", "error", err)
		}
	}()

	return body, nil
}

// sendResponse function sets headers, status code and body for response and send it to client.
func sendResponse(c echo.Context, format string, statusCode int, response any) error {
	var err error

	switch format {
	case "json":
		c.Response().Header().Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
		err = c.JSON(statusCode, response)
	case "string":
		c.Response().Header().Set(echo.HeaderContentType, echo.MIMETextPlainCharsetUTF8)

		strResponse, ok := response.(string)
		if !ok {
			return errors.New("response is not a string")
		}

		err = c.String(statusCode, strResponse)
	}

	if err != nil {
		return errors.Errorf("failed to send response: %v", err)
	}

	return nil
}

func rejectRequestWithMissingLength(next echo.HandlerFunc) echo.HandlerFunc {
	return func(c echo.Context) error {
		if c.Request().ContentLength == -1 {
			return echo.NewHTTPError(http.StatusLengthRequired, "Content-Length header required")
		}

		return next(c)
	}
}

func rejectRequestWithBody(next echo.HandlerFunc) echo.HandlerFunc {
	return func(c echo.Context) error {
		req := c.Request()
		if req.ContentLength > 0 {
			return echo.NewHTTPError(http.StatusBadRequest, "request must not have a body")
		}

		if req.ContentLength == -1 {
			buf := make([]byte, 1)

			n, err := req.Body.Read(buf)
			if err == nil && n > 0 {
				return echo.NewHTTPError(http.StatusBadRequest, "request must not have a body")
			}
		}

		return next(c)
	}
}
