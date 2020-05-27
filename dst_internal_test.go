package pgxquery

import (
	"reflect"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGetColumnToFieldIndexMap(t *testing.T) {
	t.Parallel()
	type BaseNested struct {
		H string
	}
	cases := []struct {
		name      string
		structObj interface{}
		expected  map[string][]int
		errString string
	}{
		{
			name: "smoke",
			structObj: struct {
				A string
				B int
			}{},
			expected: map[string][]int{
				"a": {0},
				"b": {1},
			},
		},
		{
			name: "to snake case",
			structObj: struct {
				FooBar  string
				FooBar2 string
				B       int
			}{},
			expected: map[string][]int{
				"foo_bar":  {0},
				"foo_bar2": {1},
				"b":        {2},
			},
		},
		{
			name: "via tag",
			structObj: struct {
				A string `db:"a_column"`
				B int    `db:"b_column"`
			}{},
			expected: map[string][]int{
				"a_column": {0},
				"b_column": {1},
			},
		},
		{
			name: "unexported field",
			structObj: struct {
				a string `db:"a_column"`
				b int
				C int
			}{},
			expected: map[string][]int{
				"c": {2},
			},
		},
		{
			name: "non distinct column",
			structObj: struct {
				A string `db:"a_column"`
				B string `db:"a_column"`
			}{},
			expected: nil,
			errString: "Column must have exactly one field pointing to it; " +
				"found 2 fields with indexes [0] and [1] pointing to 'a_column' in " +
				"struct { A string \"db:\\\"a_column\\\"\"; B string \"db:\\\"a_column\\\"\" }",
		},
		//{
		//	name: "smoke",
		//	structObj: struct {
		//		BaseNested
		//		N BaseNested
		//		L struct {
		//			U string
		//		}
		//		A string
		//		B int
		//	}{},
		//	expected: map[string][]int{
		//		"a": {0},
		//		"b": {1},
		//	},
		//},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			structType := reflect.TypeOf(tc.structObj)
			got, err := getColumnToFieldIndexMap(structType)
			if tc.errString == "" {
				require.NoError(t, err)
				assert.Equal(t, tc.expected, got)
			} else {
				assert.EqualError(t, err, tc.errString)
			}
		})
	}
}
