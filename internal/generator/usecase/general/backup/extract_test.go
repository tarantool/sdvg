package backup

import (
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

type Inline struct {
	InlineA      int               `backup:"true"         json:"inline_a"`
	InlineB      map[string]string `backup:"true"         json:"inline_b"`
	SkipInInline map[string]int    `json:"skip_in_inline"`
}

type SampleStruct struct {
	Field1             time.Time `backup:"true"                json:"field_1"`
	Field2             uuid.UUID `backup:"true"                json:"field_2"`
	Field3             string    `backup:"true"                json:"field_3"`
	Field4             float32   `backup:"true"                json:"field_4"`
	Field5             float64   `backup:"true"                json:"field_5"`
	Field6             uint64    `backup:"true"                json:"field_6"`
	Field7             bool      `backup:"true"                json:"field_7"`
	SkipInSampleStruct int       `json:"skip_in_sample_struct"`
}

type Test struct {
	TestSlice    []SampleStruct `backup:"true"       json:"test_slice"`
	TestInline   *Inline        `backup:"true"       json:",inline"`
	TestAnySlice []any          `backup:"true"       json:"test_any_slice"`
	TestField1   uint64         `backup:"true"       json:"test_field_1"`
	TestField2   uint64         `backup:"true"       json:"test_field_2"`
	SkipInTest   bool           `json:"skip_in_test"`
}

func (t *Test) deepCopy() *Test {
	if t == nil {
		return nil
	}

	deepCopy := &Test{
		TestField1: t.TestField1,
		TestField2: t.TestField2,
		SkipInTest: t.SkipInTest,
	}

	if t.TestSlice != nil {
		deepCopy.TestSlice = make([]SampleStruct, len(t.TestSlice))
		copy(deepCopy.TestSlice, t.TestSlice)
	}

	if t.TestInline != nil {
		deepCopy.TestInline = &Inline{
			InlineA: t.TestInline.InlineA,
		}

		if t.TestInline.InlineB != nil {
			deepCopy.TestInline.InlineB = make(map[string]string, len(t.TestInline.InlineB))
			for k, v := range t.TestInline.InlineB {
				deepCopy.TestInline.InlineB[k] = v
			}
		}

		if t.TestInline.SkipInInline != nil {
			deepCopy.TestInline.SkipInInline = make(map[string]int, len(t.TestInline.SkipInInline))
			for k, v := range t.TestInline.SkipInInline {
				deepCopy.TestInline.SkipInInline[k] = v
			}
		}
	}

	if t.TestAnySlice != nil {
		deepCopy.TestAnySlice = make([]any, len(t.TestAnySlice))
		copy(deepCopy.TestAnySlice, t.TestAnySlice)
	}

	return deepCopy
}

func TestExtractBackupFields(t *testing.T) {
	t.Helper()

	uuid1 := uuid.MustParse("4cc47d9b-0240-4d02-a404-6732b9bbbc0a")
	uuid2 := uuid.MustParse("89f8eab1-5f80-46b9-b631-3c5a1f72ef31")
	datetime1 := time.Date(2025, 6, 25, 12, 0, 0, 0, time.UTC)
	datetime2 := time.Date(2022, 2, 14, 18, 3, 0, 0, time.UTC)

	var uint1 uint64 = 1<<53 - 1

	var uint2 uint64 = 1 << 53

	testStruct := &Test{
		TestSlice: []SampleStruct{
			{
				Field1:             datetime1,
				Field2:             uuid1,
				Field3:             "test_string1",
				Field4:             1.26,
				Field5:             1.37,
				Field6:             68,
				Field7:             true,
				SkipInSampleStruct: 1,
			},
			{
				Field1:             datetime2,
				Field2:             uuid2,
				Field3:             "test_string2",
				Field4:             5.37,
				Field5:             146.1,
				Field6:             128,
				Field7:             false,
				SkipInSampleStruct: 2,
			},
		},
		TestInline: &Inline{
			InlineA: 5,
			InlineB: map[string]string{
				"key1": "value1",
				"key2": "value2",
			},
			SkipInInline: map[string]int{
				"key1": 1,
				"key2": 2,
			},
		},
		TestAnySlice: []any{
			1,
			2.23,
			"test",
			float32(5.3),
		},
		TestField1: uint1,
		TestField2: uint2,
		SkipInTest: false,
	}

	expected := map[string]any{
		"test_slice": []any{
			map[string]any{
				"field_1": datetime1.Format(time.RFC3339),
				"field_2": uuid1.String(),
				"field_3": "test_string1",
				"field_4": float32(1.26),
				"field_5": 1.37,
				"field_6": uint64(68),
				"field_7": true,
			},
			map[string]any{
				"field_1": datetime2.Format(time.RFC3339),
				"field_2": uuid2.String(),
				"field_3": "test_string2",
				"field_4": float32(5.37),
				"field_5": 146.1,
				"field_6": uint64(128),
				"field_7": false,
			},
		},
		"inline_a": 5,
		"inline_b": map[string]any{
			"key1": "value1",
			"key2": "value2",
		},
		"test_any_slice": []any{
			1,
			2.23,
			"test",
			float32(5.3),
		},
		"test_field_1": uint1,
		"test_field_2": uint2,
	}

	result := extractBackupFields(testStruct)

	require.Equal(t, expected, result)
}
