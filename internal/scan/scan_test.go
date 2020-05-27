package scan_test

import (
	"github.com/georgysavva/pgxquery/internal/scan"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"reflect"
	"testing"
)

type fakeRows struct {
	data    []interface{}
	columns []string
}

func (fr *fakeRows) Scan(dest ...interface{}) error {
	for i, data := range fr.data {
		dst := dest[i]
		dstV := reflect.ValueOf(dst).Elem()
		dstV.Set(reflect.ValueOf(data))
	}
	return nil
}
func (fr *fakeRows) Values() ([]interface{}, error) { return fr.data, nil }

func (fr *fakeRows) Columns() []string { return fr.columns }

func TestDestination_Fill(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name      string
		rows      *fakeRows
		expected  interface{}
		errString string
	}{
		{
			name: "struct smoke",
			rows: &fakeRows{
				data:    []interface{}{4, "bb"},
				columns: []string{"foo", "bar"},
			},
			expected: struct {
				Foo int
				Bar string
			}{
				Foo: 4,
				Bar: "bb",
			},
		},
		{
			name: "map smoke",
			rows: &fakeRows{
				data:    []interface{}{4, "bb"},
				columns: []string{"foo", "bar"},
			},
			expected: map[string]interface{}{
				"foo": 4,
				"bar": "bb",
			},
		},
		{
			name: "primitive type smoke",
			rows: &fakeRows{
				data:    []interface{}{"bb"},
				columns: []string{"bar"},
			},
			expected: "bb",
		},
		{
			name: "slice of structs",
			rows: &fakeRows{
				data:    []interface{}{4, "bb"},
				columns: []string{"foo", "bar"},
			},
			expected: []struct {
				Foo int
				Bar string
			}{
				{
					Foo: 4,
					Bar: "bb",
				},
			},
		},
		{
			name: "slice of structs by ptr",
			rows: &fakeRows{
				data:    []interface{}{4, "bb"},
				columns: []string{"foo", "bar"},
			},
			expected: []*struct {
				Foo int
				Bar string
			}{
				{
					Foo: 4,
					Bar: "bb",
				},
			},
		},
		{
			name: "slice of maps",
			rows: &fakeRows{
				data:    []interface{}{4, "bb"},
				columns: []string{"foo", "bar"},
			},
			expected: []map[string]interface{}{
				{
					"foo": 4,
					"bar": "bb",
				},
			},
		},
		{
			name: "slice of strings",
			rows: &fakeRows{
				data:    []interface{}{"bb"},
				columns: []string{"bar"},
			},
			expected: []string{"bb"},
		},
		{
			name: "struct field not found",
			rows: &fakeRows{
				data:    []interface{}{4, "bb"},
				columns: []string{"foo", "bar"},
			},
			expected: struct {
				Bar string
			}{},
			errString: "column: 'foo': no corresponding field found or it's unexported in struct { Bar string }",
		},
		{
			name: "struct duplicated column",
			rows: &fakeRows{
				data:    []interface{}{4, "bb"},
				columns: []string{"foo", "foo"},
			},
			expected: struct {
				Foo string
			}{},
			errString: "row contains duplicated column 'foo'",
		},
		{
			name: "struct field is unexported",
			rows: &fakeRows{
				data:    []interface{}{4, "bb"},
				columns: []string{"foo", "bar"},
			},
			expected: struct {
				foo int
				Bar string
			}{},
			errString: "column: 'foo': no corresponding field found or it's unexported in struct { foo int; Bar string }",
		},
		{
			name: "map string element type",
			rows: &fakeRows{
				data:    []interface{}{"ff", "bb"},
				columns: []string{"foo", "bar"},
			},
			expected: map[string]string{
				"foo": "ff",
				"bar": "bb",
			},
		},
		{
			name: "map non string key",
			rows: &fakeRows{
				data:    []interface{}{4, "bb"},
				columns: []string{"foo", "bar"},
			},
			expected:  map[int]interface{}{},
			errString: "invalid element type map[int]interface {}: map must have string key, got: int",
		},
		{
			name: "map non string key",
			rows: &fakeRows{
				data:    []interface{}{4, "bb"},
				columns: []string{"foo", "foo"},
			},
			expected:  map[string]interface{}{},
			errString: "row contains duplicated column 'foo'",
		},
		{
			name: "map invalid element type",
			rows: &fakeRows{
				data:    []interface{}{4, "bb"},
				columns: []string{"foo", "bar"},
			},
			expected:  map[string]int{},
			errString: "Column 'bar' value of type string can'be set into map[string]int",
		},
		{
			name: "primitive type 0 columns",
			rows: &fakeRows{
				data:    []interface{}{},
				columns: []string{},
			},
			expected:  "",
			errString: "to scan into a primitive type, columns number must be exactly 1, got: 0",
		},
		{
			name: "primitive type more than 1 column",
			rows: &fakeRows{
				data:    []interface{}{"ff", "bb"},
				columns: []string{"foo", "bar"},
			},
			expected:  "",
			errString: "to scan into a primitive type, columns number must be exactly 1, got: 2",
		},
	}
	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			dstType := reflect.TypeOf(tc.expected)
			dstVal := reflect.New(dstType).Elem()
			scanDst := scan.NewDestination(dstVal)

			err := scanDst.Fill(tc.rows)
			if tc.errString == "" {
				require.NoError(t, err)
				got := dstVal.Interface()
				assert.Equal(t, tc.expected, got)
			} else {
				assert.EqualError(t, err, tc.errString)
			}
		})
	}
}

func TestGetColumnToFieldIndexMap(t *testing.T) {
	t.Parallel()
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
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			structType := reflect.TypeOf(tc.structObj)
			got, err := scan.GetColumnToFieldIndexMap(structType)
			if tc.errString == "" {
				require.NoError(t, err)
				assert.Equal(t, tc.expected, got)
			} else {
				assert.EqualError(t, err, tc.errString)
			}
		})
	}
}
