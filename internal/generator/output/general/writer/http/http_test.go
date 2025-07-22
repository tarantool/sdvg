package http

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/tarantool/sdvg/internal/generator/models"
)

func TestHandleRowsBatch(t *testing.T) {
	type testCase struct {
		name         string
		bodyTemplate string
		model        *models.Model
		rows         []*models.DataRow
		expected     string
	}

	testCases := []testCase{
		{
			name:         "Default body template",
			bodyTemplate: "",
			model: &models.Model{
				Name:      "expectedModel",
				RowsCount: 1,
				Columns: []*models.Column{
					{Name: "id", Type: "integer"},
					{Name: "name", Type: "string"},
				},
			},
			rows: []*models.DataRow{
				{Values: []any{1, "test"}},
			},
			expected: `
{
	"table_name": "expectedModel",
	"rows": [
		{
			"id": 1,
			"name": "test"
		}
	]
}
`,
		},
		{
			name: "Custom body template",
			bodyTemplate: `{
	"table_name": "{{ .ModelName }}",
	"meta": {
		"rows_count": {{ len .Rows }}
	},
	"rows": {{ json .Rows }}
}`,
			model: &models.Model{
				Name:      "expectedModel",
				RowsCount: 4,
				Columns: []*models.Column{
					{Name: "id", Type: "integer"},
					{Name: "name", Type: "string"},
				},
			},
			rows: []*models.DataRow{
				{Values: []any{1, "value1"}},
				{Values: []any{2, "value2"}},
				{Values: []any{3, "value3"}},
				{Values: []any{4, "value4"}},
			},
			expected: `
{
	"table_name": "expectedModel",
	"meta": {
		"rows_count": 4
	},
	"rows": [
		{
			"id": 1,
			"name": "value1"
		},
		{
			"id": 2,
			"name": "value2"
		},
		{
			"id": 3,
			"name": "value3"
		},
		{
			"id": 4,
			"name": "value4"
		}
	]
}
`,
		},
	}

	testFunc := func(t *testing.T, tc testCase) {
		t.Helper()

		server := httptest.NewServer(
			http.HandlerFunc(
				func(w http.ResponseWriter, r *http.Request) {
					assert.Equal(t, http.MethodPost, r.Method, "expected POST method")

					assert.Equal(
						t, "application/json", r.Header.Get("Content-Type"), "expected application/json content type",
					)

					body, err := io.ReadAll(r.Body)
					assert.NoError(t, err)

					defer r.Body.Close()

					assert.JSONEq(t, tc.expected, string(body))

					w.WriteHeader(http.StatusOK)
				},
			),
		)
		defer server.Close()

		cfg := &models.HTTPParams{
			Endpoint:       server.URL + "/test",
			FormatTemplate: tc.bodyTemplate,
		}

		prepareField(t, cfg)
		prepareField(t, tc.model)

		writer := NewWriter(context.Background(), tc.model, cfg, nil)
		require.NoError(t, writer.Init())

		for _, row := range tc.rows {
			require.NoError(t, writer.WriteRow(row))
		}

		require.NoError(t, writer.Teardown())
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) { testFunc(t, tc) })
	}
}

func prepareField(t *testing.T, f models.Field) {
	t.Helper()

	require.NoError(t, f.Parse())
	f.FillDefaults()
	require.Empty(t, f.Validate())
}
