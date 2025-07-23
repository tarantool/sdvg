package models

import (
	"encoding/json"
	"github.com/otaviokr/topological-sort/toposort"
	"io"
	"os"
	"path/filepath"
	"reflect"
	"regexp"
	"strings"

	"github.com/ilyakaznacheev/cleanenv"
	"github.com/pkg/errors"
	"gopkg.in/yaml.v3"
)

// Field interface is used to summarize model field methods.
type Field interface {
	// Parse function should parse all fields with "any" type
	Parse() error
	// FillDefaults function should fill all default values
	FillDefaults()
	// Validate function should validate all values and return all list of all occurred errors
	Validate() []error
}

func FieldParse(field Field) error {
	if !reflect.ValueOf(field).IsNil() {
		if err := field.Parse(); err != nil {
			return err
		}
	}

	return nil
}

func FieldFillDefaults(field Field) {
	if !reflect.ValueOf(field).IsNil() {
		field.FillDefaults()
	}
}

func FieldValidate(field Field) []error {
	if !reflect.ValueOf(field).IsNil() {
		if err := field.Validate(); err != nil {
			return err
		}
	}

	return nil
}

func DecodeFile(path string, v any) error {
	f, err := os.OpenFile(path, os.O_RDONLY|os.O_SYNC, 0)
	if err != nil {
		return errors.New(err.Error())
	}
	defer f.Close()

	switch ext := strings.ToLower(filepath.Ext(path)); ext {
	case ".yaml", ".yml":
		err = DecodeReader("yaml", f, v)
	case ".json":
		err = DecodeReader("json", f, v)
	default:
		return errors.Errorf("unknown file format %q", ext)
	}

	if err != nil {
		return err
	}

	return nil
}

func DecodeReader(format string, r io.Reader, v any) error {
	var err error

	switch format {
	case "yaml":
		decoder := yaml.NewDecoder(r)
		decoder.KnownFields(true)
		err = decoder.Decode(v)
	case "json":
		decoder := json.NewDecoder(r)
		decoder.DisallowUnknownFields()
		err = decoder.Decode(v)
	default:
		return errors.Errorf("format %q doesn't supported", format)
	}

	if err != nil {
		return errors.New(err.Error())
	}

	err = cleanenv.ReadEnv(v)
	if err != nil {
		return errors.New(err.Error())
	}

	return nil
}

func parseErrsToString(errs []error) string {
	var sb strings.Builder

	for i, err := range errs {
		v := err.Error()

		if !strings.HasSuffix(v, ":") {
			sb.WriteString("- ")
		}

		sb.WriteString(v)

		if i != len(errs)-1 {
			sb.WriteString("\n")
		}
	}

	return sb.String()
}

func TopologicalSort(columns []*Column) ([]string, error) {
	graph := make(map[string][]string)
	for _, c := range columns {
		graph[c.Name] = make([]string, 0)

		if c.Type != "string" {
			continue
		}

		for _, r := range c.Ranges {
			if r.StringParams.Template == "" {
				continue
			}

			graph[c.Name] = extractValuesFromTemplate(r.StringParams.Template)
		}
	}

	sortedVertexes, err := toposort.ReverseTarjan(graph)
	if err != nil {
		return nil, err
	}

	return sortedVertexes, nil
}

func extractValuesFromTemplate(template string) []string {
	re := regexp.MustCompile(`{{\s*([^}]+)\s*}}`)
	matches := re.FindAllStringSubmatch(template, -1)

	var values []string
	for _, match := range matches {
		expr := match[1]

		parts := regexp.MustCompile(`\s*\|\s*|\s+`).Split(expr, -1)
		if len(parts) > 0 && parts[0] != "" && !strings.Contains(parts[0], "(") {
			values = append(values, parts[0])
		}
	}

	return values
}
