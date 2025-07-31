package http

import (
	"encoding/json"
	"fmt"
	"reflect"
	"strings"
)

func toJSON(v any) (string, error) {
	data, err := json.Marshal(v)

	return string(data), err
}

func length(v any) int {
	return reflect.ValueOf(v).Len()
}

func rowsJSON(columnNames []string, rows [][]any) (string, error) {
	var sb strings.Builder

	sb.WriteByte('[')

	for i, row := range rows {
		if i > 0 {
			sb.WriteByte(',')
		}

		sb.WriteByte('{')

		for j, columnName := range columnNames {
			if j > 0 {
				sb.WriteByte(',')
			}

			value, err := toJSON(row[j])
			if err != nil {
				return "", err
			}

			fmt.Fprintf(&sb, `"%s":%s`, columnName, value)
		}

		sb.WriteByte('}')
	}

	sb.WriteByte(']')

	return sb.String(), nil
}
