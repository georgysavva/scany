package pgxquery_test

import (
	"fmt"
	"github.com/georgysavva/pgxquery"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"testing"
)

func TestScanOneStruct(t *testing.T) {
	t.Parallel()
	cases := []ScanCase{
		{
			name: "basic",
			rows: &fakeRows{
				columns: []string{"foo", "bar"},
				data: [][]interface{}{
					{4, "b"},
				},
			},
			expected: struct {
				Foo int
				Bar string
			}{
				Foo: 4,
				Bar: "b",
			},
		},
		{
			name: "fields automatically mapped to snake case",
			rows: &fakeRows{
				columns: []string{"foo_bar", "bar_foo"},
				data: [][]interface{}{
					{4, "b"},
				},
			},
			expected: struct {
				FooBar int
				BarFoo string
			}{
				FooBar: 4,
				BarFoo: "b",
			},
		},
		{
			name: "fields automatically mapped to snake case",
			rows: &fakeRows{
				columns: []string{"foo_column", "bar_column"},
				data: [][]interface{}{
					{4, "b"},
				},
			},
			expected: struct {
				Foo int    `db:"foo_column"`
				Bar string `db:"bar_column"`
			}{
				Foo: 4,
				Bar: "b",
			},
		},
		{
			name: "field not found",
			rows: &fakeRows{
				columns: []string{"foo", "bar"},
				data: [][]interface{}{
					{4, "b"},
				},
			},
			expected: struct {
				Bar string
			}{},
			errString: "column: 'foo': no corresponding field found or it's unexported in struct { Bar string }",
		},
		{
			name: "rows contain duplicated column",
			rows: &fakeRows{
				columns: []string{"foo", "foo"},
				data: [][]interface{}{
					{4, "b"},
				},
			},
			expected: struct {
				Foo string
			}{},
			errString: "row contains duplicated column 'foo'",
		},
		{
			name: "field is ignored",
			rows: &fakeRows{
				columns: []string{"foo", "bar"},
				data: [][]interface{}{
					{4, "b"},
				},
			},
			expected: struct {
				Foo int `db:"-"`
				Bar string
			}{},
			errString: "column: 'foo': no corresponding field found or it's unexported in " +
				"struct { Foo int \"db:\\\"-\\\"\"; Bar string }",
		},
		{
			name: "field is unexported",
			rows: &fakeRows{
				columns: []string{"foo", "bar"},
				data: [][]interface{}{
					{4, "b"},
				},
			},
			expected: struct {
				foo int
				Bar string
			}{},
			errString: "column: 'foo': no corresponding field found or it's unexported in struct { foo int; Bar string }",
		},
		{
			name: "field is unexported via tag",
			rows: &fakeRows{
				columns: []string{"foo_column", "bar"},
				data: [][]interface{}{
					{4, "b"},
				},
			},
			expected: struct {
				foo int `db:"foo_column"`
				Bar string
			}{},
			errString: "column: 'foo_column': no corresponding field found or it's unexported in " +
				"struct { foo int \"db:\\\"foo_column\\\"\"; Bar string }",
		},
		{
			name: "duplicated tag",
			rows: &fakeRows{
				columns: []string{"foo_column", "bar"},
				data: [][]interface{}{
					{4, "b"},
				},
			},
			expected: struct {
				Foo int    `db:"foo_column"`
				Bar string `db:"foo_column"`
			}{},
			errString: "Column must have exactly one field pointing to it; " +
				"found 2 fields with indexes [0] and [1] pointing to 'foo_column' in " +
				"struct { Foo int \"db:\\\"foo_column\\\"\"; Bar string \"db:\\\"foo_column\\\"\" }",
		},
	}
	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			tc.exactlyOneRow = true
			tc.test(t)
		})
	}
}

func TestScanOneMap(t *testing.T) {
	t.Parallel()
	cases := []ScanCase{
		{
			name: "basic",
			rows: &fakeRows{
				columns: []string{"foo", "bar"},
				data: [][]interface{}{
					{4, "b"},
				},
			},
			expected: map[string]interface{}{
				"foo": 4,
				"bar": "b",
			},
		},
		{
			name: "non interface{} element type",
			rows: &fakeRows{
				columns: []string{"foo", "bar"},
				data: [][]interface{}{
					{"f", "b"},
				},
			},
			expected: map[string]string{
				"foo": "f",
				"bar": "b",
			},
		},
		{
			name: "value converted to the right type",
			rows: &fakeRows{
				columns: []string{"foo", "bar"},
				data: [][]interface{}{
					{int8(4), int8(5)},
				},
			},
			expected: map[string]int{
				"foo": 4,
				"bar": 5,
			},
		},
		{
			name: "non string key is not allowed",
			rows: &fakeRows{
				columns: []string{"foo", "bar"},
				data: [][]interface{}{
					{4, "b"},
				},
			},
			expected:  map[int]interface{}{},
			errString: "invalid type map[int]interface {}: map must have string key, got: int",
		},
		{
			name: "rows contain duplicated column",
			rows: &fakeRows{
				columns: []string{"foo", "foo"},
				data: [][]interface{}{
					{4, "b"},
				},
			},
			expected:  map[string]interface{}{},
			errString: "row contains duplicated column 'foo'",
		},
		{
			name: "invalid element type",
			rows: &fakeRows{
				columns: []string{"foo", "bar"},
				data: [][]interface{}{
					{4, "b"},
				},
			},
			expected:  map[string]int{},
			errString: "Column 'bar' value of type string can'be set into map[string]int",
		},
	}
	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			tc.exactlyOneRow = true
			tc.test(t)
		})
	}
}

func TestScanOnePrimitiveType(t *testing.T) {
	t.Parallel()
	cases := []ScanCase{
		{
			name: "basic",
			rows: &fakeRows{
				columns: []string{"bar"},
				data: [][]interface{}{
					{"b"},
				},
			},
			expected: "b",
		},
		{
			name: "string by ptr",
			rows: &fakeRows{
				columns: []string{"bar"},
				data: [][]interface{}{
					{"b"},
				},
			},
			expected: "b",
		},
		{
			name: "slice as single column",
			rows: &fakeRows{
				columns: []string{"bar"},
				data: [][]interface{}{
					{[]string{"a", "b", "c"}},
				},
			},
			expected: []string{"a", "b", "c"},
		},
		{
			name: "0 columns",
			rows: &fakeRows{
				data: [][]interface{}{
					{"b"},
				},
				columns: []string{},
			},
			expected:  "",
			errString: "to fill into a primitive type, columns number must be exactly 1, got: 0",
		},
		{
			name: "more than 1 column",
			rows: &fakeRows{
				columns: []string{"foo", "bar"},
				data: [][]interface{}{
					{"f", "b"},
				},
			},
			expected:  "",
			errString: "to fill into a primitive type, columns number must be exactly 1, got: 2",
		},
	}
	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			tc.exactlyOneRow = true
			tc.test(t)
		})
	}
}

func TestScanAll(t *testing.T) {
	t.Parallel()
	cases := []ScanCase{
		{
			name: "slice of structs",
			rows: &fakeRows{
				columns: []string{"foo", "bar"},
				data: [][]interface{}{
					{4, "b"},
					{44, "bb"},
					{444, "bbb"},
				},
			},
			expected: []struct {
				Foo int
				Bar string
			}{
				{Foo: 4, Bar: "b"},
				{Foo: 44, Bar: "bb"},
				{Foo: 444, Bar: "bbb"},
			},
		},
		{
			name: "slice of structs by ptr",
			rows: &fakeRows{
				columns: []string{"foo", "bar"},
				data: [][]interface{}{
					{4, "b"},
					{44, "bb"},
					{444, "bbb"},
				},
			},
			expected: []*struct {
				Foo int
				Bar string
			}{
				{Foo: 4, Bar: "b"},
				{Foo: 44, Bar: "bb"},
				{Foo: 444, Bar: "bbb"},
			},
		},
		{
			name: "slice of maps",
			rows: &fakeRows{
				columns: []string{"foo", "bar"},
				data: [][]interface{}{
					{4, "b"},
					{44, "bb"},
					{444, "bbb"},
				},
			},
			expected: []map[string]interface{}{
				{"foo": 4, "bar": "b"},
				{"foo": 44, "bar": "bb"},
				{"foo": 444, "bar": "bbb"},
			},
		},
		{
			name: "slice of maps by ptr",
			rows: &fakeRows{
				columns: []string{"foo", "bar"},
				data: [][]interface{}{
					{4, "b"},
					{44, "bb"},
					{444, "bbb"},
				},
			},
			expected: []*map[string]interface{}{
				{"foo": 4, "bar": "b"},
				{"foo": 44, "bar": "bb"},
				{"foo": 444, "bar": "bbb"},
			},
		},
		{
			name: "slice of strings",
			rows: &fakeRows{
				columns: []string{"bar"},
				data: [][]interface{}{
					{"b"},
					{"bb"},
					{"bbb"},
				},
			},
			expected: []string{"b", "bb", "bbb"},
		},
		{
			name: "slice of strings by ptr",
			rows: &fakeRows{
				columns: []string{"bar"},
				data: [][]interface{}{
					{makeStrPtr("b")},
					{nil},
					{makeStrPtr("bbb")},
				},
			},
			expected: []*string{makeStrPtr("b"), nil, makeStrPtr("bbb")},
		},
		{
			name: "slice of slices",
			rows: &fakeRows{
				columns: []string{"bar"},
				data: [][]interface{}{
					{[]string{"a", "b"}},
					{[]string{"aa", "bb"}},
					{[]string{"aaa", "bbb"}},
				},
			},
			expected: [][]string{
				{"a", "b"},
				{"aa", "bb"},
				{"aaa", "bbb"},
			},
		},
	}
	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			tc.exactlyOneRow = false
			tc.test(t)
		})
	}
}

func TestScanAllResetsDstSlice(t *testing.T) {
	t.Parallel()
	fr := &fakeRows{
		columns: []string{"bar"},
		data: [][]interface{}{
			{"b"},
			{"bb"},
			{"bbb"},
		},
	}
	expected := []string{"b", "bb", "bbb"}
	var got []string
	var err error
	for i := 0; i < 3; i++ {
		t.Run(fmt.Sprintf("iteraction %d", i), func(t *testing.T) {
			err = pgxquery.ScanAll(&got, fr)
			require.NoError(t, err)
			assert.Equal(t, expected, got)
			fr.Reset()
		})
	}
}

func TestScanInvalidDestinations(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name          string
		exactlyOneRow bool
		dst           interface{}
		errString     string
	}{
		{
			name: "scan one: non pointer",
			dst: struct {
				Foo string
			}{},
			exactlyOneRow: true,
			errString:     "destinationMeta must be a pointer, got: struct { Foo string }",
		},
		{
			name: "scan all: non pointer",
			dst: struct {
				Foo string
			}{},
			exactlyOneRow: false,
			errString:     "destinationMeta must be a pointer, got: struct { Foo string }",
		},
		{
			name:          "scan one: map",
			dst:           map[string]interface{}{},
			exactlyOneRow: true,
			errString:     "destinationMeta must be a pointer, got: map[string]interface {}",
		},
		{
			name:          "scan all: map",
			dst:           map[string]interface{}{},
			exactlyOneRow: false,
			errString:     "destinationMeta must be a pointer, got: map[string]interface {}",
		},
		{
			name:          "scan one: slice",
			dst:           []struct{ Foo string }{},
			exactlyOneRow: true,
			errString:     "destinationMeta must be a pointer, got: []struct { Foo string }",
		},
		{
			name:          "scan all: slice",
			dst:           []struct{ Foo string }{},
			exactlyOneRow: false,
			errString:     "destinationMeta must be a pointer, got: []struct { Foo string }",
		},
		{
			name:          "scan one: nil",
			dst:           nil,
			exactlyOneRow: true,
			errString:     "destinationMeta must be a non nil pointer",
		},
		{
			name:          "scan all: nil",
			dst:           nil,
			exactlyOneRow: false,
			errString:     "destinationMeta must be a non nil pointer",
		},
		{
			name:          "scan one: (*int)(nil)",
			dst:           (*int)(nil),
			exactlyOneRow: true,
			errString:     "destinationMeta must be a non nil pointer",
		},
		{
			name:          "scan all: (*int)(nil)",
			dst:           (*int)(nil),
			exactlyOneRow: false,
			errString:     "destinationMeta must be a non nil pointer",
		},
		{
			name: "scan all: not pointer to slice",
			dst: &struct {
				A string
			}{},
			exactlyOneRow: false,
			errString:     "destinationMeta must be a pointer to a slice, got: *struct { A string }",
		},
	}
	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			fr := &fakeRows{}
			var err error
			if tc.exactlyOneRow {
				err = pgxquery.ScanOne(tc.dst, fr)
			} else {
				err = pgxquery.ScanAll(tc.dst, fr)
			}
			assert.EqualError(t, err, tc.errString)
		})
	}
}

func TestScanOneRowsMismatch(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name      string
		rows      *fakeRows
		errString string
	}{
		{
			name: "0 rows",
			rows: &fakeRows{
				columns: []string{"foo"},
				data:    [][]interface{}{},
			},
			errString: "no row was found",
		},
		{
			name: "more than 1 row",
			rows: &fakeRows{
				columns: []string{"foo"},
				data: [][]interface{}{
					{"b"},
					{"bb"},
					{"bbb"},
				},
			},
			errString: "expected 1 row, got: 3",
		},
	}
	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			var dst string
			err := pgxquery.ScanOne(&dst, tc.rows)
			assert.EqualError(t, err, tc.errString)
		})
	}
}
