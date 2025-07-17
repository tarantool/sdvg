package tcs

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"sdvg/internal/generator/models"
)

func prepareField(t *testing.T, f models.Field) {
	t.Helper()

	require.NoError(t, f.Parse())
	f.FillDefaults()
	require.Empty(t, f.Validate())
}

func TestHandleRowsBatch(t *testing.T) {
	server := httptest.NewServer(
		http.HandlerFunc(
			func(w http.ResponseWriter, r *http.Request) {
				assert.Equal(t, http.MethodPost, r.Method, "expected POST method")

				assert.Equal(
					t, "application/json", r.Header.Get("Content-Type"), "expected application/json content type",
				)

				var body map[string]any
				err := json.NewDecoder(r.Body).Decode(&body)
				assert.NoError(t, err, "failed to decode request body")

				defer r.Body.Close()

				assert.Equal(t, "exampleModel", body["table_name"], "expected table_name to be 'exampleModel'")

				rows, ok := body["rows"].([]any)

				assert.True(t, ok, "expected rows to be an array")
				assert.Len(t, rows, 2)

				for i, rowObj := range rows {
					row, ok := rowObj.(map[string]any)
					assert.True(t, ok)

					strValue := fmt.Sprintf("value%d", i)
					numValue := i

					assert.Equal(t, strValue, row["column1"], "unexpected value for column1")
					assert.EqualValues(t, numValue, row["column2"], "unexpected value for column2")
				}

				w.WriteHeader(http.StatusOK)
			},
		),
	)
	defer server.Close()

	cfg := &models.TCSConfig{
		HTTPParams: models.HTTPParams{
			Endpoint: server.URL + "/test",
		},
	}

	prepareField(t, cfg)

	model := &models.Model{
		Name:      "exampleModel",
		RowsCount: 2,
		Columns: []*models.Column{
			{Name: "column1", Type: "string"},
			{Name: "column2", Type: "integer"},
		},
	}

	prepareField(t, model)

	w := NewWriter(context.Background(), model, cfg, nil)
	require.NoError(t, w.Init())

	rows := []*models.DataRow{
		{Values: []any{"value0", 0}},
		{Values: []any{"value1", 1}},
	}

	for _, row := range rows {
		err := w.WriteRow(row)
		require.NoError(t, err, "failed write row")
	}

	require.NoError(t, w.Teardown())
}
