package backup

import (
	"fmt"
	"reflect"
	"strconv"
	"strings"
)

// valueKind classifies an element we encounter while walking a config value.
// It tells the diff-formatter whether the step is inside a map, slice/array
// or struct so that the path can be rendered as “.field”, “[idx]” or
// "[key]" respectively.
type valueKind uint8

const (
	mapKind valueKind = iota
	sliceKind
	structKind
)

// pathStep is one component of a logical path inside the config/backup object:
//   - key   – map key, slice index or struct field name
//   - kind  – how to format this step when printing a diff path.
type pathStep struct {
	key  string
	kind valueKind
}

// diffEntry represents a single difference found during comparison,
// including the path to the field, an optional marker (like "ADDED"/"REMOVED"),
// and the backup/cfg values.
type diffEntry struct {
	path                  string
	marker                string
	backupValue, cfgValue any
}

// trackDiffValues appends a diff entry showing a value difference between backup and cfg,
// ordering them consistently based on isLeftCfg.
func trackDiffValues(
	left any,
	right any,
	isLeftCfg bool,
	pathSteps []pathStep,
	diffs *[]diffEntry,
) {
	if diffs == nil {
		return
	}

	cfgValue := left
	backupValue := right

	if !isLeftCfg {
		cfgValue, backupValue = backupValue, cfgValue
	}

	*diffs = append(*diffs, diffEntry{
		path:        formatPath(pathSteps),
		backupValue: backupValue,
		cfgValue:    cfgValue,
	})
}

// trackDiffMarker appends a diff entry indicating an addition or removal,
// using only a marker without value details.
func trackDiffMarker(marker string, path []pathStep, diffs *[]diffEntry) {
	if diffs == nil {
		return
	}

	*diffs = append(*diffs, diffEntry{
		path:   formatPath(path),
		marker: marker,
	})
}

// handleMissingValue convenience function to track either "ADDED" or "REMOVED" diffs
// when one side is missing.
func handleMissingValue(isLeftCfg bool, path []pathStep, diffs *[]diffEntry) bool {
	var marker string

	if isLeftCfg {
		marker = "ADDED"
	} else {
		marker = "REMOVED"
	}

	trackDiffMarker(marker, path, diffs)

	return false
}

// formatPath converts a list of pathStep entries into a human-readable string path for diffs.
func formatPath(p []pathStep) string {
	if len(p) == 0 {
		return ""
	}

	var b strings.Builder

	b.WriteString(p[0].key)

	for _, s := range p[1:] {
		switch s.kind {
		case mapKind, sliceKind:
			b.WriteString(fmt.Sprintf("[%v]", s.key))
		case structKind:
			b.WriteString(fmt.Sprintf(".%v", s.key))
		}
	}

	return b.String()
}

// formatValue formats a single value for display in the diff output, quoting strings and formatting floats.
func formatValue(v any) string {
	switch val := v.(type) {
	case string:
		return fmt.Sprintf("%q", val)
	case float32:
		return strconv.FormatFloat(float64(val), 'f', -1, 32)
	case float64:
		return strconv.FormatFloat(val, 'f', -1, 64)
	default:
		return fmt.Sprintf("%v", v)
	}
}

// formatDiff builds a human-readable multi-line string summarizing a list of diffEntry differences.
func formatDiff(diffs []diffEntry) string {
	var sb strings.Builder

	sb.WriteString("config changed:\n")

	for _, diff := range diffs {
		if diff.marker != "" {
			sb.WriteString(fmt.Sprintf("%s %s\n", diff.path, diff.marker))
		} else {
			sb.WriteString(fmt.Sprintf(
				"%s %s -> %s\n",
				diff.path,
				formatValue(diff.backupValue),
				formatValue(diff.cfgValue),
			))
		}
	}

	return strings.TrimSpace(sb.String())
}

// parseJSONTag parses a struct field’s json tag to get the key name and determine if it uses "inline".
func parseJSONTag(sf reflect.StructField) (string, bool) {
	tag := sf.Tag.Get("json")
	if tag == "-" {
		return sf.Name, false
	}

	parts := strings.Split(tag, ",")
	name := parts[0]

	inline := false

	for _, opt := range parts[1:] {
		if opt == "inline" {
			inline = true
		}
	}

	if name == "" && !inline {
		name = sf.Name
	}

	return name, inline
}

// sliceIsPrimitive checks whether every element in a slice is a primitive value.
func sliceIsPrimitive(s reflect.Value) bool {
	for i := range s.Len() {
		if !valueIsPrimitive(s.Index(i)) {
			return false
		}
	}

	return true
}

// valueIsPrimitive determines if a value is primitive (numbers, booleans, strings)
// after unwrapping pointers/interfaces.
func valueIsPrimitive(v reflect.Value) bool {
	if !v.IsValid() {
		return true
	}

	v = unwrapPointersOrInterfaces(v)

	if isNil(v) || !v.IsValid() {
		return true
	}

	switch v.Kind() {
	case reflect.Bool,
		reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64,
		reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64,
		reflect.Float32, reflect.Float64,
		reflect.String:
		return true
	default:
		return false
	}
}

// isIntegerKind checks whether a [reflect.Kind] represents an integer type (signed or unsigned).
func isIntegerKind(k reflect.Kind) bool {
	switch k {
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64,
		reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		return true
	default:
		return false
	}
}
