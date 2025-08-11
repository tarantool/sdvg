package confirm

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	rendererMock "github.com/tarantool/sdvg/internal/generator/cli/render/mock"
)

func TestConfirmTTY(t *testing.T) {
	input := bytes.Buffer{}
	output := bytes.Buffer{}

	confirm := BuildConfirmTTY(&input, &output)

	testCases := []struct {
		name        string
		ctx         context.Context
		question    string
		input       string
		expected    bool
		expectedErr error
	}{
		{
			name:     "Y",
			question: "question",
			input:    "Y",
			expected: true,
		},
		{
			name:     "y",
			question: "question",
			input:    "y",
			expected: true,
		},
		{
			name:        "yes",
			question:    "question",
			input:       "yes",
			expectedErr: ErrPromptFailed,
		},
		{
			name:     "N",
			question: "question",
			input:    "N",
			expected: false,
		},
		{
			name:     "n",
			question: "question",
			input:    "n",
			expected: false,
		},
		{
			name:        "no",
			question:    "question",
			input:       "no",
			expectedErr: ErrPromptFailed,
		},
		{
			name:        "Context canceled",
			expectedErr: context.Canceled,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			ctx := context.Background()

			if errors.Is(tc.expectedErr, context.Canceled) {
				var cancel context.CancelFunc

				ctx, cancel = context.WithCancel(ctx)
				cancel()
			}

			input.WriteString(tc.input + "\n")

			res, err := confirm(ctx, tc.question)
			require.True(t, errors.Is(err, tc.expectedErr), fmt.Sprintf("expected: %v, got: %v", tc.expectedErr, err))

			require.Equal(t, tc.expected, res)

			input.Reset()
			output.Reset()
		})
	}
}

var errMockTest = errors.New("mock test error")

func TestConfirmNoTTY(t *testing.T) {
	output := bytes.Buffer{}

	isUpdatePaused := atomic.Bool{}

	testCases := []struct {
		name        string
		ctx         context.Context
		question    string
		ch          chan time.Time
		expected    bool
		expectedErr error
		mockFunc    func(r *rendererMock.Renderer)
	}{
		{
			name:     "Y",
			question: "question",
			expected: true,
			mockFunc: func(r *rendererMock.Renderer) {
				r.On("ReadLine").
					Return("Y"+"\n", nil)

				r.
					On("IsTerminal").
					Return(true)
			},
		},
		{
			name:     "y",
			question: "question",
			expected: true,
			mockFunc: func(r *rendererMock.Renderer) {
				r.On("ReadLine").
					Return("y"+"\n", nil)

				r.
					On("IsTerminal").
					Return(true)
			},
		},
		{
			name:     "yes",
			question: "question",
			mockFunc: func(r *rendererMock.Renderer) {
				r.On("ReadLine").
					Return("yes"+"\n", errMockTest)
			},
			expectedErr: errMockTest,
		},
		{
			name:     "N",
			question: "question",
			mockFunc: func(r *rendererMock.Renderer) {
				r.On("ReadLine").
					Return("N"+"\n", nil)

				r.
					On("IsTerminal").
					Return(true)
			},
			expected: false,
		},
		{
			name:     "n",
			question: "question",
			mockFunc: func(r *rendererMock.Renderer) {
				r.On("ReadLine").
					Return("n"+"\n", nil)

				r.
					On("IsTerminal").
					Return(true)
			},
			expected: false,
		},
		{
			name:     "no",
			question: "question",
			mockFunc: func(r *rendererMock.Renderer) {
				r.On("ReadLine").
					Return("no"+"\n", errMockTest)
			},
			expectedErr: errMockTest,
		},
		{
			name: "Context canceled",
			mockFunc: func(r *rendererMock.Renderer) {
				r.On("ReadLine").
					Return("", nil).Maybe()
			},
			expectedErr: context.Canceled,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			r := rendererMock.NewRenderer(t)

			tc.mockFunc(r)

			confirm := BuildConfirmNoTTY(r, &output, &isUpdatePaused)

			ctx := context.Background()

			if errors.Is(tc.expectedErr, context.Canceled) {
				var cancel context.CancelFunc

				ctx, cancel = context.WithCancel(ctx)
				cancel()
			}

			res, err := confirm(ctx, tc.question)
			require.True(t, errors.Is(err, tc.expectedErr), fmt.Sprintf("expected: %v, got: %v", tc.expectedErr, err))

			require.Equal(t, tc.expected, res)

			output.Reset()
		})
	}
}

func TestConfirmNoTTY_IsUpdatePaused(t *testing.T) {
	output := bytes.Buffer{}

	isUpdatePaused := atomic.Bool{}

	r := rendererMock.NewRenderer(t)

	confirm := BuildConfirmNoTTY(r, &output, &isUpdatePaused)

	mockFunc := func(r *rendererMock.Renderer, ch chan time.Time) {
		r.On("ReadLine").WaitUntil(ch).
			Return("Y"+"\n", nil)

		r.
			On("IsTerminal").
			Return(true)
	}

	ch := make(chan time.Time)

	mockFunc(r, ch)

	go confirm(context.Background(), "")

	start := time.Now()
	ch <- start

	for isUpdatePaused.Load() {
		if time.Now().Sub(start) > 2*time.Second {
			t.Fatal("isUpdatePaused has not been called")
		}
	}
}
