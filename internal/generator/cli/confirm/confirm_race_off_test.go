//go:build !race

package confirm

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestConfirmTTY(t *testing.T) {
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
			input := bytes.Buffer{}
			output := bytes.Buffer{}

			confirm := BuildConfirmTTY(&input, &output)

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
		})
	}
}
