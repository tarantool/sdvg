package backup

import (
	"fmt"
	"reflect"
	"time"

	"github.com/google/uuid"
)

// extractBackupFields walks over a cfg value and returns a map exactly in the
// shape expected by the “backup” file: only fields tagged `backup:"true"` are
// kept, integers outside JSON’s safe range are converted to strings, times are
// RFC 3339, UUIDs are canonical strings, etc.
func extractBackupFields(cfg any) map[string]any {
	v := reflect.ValueOf(cfg)

	raw := extractRecursive(v)
	if m, ok := raw.(map[string]any); ok {
		return m
	}

	return map[string]any{}
}

// extractRecursive is the workhorse that performs the recursive traversal and
// normalisation rules described above. It returns an `any` that can be
// marshalled to JSON without losing information.
func extractRecursive(v reflect.Value) any {
	for v.IsValid() && (v.Kind() == reflect.Pointer || v.Kind() == reflect.Interface) {
		if v.IsNil() {
			return nil
		}

		v = v.Elem()
	}

	if !v.IsValid() {
		return nil
	}

	switch v.Kind() {
	case reflect.Struct:
		return extractStruct(v)

	case reflect.Map:
		return extractMap(v)

	case reflect.Slice, reflect.Array:
		return extractSliceAndArray(v)
	}

	return v.Interface()
}

// extractStruct processes a single struct value: it honours the `backup:"true"`
// tag, supports anonymous inline structs, converts time.Time to RFC3339, and
// recurses into every eligible field.
//
//nolint:forcetypeassert
func extractStruct(v reflect.Value) any {
	if v.Type() == reflect.TypeFor[time.Time]() {
		return v.Interface().(time.Time).Format(time.RFC3339)
	}

	out := make(map[string]any)
	t := v.Type()

	for i := range t.NumField() {
		f := t.Field(i)
		if f.Tag.Get("backup") != "true" {
			continue
		}

		name, inline := parseJSONTag(f)
		val := extractRecursive(v.Field(i))

		if inline {
			if inner, ok := val.(map[string]any); ok {
				for k, iv := range inner {
					out[k] = iv
				}
			}

			continue
		}

		out[name] = val
	}

	return out
}

// extractMap copies the contents of a map into a fresh map[string]any,
// recursively normalising every value.
func extractMap(v reflect.Value) any {
	out := make(map[string]any, v.Len())
	iter := v.MapRange()

	for iter.Next() {
		k := fmt.Sprint(iter.Key().Interface())
		out[k] = extractRecursive(iter.Value())
	}

	return out
}

// extractSliceAndArray converts slices/arrays to []any, turning UUID elements
// into strings and recursing into everything else.
//
//nolint:forcetypeassert
func extractSliceAndArray(v reflect.Value) any {
	if v.Type() == reflect.TypeFor[uuid.UUID]() {
		return v.Interface().(uuid.UUID).String()
	}

	n := v.Len()
	if n == 0 {
		return nil
	}

	out := make([]any, n)
	for i := range n {
		out[i] = extractRecursive(v.Index(i))
	}

	return out
}
