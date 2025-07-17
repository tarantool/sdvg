package backup

import (
	"fmt"
	"reflect"
	"strconv"
)

// isNil returns true if the [reflect.Value] is nil, safely checking only kinds that can be nil.
func isNil(v reflect.Value) bool {
	return canBeNil(v.Kind()) && v.IsNil()
}

// canBeNil determines whether a given reflect.Kind type supports nil values
// (e.g., pointers, interfaces, slices).
func canBeNil(k reflect.Kind) bool {
	switch k {
	case reflect.Pointer, reflect.Interface, reflect.Map, reflect.Slice, reflect.Chan, reflect.Func:
		return true
	default:
		return false
	}
}

// unwrapPointersOrInterfaces recursively unwraps pointers and interfaces to reach
// the underlying concrete value, stopping if it finds nil.
func unwrapPointersOrInterfaces(v reflect.Value) reflect.Value {
	for v.IsValid() && (v.Kind() == reflect.Pointer || v.Kind() == reflect.Interface) {
		if v.IsNil() {
			return v
		}

		v = v.Elem()
	}

	return v
}

// sliceElementKey extracts a key from a slice element to identify it during
// alignment: looks for struct field "Name" or map key "name"/"Name".
func sliceElementKey(v reflect.Value) (string, bool) {
	v = unwrapPointersOrInterfaces(v)

	if isNil(v) || !v.IsValid() {
		return "", false
	}

	switch v.Kind() {
	case reflect.Struct:
		if f, ok := v.Type().FieldByName("Name"); ok {
			return fmt.Sprint(v.FieldByIndex(f.Index).Interface()), true
		}
	case reflect.Map:
		iter := v.MapRange()
		for iter.Next() {
			k := fmt.Sprint(iter.Key().Interface())
			if k == "name" || k == "Name" {
				return fmt.Sprint(iter.Value().Interface()), true
			}
		}
	}

	return "", false
}

// equalSliceShallow performs a shallow equality check between two slice
// elements: compares their keys if available, otherwise falls back to deep comparison.
func equalSliceShallow(left reflect.Value, right reflect.Value) bool {
	if leftKey, leftOk := sliceElementKey(left); leftOk {
		if rightKey, rightOk := sliceElementKey(right); rightOk {
			return leftKey == rightKey
		}
	}

	return compareRecursive(left, right, true, nil, nil)
}

// sliceElementIndex generates a string index for a slice element using its key
// if available, or falls back to the numeric index.
func sliceElementIndex(element reflect.Value, intIndex int) string {
	var index string

	key, ok := sliceElementKey(element)
	if ok {
		index = key
	} else {
		index = strconv.Itoa(intIndex)
	}

	return index
}
