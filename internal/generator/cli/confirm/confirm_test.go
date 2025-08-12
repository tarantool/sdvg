package confirm

import (
	"bytes"
	"context"
	"sync/atomic"
	"testing"
	"time"

	"github.com/pkg/errors"
	"github.com/stretchr/testify/require"
	rendererMock "github.com/tarantool/sdvg/internal/generator/cli/render/mock"
)

var errMockTest = errors.New("mock test error")

func TestConfirmNoTTY(t *testing.T) {
	testCases := []struct {
		name        string
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

			output := bytes.Buffer{}
			isUpdatePaused := atomic.Bool{}

			confirm := BuildConfirmNoTTY(r, &output, &isUpdatePaused)

			ctx := context.Background()

			if errors.Is(tc.expectedErr, context.Canceled) {
				var cancel context.CancelFunc

				ctx, cancel = context.WithCancel(ctx)
				cancel()
			}

			res, err := confirm(ctx, tc.question)
			require.ErrorIs(t, err, tc.expectedErr, "expected: %v, got: %v", tc.expectedErr, err)

			require.Equal(t, tc.expected, res)
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

	//nolint:errcheck
	go confirm(context.Background(), "")

	start := time.Now()
	ch <- start

	for isUpdatePaused.Load() {
		if time.Since(start) > 2*time.Second {
			t.Fatal("isUpdatePaused has not been called")
		}
	}
}
