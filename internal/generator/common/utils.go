package common

import (
	"bytes"
	"cmp"
	"context"
	"fmt"
	"os"
	"reflect"
	"regexp"
	"slices"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/otaviokr/topological-sort/toposort"
	"github.com/pkg/errors"
	"gopkg.in/yaml.v3"
)

func Any(flags ...bool) bool {
	for _, flag := range flags {
		if flag {
			return true
		}
	}

	return false
}

func Filter[T any](arr []T, f func(T) bool) []T {
	filtered := make([]T, 0)

	for _, v := range arr {
		if f(v) {
			filtered = append(filtered, v)
		}
	}

	return filtered
}

// AnyToStruct converts map or struct to selected struct.
func AnyToStruct[T any](data any) (*T, error) {
	var res T

	bytesData, err := yaml.Marshal(data)
	if err != nil {
		return &res, errors.New(err.Error())
	}

	decoder := yaml.NewDecoder(bytes.NewReader(bytesData))
	decoder.KnownFields(true)

	err = decoder.Decode(&res)
	if err != nil {
		return &res, errors.New(err.Error())
	}

	return &res, nil
}

// GetKey function returns unique string key separated by point.
func GetKey(prefix, name string) string {
	if prefix == "" {
		prefix = "?"
	}

	if name == "" {
		name = "?"
	}

	return strings.Join([]string{prefix, name}, ".")
}

// Trim returns part of string between prefix and suffix.
func Trim(str, prefix, suffix string) string {
	prefixStart := strings.Index(str, prefix)
	contentStart := prefixStart + len(prefix)
	suffixStart := strings.Index(str[contentStart:], suffix)

	if prefixStart != -1 && suffixStart != -1 {
		str = str[contentStart : contentStart+suffixStart]
	}

	return strings.TrimSpace(str)
}

// SortSlice sorts slice of any, where slice elements can be nil, int, float64, string or time.Time.
func SortSlice(slice []any, comparator CmpFunc) {
	slices.SortFunc(slice, func(a, b any) int {
		if a == nil || b == nil {
			return CmpNil(a, b)
		}

		return comparator(a, b)
	})
}

var ErrDirNotExists = errors.New("directory doesn't exist")

func WalkWithFilter(dir string, filterFunc func(entry os.DirEntry) bool) ([]string, error) {
	fileInfo, err := os.Stat(dir)
	if os.IsNotExist(err) {
		return nil, errors.WithMessagef(ErrDirNotExists, "directory %q", dir)
	}

	if !fileInfo.IsDir() {
		return nil, errors.Errorf("%q is not a directory", dir)
	}

	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, errors.New(err.Error())
	}

	var files []string

	for _, entry := range entries {
		if filterFunc(entry) {
			files = append(files, entry.Name())
		}
	}

	return files, nil
}

type CmpFunc func(x, y any) int

//nolint:forcetypeassert
func CmpString(x, y any) int {
	aStr := x.(string)
	bStr := y.(string)

	return cmp.Compare(aStr, bStr)
}

//nolint:forcetypeassert
func CmpUUID(x, y any) int {
	aUUID := x.(uuid.UUID)
	bUUID := y.(uuid.UUID)

	return cmp.Compare(aUUID.String(), bUUID.String())
}

func CmpInt(x, y any) int {
	aInt := reflect.ValueOf(x).Int()
	bInt := reflect.ValueOf(y).Int()

	return cmp.Compare(aInt, bInt)
}

func CmpFloat(x, y any) int {
	aFloat := reflect.ValueOf(x).Float()
	bFloat := reflect.ValueOf(y).Float()

	return cmp.Compare(aFloat, bFloat)
}

//nolint:forcetypeassert
func CmpTime(x, y any) int {
	aTime := x.(time.Time)
	bTime := y.(time.Time)

	switch {
	case aTime.Before(bTime):
		return -1
	case aTime.After(bTime):
		return 1
	default:
		return 0
	}
}

func CmpNil(x, y any) int {
	switch {
	case x == nil && y == nil:
		return 0
	case x == nil:
		return -1
	default:
		return 1
	}
}

func ConvertToInt(value any, targetType reflect.Type) (any, error) {
	v := reflect.ValueOf(value)

	switch v.Kind() {
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		return v.Convert(targetType).Interface(), nil

	case reflect.Float32, reflect.Float64:
		floatVal := v.Float()
		intVal := int64(floatVal)

		if floatVal != float64(intVal) {
			return nil, errors.Errorf("cannot convert float %v to %s without precision loss", floatVal, targetType)
		}

		return reflect.ValueOf(intVal).Convert(targetType).Interface(), nil

	case reflect.String:
		intVal, err := strconv.Atoi(v.String())
		if err != nil {
			return nil, errors.Errorf("cannot convert string %v to %s", v.String(), targetType)
		}

		return reflect.ValueOf(intVal).Convert(targetType).Interface(), nil
	default:
		return nil, errors.Errorf("cannot convert %T to %s", value, targetType)
	}
}

func ConvertToFloat(value any, targetType reflect.Type) (any, error) {
	v := reflect.ValueOf(value)

	switch v.Kind() {
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64, reflect.Float32, reflect.Float64:
		return v.Convert(targetType).Interface(), nil

	case reflect.String:
		floatVal, err := strconv.ParseFloat(v.String(), targetType.Bits())
		if err != nil {
			return nil, errors.Errorf("cannot convert string %v to %s", v.String(), targetType)
		}

		return reflect.ValueOf(floatVal).Convert(targetType).Interface(), nil

	default:
		return nil, errors.Errorf("cannot convert %T to %s", value, targetType)
	}
}

func ConvertToString(value any, targetType reflect.Type) (any, error) {
	v := reflect.ValueOf(value)

	if v.Kind() == reflect.String {
		return v.Convert(targetType).Interface(), nil
	}

	return fmt.Sprint(value), nil
}

func ConvertToTime(value any, targetType reflect.Type) (any, error) {
	switch val := value.(type) {
	case time.Time:
		return value, nil

	case string:
		t, err := time.Parse(time.RFC3339, val)
		if err != nil {
			return nil, errors.Errorf("cannot convert string %v to %s", val, targetType)
		}

		return t, nil
	case int, int8, int16, int32, int64, float32, float64:
		var sec, nsec int64

		if intValue, ok := val.(int64); ok {
			sec = intValue
		} else if floatValue, ok := val.(float64); ok {
			sec = int64(floatValue)
			nsec = int64((floatValue - float64(sec)) * 1e9) //nolint:mnd
		}

		return time.Unix(sec, nsec), nil
	default:
		return nil, errors.Errorf("cannot convert %T to %s", value, targetType)
	}
}

func ConvertToUUID(value any, targetType reflect.Type) (any, error) {
	switch val := value.(type) {
	case uuid.UUID:
		return val, nil

	case string:
		uuidVal, err := uuid.Parse(val)
		if err != nil {
			return nil, errors.Errorf("cannot convert string %v to %s", val, targetType)
		}

		return uuidVal, nil
	default:
		return nil, errors.Errorf("cannot convert %T to %s", value, targetType)
	}
}

// GetIndicesInOrder returns an array of indexes of the elements from the "original"
// in the order of the elements from the "ordered".
func GetIndicesInOrder[T comparable](original []T, ordered []T) []int {
	if len(original) == 0 {
		return []int{}
	}

	indexMap := make(map[T]int, len(original))
	for i, v := range original {
		indexMap[v] = i
	}

	result := make([]int, len(ordered))

	for i, v := range ordered {
		if idx, ok := indexMap[v]; ok {
			result[i] = idx
		} else {
			result[i] = -1
		}
	}

	return result
}

// ShiftElementsToEnd moves all items with ids equal to orderedShiftedIds to the end in the same order.
// if T is a reference type (struct pointer, interface, ...), then the final slice will contain the original objects.
func ShiftElementsToEnd[T any, K comparable](source []T, orderedShiftedIDs []K, getElemID func(T) K) []T {
	if len(orderedShiftedIDs) == 0 || len(source) == 0 {
		return source
	}

	orderedResult := make([]T, 0, len(source))
	elementsToMove := make(map[K]T)

	for _, element := range source {
		elementID := getElemID(element)
		if slices.Contains(orderedShiftedIDs, elementID) {
			elementsToMove[elementID] = element
		} else {
			orderedResult = append(orderedResult, element)
		}
	}

	for _, id := range orderedShiftedIDs {
		if element, ok := elementsToMove[id]; ok {
			orderedResult = append(orderedResult, element)
		}
	}

	return orderedResult
}

// MakeSet creates new set as map without values.
func MakeSet[T comparable, V any](m1 map[T]V) map[T]struct{} {
	set := make(map[T]struct{}, len(m1))
	for key := range m1 {
		set[key] = struct{}{}
	}

	return set
}

func CtxClosed(ctx context.Context) bool {
	select {
	case <-ctx.Done():
		return true
	default:
		return false
	}
}

func ExtractValuesFromTemplate(template string) []string {
	re := regexp.MustCompile(`{{.*?\.([^\s|}]+).*?}}`)
	matches := re.FindAllStringSubmatch(template, -1)

	values := make([]string, 0, len(matches))

	for _, match := range matches {
		expr := match[0]
		val := match[1]

		if strings.Contains(expr, "(") && strings.Contains(expr, ")") {
			continue
		}

		values = append(values, val)
	}

	return values
}

// TopologicalSort sorts the given items in topological order using the provided
// function to extract node name and dependencies.
// Returns the sorted node names, a flag indicating if any dependencies exist,
// and an error if a cycle is detected.
func TopologicalSort[T any](items []T, nodeFunc func(T) (string, []string)) ([]string, bool, error) {
	var (
		graph           = make(map[string][]string, len(items))
		sortedVertexes  = make([]string, len(items))
		hasDependencies bool
		err             error
	)

	for i, item := range items {
		name, dependencies := nodeFunc(item)
		if len(dependencies) > 0 {
			hasDependencies = true
		}

		sortedVertexes[i] = name
		graph[name] = dependencies
	}

	if !hasDependencies {
		return sortedVertexes, false, nil
	}

	sortedVertexes, err = toposort.ReverseTarjan(graph)
	if err != nil {
		return nil, false, errors.New(err.Error())
	}

	return sortedVertexes, hasDependencies, nil
}
