package serve

import (
	"bytes"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/labstack/echo/v4"
	"github.com/pkg/errors"
	"github.com/stretchr/testify/require"
)

type errorReader struct{}

func (e *errorReader) Read(_ []byte) (int, error) {
	return 0, errors.New("read error")
}

func TestGetRequestBody(t *testing.T) {
	type testCase struct {
		name          string
		body          io.Reader
		expectError   bool
		expectContent []byte
	}

	testCases := []testCase{
		{
			name:          "Valid request body",
			body:          bytes.NewBufferString("test body"),
			expectError:   false,
			expectContent: []byte("test body"),
		},
		{
			name:          "Empty request body",
			body:          nil,
			expectError:   false,
			expectContent: []byte{},
		},
		{
			name:          "Error reading body",
			body:          &errorReader{},
			expectError:   true,
			expectContent: nil,
		},
	}

	testFunc := func(t *testing.T, tc testCase) {
		t.Helper()

		e := echo.New()
		req := httptest.NewRequest(http.MethodPost, "/", tc.body)
		res := httptest.NewRecorder()
		c := e.NewContext(req, res)

		body, err := getRequestBody(c)

		require.Equal(t, tc.expectError, err != nil)
		require.Equal(t, tc.expectContent, body)
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) { testFunc(t, tc) })
	}
}

func TestSendResponse(t *testing.T) {
	type testCase struct {
		name        string
		format      string
		statusCode  int
		response    any
		expectError bool
		expectBody  string
		contentType string
	}

	testCases := []testCase{
		{
			name:        "Valid JSON response",
			format:      "json",
			statusCode:  http.StatusOK,
			response:    map[string]string{"message": "success"},
			expectError: false,
			expectBody:  "{\"message\":\"success\"}\n",
			contentType: echo.MIMEApplicationJSON,
		},
		{
			name:        "Valid string response",
			format:      "string",
			statusCode:  http.StatusOK,
			response:    "response",
			expectError: false,
			expectBody:  "response",
			contentType: echo.MIMETextPlainCharsetUTF8,
		},
		{
			name:        "Invalid string response",
			format:      "string",
			statusCode:  http.StatusOK,
			response:    123,
			expectError: true,
			contentType: echo.MIMETextPlainCharsetUTF8,
		},
	}

	testFunc := func(t *testing.T, tc testCase) {
		t.Helper()

		e := echo.New()
		req := httptest.NewRequest(http.MethodGet, "/", nil)
		res := httptest.NewRecorder()
		c := e.NewContext(req, res)

		err := sendResponse(c, tc.format, tc.statusCode, tc.response)

		require.Equal(t, tc.expectError, err != nil)
		require.Equal(t, tc.statusCode, res.Code)
		require.Equal(t, tc.expectBody, res.Body.String())
		require.Equal(t, tc.contentType, res.Header().Get(echo.HeaderContentType))
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) { testFunc(t, tc) })
	}
}
