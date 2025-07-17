package backup

import (
	"fmt"
	"reflect"
	"time"

	"github.com/google/uuid"
)

// compareBackupField starts the deep comparison of two top-level objects,
// returning whether they are equal and a list of differences.
func compareBackupField(cfg, backup any) (bool, []diffEntry) {
	var diffs []diffEntry

	ok := compareRecursive(
		reflect.ValueOf(cfg),
		reflect.ValueOf(backup),
		true,
		nil,
		&diffs,
	)

	return ok, diffs
}

// compareRecursive performs a recursive, deep comparison of two reflect.Values,
// dispatching to appropriate handlers for structs, maps, slices, or primitives.
func compareRecursive(left, right reflect.Value, isLeftCfg bool, path []pathStep, diffs *[]diffEntry) bool {
	if !left.IsValid() {
		if !right.IsValid() {
			return true
		}

		return compareRecursive(right, reflect.Value{}, !isLeftCfg, path, diffs)
	}

	left = unwrapPointersOrInterfaces(left)
	if isNil(left) {
		if !right.IsValid() || isNil(right) {
			return true
		}

		return compareRecursive(right, reflect.Value{}, !isLeftCfg, path, diffs)
	}

	right = unwrapPointersOrInterfaces(right)

	switch left.Kind() {
	case reflect.Struct:
		return compareStruct(left, right, isLeftCfg, path, diffs)

	case reflect.Map:
		return compareMap(left, right, isLeftCfg, path, diffs)

	case reflect.Slice, reflect.Array:
		return compareSliceAndArray(left, right, isLeftCfg, path, diffs)

	default:
		return comparePrimitives(left, right, isLeftCfg, path, diffs)
	}
}

// compareStruct compares two struct values field by field,
// handling special cases like time.Time, and tracking differences.
//
//nolint:forcetypeassert
func compareStruct(left, right reflect.Value, isLeftCfg bool, path []pathStep, diffs *[]diffEntry) bool {
	if !left.IsValid() && !right.IsValid() || isNil(left) && isNil(right) {
		return true
	}

	if !right.IsValid() {
		return handleMissingValue(isLeftCfg, path, diffs)
	}

	if left.Type() != right.Type() {
		trackDiffValues(left.Interface(), right.Interface(), isLeftCfg, path, diffs)

		return false
	}

	if left.Type() == reflect.TypeFor[time.Time]() {
		leftTime := left.Interface().(time.Time)
		rightTime := right.Interface().(time.Time)

		if !leftTime.Equal(rightTime) {
			trackDiffValues(leftTime.Format(time.RFC3339), rightTime.Format(time.RFC3339), isLeftCfg, path, diffs)

			return false
		}

		return true
	}

	return compareStructInner(left, right, isLeftCfg, path, diffs)
}

// compareStructInner helper for compareStruct that performs the actual
// field-by-field comparison inside structs, processing only fields tagged with backup:"true".
//
//nolint:gocritic
func compareStructInner(left, right reflect.Value, isLeftCfg bool, path []pathStep, diffs *[]diffEntry) bool {
	equal := true
	t := left.Type()

	for i := range t.NumField() {
		sf := t.Field(i)
		if sf.Tag.Get("backup") != "true" {
			continue
		}

		key, inline := parseJSONTag(sf)

		fieldValL := left.Field(i)
		fieldValR := right.Field(i)

		var newPath []pathStep
		if inline {
			newPath = path
		} else {
			newPath = append(path, pathStep{key: key, kind: structKind})
		}

		if !compareRecursive(fieldValL, fieldValR, isLeftCfg, newPath, diffs) {
			equal = false
		}
	}

	return equal
}

// compareMap compares two maps, checking keys and values on both sides,
// and recording missing or differing entries.
func compareMap(left, right reflect.Value, isLeftCfg bool, path []pathStep, diffs *[]diffEntry) bool {
	if !left.IsValid() && !right.IsValid() || isNil(left) && isNil(right) {
		return true
	}

	if !right.IsValid() {
		return handleMissingValue(isLeftCfg, path, diffs)
	}

	if left.Type() != right.Type() {
		trackDiffValues(left.Interface(), right.Interface(), isLeftCfg, path, diffs)

		return false
	}

	equal := true
	seenKeys := make(map[string]struct{})

	if !compareMapInner(left, right, isLeftCfg, path, diffs, seenKeys) {
		equal = false
	}

	if !compareMapInner(right, left, !isLeftCfg, path, diffs, seenKeys) {
		equal = false
	}

	return equal
}

// compareMapInner helper for compareMap that iterates over one map,
// comparing and tracking differences for each key-value pair.
//
//nolint:gocritic
func compareMapInner(
	left, right reflect.Value,
	isLeftCfg bool,
	path []pathStep,
	diffs *[]diffEntry,
	seenKeys map[string]struct{},
) bool {
	equal := true

	for iter := left.MapRange(); iter.Next(); {
		kStr := fmt.Sprint(iter.Key().Interface())

		if _, seen := seenKeys[kStr]; seen {
			continue
		}

		seenKeys[kStr] = struct{}{}
		leftVal := iter.Value()
		rightVal := right.MapIndex(iter.Key())

		newPath := append(path, pathStep{key: kStr, kind: mapKind})

		if !rightVal.IsValid() {
			equal = handleMissingValue(isLeftCfg, newPath, diffs)

			continue
		}

		if !compareRecursive(leftVal, rightVal, isLeftCfg, newPath, diffs) {
			equal = false
		}
	}

	return equal
}

// compareSliceAndArray compares two slices or arrays, also handling special cases like time.Time
//
//nolint:forcetypeassert
func compareSliceAndArray(left, right reflect.Value, isLeftCfg bool, path []pathStep, diffs *[]diffEntry) bool {
	if !left.IsValid() && !right.IsValid() || isNil(left) && isNil(right) {
		return true
	}

	if !right.IsValid() || isNil(right) {
		return handleMissingValue(isLeftCfg, path, diffs)
	}

	if left.Type() != right.Type() {
		trackDiffValues(left.Interface(), right.Interface(), isLeftCfg, path, diffs)

		return false
	}

	if left.Type() == reflect.TypeFor[uuid.UUID]() {
		lUUID := left.Interface().(uuid.UUID)
		rUUID := right.Interface().(uuid.UUID)

		if lUUID != rUUID {
			trackDiffValues(lUUID, rUUID, isLeftCfg, path, diffs)

			return false
		}

		return true
	}

	return compareSliceInner(left, right, isLeftCfg, path, diffs)
}

// compareSliceInner helper for compareSliceAndArray that performs element-by-element
// comparison of two equally-sized slices, otherwise aligns complex elements with LCS.
//
//nolint:gocritic
func compareSliceInner(left, right reflect.Value, isLeftCfg bool, path []pathStep, diffs *[]diffEntry) bool {
	if left.Len() == right.Len() {
		equal := true

		for i := range left.Len() {
			sliceIndex := sliceElementIndex(left.Index(i), i)
			newPath := append(path, pathStep{key: sliceIndex, kind: sliceKind})

			if !compareRecursive(left.Index(i), right.Index(i), isLeftCfg, newPath, diffs) {
				equal = false
			}
		}

		return equal
	}

	if sliceIsPrimitive(left) {
		trackDiffValues(left.Interface(), right.Interface(), isLeftCfg, path, diffs)

		return false
	}

	return compareSliceWithLCS(left, right, isLeftCfg, path, diffs)
}

// compareSliceWithLCS compares two slices/arrays that differ in length and
// contain non-primitive elements. It aligns the elements with the
// Longest-Common-Subsequence (LCS) algorithm first and then recurses on the
// aligned pairs so that the final diff reports insertions / deletions instead
// of a noisy “everything shifted”.
//
// Algorithm:
//
//  1. Build the dynamic-programming table lcsTable using a shallow
//     equality predicate (`equalSliceShallow`).  Two elements are considered
//     equal if they expose the same logical key (struct field "Name", map key
//     "name", etc.).
//
//  2. Back-track through lcsTable to create a list of operations:
//
//     - both – element exists in both (diagonal move)
//
//     - onlyLeft – element only in left slice (up move)
//
//     - onlyRight – element only in right slice (left move)
//
//  3. Replay the operations in forward order. For each step:
//
//     - Build a human-readable JSON path
//
//     - Call `compareRecursive`
//
// Accumulate equality: the function returns true only if every recursive
// comparison returns true.
//
//nolint:cyclop,gocritic
func compareSliceWithLCS(left, right reflect.Value, isLeftCfg bool, path []pathStep, diffs *[]diffEntry) bool {
	leftSliceLen, rightSliceLen := left.Len(), right.Len()

	lcsTable := make([][]int, leftSliceLen+1)
	for i := range lcsTable {
		lcsTable[i] = make([]int, rightSliceLen+1)
	}

	for i := range leftSliceLen {
		for j := range rightSliceLen {
			switch {
			case equalSliceShallow(left.Index(i), right.Index(j)):
				lcsTable[i+1][j+1] = lcsTable[i][j] + 1

			case lcsTable[i][j+1] >= lcsTable[i+1][j]:
				lcsTable[i+1][j+1] = lcsTable[i][j+1]

			default:
				lcsTable[i+1][j+1] = lcsTable[i+1][j]
			}
		}
	}

	type operationKind uint8

	const (
		both operationKind = iota
		onlyLeft
		onlyRight
	)

	type lcsOperation struct {
		kind     operationKind
		leftIdx  int
		rightIdx int
	}

	var operations []lcsOperation

	for i, j := leftSliceLen, rightSliceLen; i > 0 || j > 0; {
		switch {
		case i > 0 && j > 0 && equalSliceShallow(left.Index(i-1), right.Index(j-1)):
			operations = append(operations, lcsOperation{both, i - 1, j - 1})
			i--
			j--
		case j > 0 && (i == 0 || lcsTable[i][j-1] >= lcsTable[i-1][j]):
			operations = append(operations, lcsOperation{onlyRight, -1, j - 1})
			j--
		default:
			operations = append(operations, lcsOperation{onlyLeft, i - 1, -1})
			i--
		}
	}

	equal := true
	currentIdx := 0

	for k := len(operations) - 1; k >= 0; k-- {
		var (
			element   reflect.Value
			operation = operations[k]
		)

		if leftSliceLen > rightSliceLen {
			element = left.Index(currentIdx)
		} else {
			element = right.Index(currentIdx)
		}

		sliceIndex := sliceElementIndex(element, currentIdx)
		newPath := append(path, pathStep{key: sliceIndex, kind: sliceKind})

		switch operation.kind {
		case onlyLeft:
			if !compareRecursive(left.Index(operation.leftIdx), reflect.Value{}, isLeftCfg, newPath, diffs) {
				equal = false
			}

		case onlyRight:
			if !compareRecursive(reflect.Value{}, right.Index(operation.rightIdx), isLeftCfg, newPath, diffs) {
				equal = false
			}

			currentIdx++

		case both:
			if !compareRecursive(left.Index(operation.leftIdx), right.Index(operation.rightIdx), isLeftCfg, newPath, diffs) {
				equal = false
			}

			currentIdx++
		}
	}

	return equal
}

// comparePrimitives compares two primitive values; handles special cases like int vs float64
// (from JSON) and otherwise falls back to reflect.DeepEqual.
//
//nolint:cyclop
func comparePrimitives(left, right reflect.Value, isLeftCfg bool, path []pathStep, diffs *[]diffEntry) bool {
	if !left.IsValid() && !right.IsValid() || isNil(left) && isNil(right) {
		return true
	}

	if !right.IsValid() {
		return handleMissingValue(isLeftCfg, path, diffs)
	}

	leftKind, rightKind := left.Kind(), right.Kind()

	if (isIntegerKind(leftKind) && rightKind == reflect.Float64) ||
		(isIntegerKind(rightKind) && leftKind == reflect.Float64) {
		var intVal, floatVal float64

		if isIntegerKind(leftKind) {
			intVal = left.Convert(reflect.TypeFor[float64]()).Float()
			floatVal = right.Float()
		} else {
			intVal = right.Convert(reflect.TypeFor[float64]()).Float()
			floatVal = left.Float()
		}

		if intVal == floatVal {
			return true
		}

		trackDiffValues(intVal, floatVal, isLeftCfg, path, diffs)

		return false
	}

	if !reflect.DeepEqual(left.Interface(), right.Interface()) {
		trackDiffValues(left.Interface(), right.Interface(), isLeftCfg, path, diffs)

		return false
	}

	return true
}
