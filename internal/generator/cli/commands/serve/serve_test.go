package serve

import (
	"context"
	"net/http"
	"os"
	"testing"
	"time"

	"github.com/spf13/pflag"
	"github.com/stretchr/testify/require"
	"github.com/tarantool/sdvg/internal/generator/cli/commands"
	openaiMock "github.com/tarantool/sdvg/internal/generator/cli/openai/mock"
	"github.com/tarantool/sdvg/internal/generator/cli/options"
	"github.com/tarantool/sdvg/internal/generator/cli/streams"
	"github.com/tarantool/sdvg/internal/generator/models"
	usecaseMock "github.com/tarantool/sdvg/internal/generator/usecase/mock"
)

func TestSetupFlags(t *testing.T) {
	t.Helper()

	expectedFlags := []string{
		commands.HTTPListenAddressFlag,
		commands.HTTPReadTimeoutFlag,
		commands.HTTPWriteTimeoutFlag,
		commands.HTTPIdleTimeoutFlag,
	}

	flagSet := pflag.NewFlagSet("", pflag.ExitOnError)
	opts := &serveOptions{}

	setupFlags(flagSet, opts)

	var actualFlags []string

	flagSet.VisitAll(func(flag *pflag.Flag) {
		actualFlags = append(actualFlags, flag.Name)
	})

	require.ElementsMatch(t, expectedFlags, actualFlags)
}

func TestConfigureOptions(t *testing.T) {
	type testCase struct {
		name       string
		opts       *serveOptions
		httpConfig models.HTTPConfig
		expected   serveOptions
	}

	testCases := []testCase{
		{
			name: "Default values",
			opts: &serveOptions{},
			httpConfig: models.HTTPConfig{
				ListenAddress: ":8080",
				ReadTimeout:   60,
				WriteTimeout:  60,
				IdleTimeout:   60,
			},
			expected: serveOptions{
				listenAddress: ":8080",
				readTimeout:   60,
				writeTimeout:  60,
				idleTimeout:   60,
			},
		},
		{
			name: "Custom values",
			opts: &serveOptions{
				listenAddress: ":5050",
				readTimeout:   30,
				writeTimeout:  30,
				idleTimeout:   30,
			},
			httpConfig: models.HTTPConfig{
				ListenAddress: ":8080",
				ReadTimeout:   60,
				WriteTimeout:  60,
				IdleTimeout:   60,
			},
			expected: serveOptions{
				listenAddress: ":5050",
				readTimeout:   30,
				writeTimeout:  30,
				idleTimeout:   30,
			},
		},
		{
			name: "Merge custom and default values",
			opts: &serveOptions{
				listenAddress: ":5050",
				readTimeout:   30,
			},
			httpConfig: models.HTTPConfig{
				WriteTimeout: 60,
				IdleTimeout:  60,
			},
			expected: serveOptions{
				listenAddress: ":5050",
				readTimeout:   30,
				writeTimeout:  60,
				idleTimeout:   60,
			},
		},
	}

	testFunc := func(t *testing.T, tc testCase) {
		t.Helper()

		configureOptions(tc.opts, tc.httpConfig)

		require.Equal(t, tc.expected, *tc.opts)
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) { testFunc(t, tc) })
	}
}

func TestRunServe(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	uc := usecaseMock.NewUseCase(t)
	openAI := openaiMock.NewOpenAIService(t)

	opts := &serveOptions{
		useCase:       uc,
		openAI:        openAI,
		listenAddress: ":8080",
		readTimeout:   1 * time.Second,
		writeTimeout:  1 * time.Second,
		idleTimeout:   1 * time.Second,
	}

	errCh := make(chan error, 1)
	go func() {
		errCh <- runServe(ctx, opts)
	}()

	eventuallyFunc := func() bool {
		//nolint:noctx
		resp, err := http.Get("http://" + opts.listenAddress)
		if err != nil {
			return false
		}

		_ = resp.Body.Close()

		return true
	}

	require.Eventually(t, eventuallyFunc, 1*time.Second, 100*time.Millisecond)

	cancel()

	err := <-errCh
	require.NoError(t, err)
}

func TestNewServeCommand(t *testing.T) {
	type testCase struct {
		name          string
		addr          string
		expectedError bool
	}

	testCases := []testCase{
		{
			name:          "Valid address",
			addr:          ":8080",
			expectedError: false,
		},
		{
			name:          "Invalid address",
			addr:          "8080",
			expectedError: true,
		},
	}

	testFunc := func(t *testing.T, tc testCase) {
		t.Helper()

		cliOpts := &options.CliOptions{}
		cliOpts.SetOut(streams.NewOut(os.Stdout))
		cliOpts.SetAppConfig(&models.AppConfig{
			HTTPConfig: models.HTTPConfig{
				ListenAddress: tc.addr,
				ReadTimeout:   1 * time.Second,
				WriteTimeout:  1 * time.Second,
				IdleTimeout:   1 * time.Second,
			},
		})

		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()

		errCh := make(chan error, 1)

		cmd := NewServeCommand(cliOpts)
		cmd.SetArgs([]string{})

		go func() {
			errCh <- cmd.ExecuteContext(ctx)
		}()

		time.AfterFunc(1*time.Second, cancel)

		err := <-errCh

		require.Equal(t, tc.expectedError, err != nil)
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) { testFunc(t, tc) })
	}
}
