package backup

import (
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

func TestCompareBackupField(t *testing.T) {
	uuid1 := uuid.MustParse("4cc47d9b-0240-4d02-a404-6732b9bbbc0a")
	uuid2 := uuid.MustParse("89f8eab1-5f80-46b9-b631-3c5a1f72ef31")
	datetime1 := time.Date(2025, 6, 25, 12, 0, 0, 0, time.UTC)
	datetime2 := time.Date(2022, 2, 14, 18, 3, 0, 0, time.UTC)

	testStruct := &Test{
		TestSlice: []SampleStruct{
			{
				Field1: datetime1,
				Field2: uuid1,
				Field3: "str1",
				Field4: 1.2,
				Field5: 3.4,
				Field6: 55,
				Field7: true,
			},
			{
				Field1: datetime2,
				Field2: uuid2,
				Field3: "str2",
				Field4: 5.6,
				Field5: 7.8,
				Field6: 77,
				Field7: false,
			},
		},
		TestInline: &Inline{
			InlineA: 5,
			InlineB: map[string]string{
				"k1": "v1",
				"k2": "v2",
			},
		},
		TestAnySlice: []any{
			"test",
			5,
		},
	}

	type testCase struct {
		name         string
		modifyCfg    func(c *Test)
		ok           bool
		expDiffPaths []string
	}

	testCases := []testCase{
		{
			name:         "Identical",
			modifyCfg:    func(c *Test) {},
			ok:           true,
			expDiffPaths: nil,
		},
		{
			name: "Changed",
			modifyCfg: func(c *Test) {
				c.TestSlice[0].Field3 = "DIFF"
				c.TestInline.InlineA = 99
				c.TestInline.InlineB["k"] = "new V"
				c.TestAnySlice[1] = "another"
			},
			ok: false,
			expDiffPaths: []string{
				"test_slice[0].field_3",
				"inline_a",
				"inline_b[k]",
				"test_any_slice[1]",
			},
		},
		{
			name: "Remove slice element",
			modifyCfg: func(c *Test) {
				c.TestSlice = c.TestSlice[:1]
			},
			expDiffPaths: []string{
				"test_slice[1]",
			},
		},
		{
			name: "Added slice element",
			modifyCfg: func(c *Test) {
				c.TestSlice = append(c.TestSlice, SampleStruct{
					Field1: time.Date(2005, 8, 11, 9, 54, 0, 0, time.UTC),
					Field2: uuid.MustParse("8cc47d9b-0240-6d02-a454-6732b9bbbc0a"),
					Field3: "new_string",
					Field4: 1.2,
					Field5: 3.4,
					Field6: 55,
					Field7: true,
				})
			},
			expDiffPaths: []string{
				"test_slice[2]",
			},
		},
		{
			name: "Removed map element",
			modifyCfg: func(c *Test) {
				delete(c.TestInline.InlineB, "k2")
			},
			expDiffPaths: []string{
				"inline_b[k2]",
			},
		},
		{
			name: "Added map element",
			modifyCfg: func(c *Test) {
				c.TestInline.InlineB["k3"] = "v3"
			},
			expDiffPaths: []string{
				"inline_b[k3]",
			},
		},
	}

	testFunc := func(t *testing.T, tc testCase) {
		t.Helper()

		testStructCopy := testStruct.deepCopy()
		tc.modifyCfg(testStructCopy)

		ok, diffs := compareBackupField(&testStructCopy, testStruct)
		require.Equal(t, tc.ok, ok)

		require.Len(t, diffs, len(tc.expDiffPaths))

		gotPaths := make(map[string]bool)
		for _, d := range diffs {
			gotPaths[d.path] = true
		}

		for _, p := range tc.expDiffPaths {
			require.True(t, gotPaths[p])
		}
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) { testFunc(t, tc) })
	}
}
