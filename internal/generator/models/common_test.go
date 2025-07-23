package models

import (
	"github.com/stretchr/testify/require"
	"testing"
)

func TestExtractValuesFromTemplate(t *testing.T) {
	type testCase struct {
		name     string
		template string
		expected []string
	}

	testCases := []testCase{
		{
			name:     "Empty template",
			template: "",
			expected: nil,
		},
		{
			name:     "Valid template",
			template: "{{ foo }}.{{boo}}",
			expected: []string{"foo", "boo"},
		},
		{
			name:     "Template with filters",
			template: "{{ foo | upper | lower }}",
			expected: []string{"foo"},
		},
		{
			name:     "Template with functions",
			template: "{{ upper('foo') | lower }}@{{ boo }}",
			expected: []string{"boo"},
		},
		{
			name:     "Invalid template",
			template: "{_{ foo }}",
			expected: nil,
		},
	}

	testFunc := func(t *testing.T, tc testCase) {
		t.Helper()

		actual := extractValuesFromTemplate(tc.template)
		require.Equal(t, tc.expected, actual)
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) { testFunc(t, tc) })
	}
}

func TestTopologicalSort(t *testing.T) {
	type testCase struct {
		name     string
		columns  []*Column
		wantErr  bool
		expected []string
	}

	testCases := []testCase{
		{
			name:     "Empty columns",
			columns:  []*Column{},
			wantErr:  false,
			expected: []string{},
		},
		{
			name: "Columns with dependencies",
			columns: []*Column{
				{
					Name: "1",
					Type: "string",
					Ranges: []*Params{
						{
							StringParams: &ColumnStringParams{
								Template: "{{ 3 }}",
							},
						},
					},
				},
				{
					Name: "2",
					Type: "string",
					Ranges: []*Params{
						{
							StringParams: &ColumnStringParams{
								Template: "{{ 4 }}",
							},
						},
					},
				},
				{
					Name: "3",
					Type: "string",
					Ranges: []*Params{
						{
							StringParams: &ColumnStringParams{
								Template: "{{ 2 }}",
							},
						},
					},
				},
				{
					Name: "4",
					Type: "string",
					Ranges: []*Params{
						{
							StringParams: &ColumnStringParams{
								Template: "",
							},
						},
					},
				},
			},
			wantErr:  false,
			expected: []string{"4", "2", "3", "1"},
		},
		{
			name: "Columns with cycle dependencies",
			columns: []*Column{
				{
					Name: "1",
					Type: "string",
					Ranges: []*Params{
						{
							StringParams: &ColumnStringParams{
								Template: "{{ 2 }}",
							},
						},
					},
				},
				{
					Name: "2",
					Type: "string",
					Ranges: []*Params{
						{
							StringParams: &ColumnStringParams{
								Template: "{{ 1 }}",
							},
						},
					},
				},
			},
			wantErr:  true,
			expected: nil,
		},
	}

	testFunc := func(t *testing.T, tc testCase) {
		t.Helper()

		actual, err := TopologicalSort(tc.columns)
		require.Equal(t, tc.wantErr, err != nil)
		require.Equal(t, tc.expected, actual)
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) { testFunc(t, tc) })
	}
}
