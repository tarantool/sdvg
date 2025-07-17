package commands

import (
	"io"
	"testing"

	"github.com/pkg/errors"
	"github.com/spf13/cobra"
	"github.com/stretchr/testify/require"
)

type testCase struct {
	name          string
	args          []string
	validateFunc  cobra.PositionalArgs
	expectedError string
}

func newDummyCommand(validationFunc cobra.PositionalArgs) *cobra.Command {
	cmd := &cobra.Command{
		Use:  "dummy",
		Args: validationFunc,
		RunE: func(_ *cobra.Command, _ []string) error {
			return errors.New("no error")
		},
	}
	cmd.SetOut(io.Discard)
	cmd.SetErr(io.Discard)

	return cmd
}

func TestNoArgs(t *testing.T) {
	testCases := []testCase{
		{
			name:          "Without args",
			args:          []string{},
			validateFunc:  NoArgs,
			expectedError: "no error",
		},
		{
			name:          "With args",
			args:          []string{"foo"},
			validateFunc:  NoArgs,
			expectedError: "accepts no arguments",
		},
	}

	testFunc := func(t *testing.T, tc testCase) {
		t.Helper()

		cmd := newDummyCommand(tc.validateFunc)
		cmd.SetArgs(tc.args)
		require.ErrorContains(t, cmd.Execute(), tc.expectedError)
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) { testFunc(t, tc) })
	}
}

func TestRequiresMaxArgs(t *testing.T) {
	testCases := []testCase{
		{
			name:          "Without args",
			args:          []string{},
			validateFunc:  RequiresMaxArgs(0),
			expectedError: "no error",
		},
		{
			name:          "With 2 args",
			args:          []string{"foo", "bar"},
			validateFunc:  RequiresMaxArgs(1),
			expectedError: "at most 1 argument",
		},
		{
			name:          "With 3 args",
			args:          []string{"foo", "bar", "baz"},
			validateFunc:  RequiresMaxArgs(2),
			expectedError: "at most 2 arguments",
		},
	}

	testFunc := func(t *testing.T, tc testCase) {
		t.Helper()

		cmd := newDummyCommand(tc.validateFunc)
		cmd.SetArgs(tc.args)
		require.ErrorContains(t, cmd.Execute(), tc.expectedError)
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) { testFunc(t, tc) })
	}
}

func TestPluralize(t *testing.T) {
	type testCase struct {
		name     string
		word     string
		number   int
		expected string
	}

	testCases := []testCase{
		{
			name:     "Singular",
			word:     "argument",
			number:   1,
			expected: "argument",
		},
		{
			name:     "Plural",
			word:     "argument",
			number:   2,
			expected: "arguments",
		},
	}

	testFunc := func(t *testing.T, tc testCase) {
		t.Helper()

		actual := pluralize(tc.word, tc.number)
		require.Equal(t, tc.expected, actual)
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) { testFunc(t, tc) })
	}
}
