package pgxquery_test

import (
	"reflect"
	"testing"

	"github.com/georgysavva/pgxquery"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestScanAll(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name     string
		rows     *fakeRows
		expected interface{}
	}{
		{
			name: "slice of structs",
			rows: &fakeRows{
				columns: []string{"foo", "bar"},
				data: [][]interface{}{
					{"foo val", "bar val"},
					{"foo val 2", "bar val 2"},
					{"foo val 3", "bar val 3"},
				},
			},
			expected: []struct {
				Foo string
				Bar string
			}{
				{Foo: "foo val", Bar: "bar val"},
				{Foo: "foo val 2", Bar: "bar val 2"},
				{Foo: "foo val 3", Bar: "bar val 3"},
			},
		},
		{
			name: "slice of structs by ptr",
			rows: &fakeRows{
				columns: []string{"foo", "bar"},
				data: [][]interface{}{
					{"foo val", "bar val"},
					{"foo val 2", "bar val 2"},
					{"foo val 3", "bar val 3"},
				},
			},
			expected: []*struct {
				Foo string
				Bar string
			}{
				{Foo: "foo val", Bar: "bar val"},
				{Foo: "foo val 2", Bar: "bar val 2"},
				{Foo: "foo val 3", Bar: "bar val 3"},
			},
		},
		{
			name: "slice of maps",
			rows: &fakeRows{
				columns: []string{"foo", "bar"},
				data: [][]interface{}{
					{"foo val", "bar val"},
					{"foo val 2", "bar val 2"},
					{"foo val 3", "bar val 3"},
				},
			},
			expected: []map[string]interface{}{
				{"foo": "foo val", "bar": "bar val"},
				{"foo": "foo val 2", "bar": "bar val 2"},
				{"foo": "foo val 3", "bar": "bar val 3"},
			},
		},
		{
			name: "slice of maps by ptr",
			rows: &fakeRows{
				columns: []string{"foo", "bar"},
				data: [][]interface{}{
					{"foo val", "bar val"},
					{"foo val 2", "bar val 2"},
					{"foo val 3", "bar val 3"},
				},
			},
			expected: []*map[string]interface{}{
				{"foo": "foo val", "bar": "bar val"},
				{"foo": "foo val 2", "bar": "bar val 2"},
				{"foo": "foo val 3", "bar": "bar val 3"},
			},
		},
		{
			name: "slice of strings",
			rows: &fakeRows{
				columns: []string{"bar"},
				data: [][]interface{}{
					{"bar val"},
					{"bar val 2"},
					{"bar val 3"},
				},
			},
			expected: []string{"bar val", "bar val 2", "bar val 3"},
		},
		{
			name: "slice of strings by ptr",
			rows: &fakeRows{
				columns: []string{"bar"},
				data: [][]interface{}{
					{makeStrPtr("bar val")},
					{nil},
					{makeStrPtr("bar val 3")},
				},
			},
			expected: []*string{makeStrPtr("bar val"), nil, makeStrPtr("bar val 3")},
		},
		{
			name: "slice of slices",
			rows: &fakeRows{
				columns: []string{"bar"},
				data: [][]interface{}{
					{[]string{"bar val", "bar val 2"}},
					{[]string{"bar val 3", "bar val 4"}},
					{[]string{"bar val 5", "bar val 6"}},
				},
			},
			expected: [][]string{
				{"bar val", "bar val 2"},
				{"bar val 3", "bar val 4"},
				{"bar val 5", "bar val 6"},
			},
		},
	}
	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			dstVal := newDstValue(tc.expected)
			err := pgxquery.ScanAll(dstVal.Addr().Interface(), tc.rows)
			require.NoError(t, err)
			assertDstValueEqual(t, tc.expected, dstVal)
		})
	}
}

func TestDoScan_StructDestination(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name     string
		rows     *fakeRows
		expected interface{}
	}{
		{
			name: "fields without tag are filled from column via snake case mapping",
			rows: &fakeRows{
				columns: []string{"foo_column", "bar"},
				data: [][]interface{}{
					{"foo val", "bar val"},
				},
			},
			expected: struct {
				FooColumn string
				Bar       string
			}{
				FooColumn: "foo val",
				Bar:       "bar val",
			},
		},
		{
			name: "fields with tag are filled from columns via tag",
			rows: &fakeRows{
				columns: []string{"foo_column"},
				data: [][]interface{}{
					{"foo val"},
				},
			},
			expected: struct {
				Foo string `db:"foo_column"`
			}{
				Foo: "foo val",
			},
		},
		{
			name: "field with ignore tag isn't filled",
			rows: &fakeRows{
				columns: []string{"foo"},
				data: [][]interface{}{
					{"foo val"},
				},
			},
			expected: struct {
				Foo string `db:"-"`
				Bar string `db:"foo"`
			}{
				Foo: "",
				Bar: "foo val",
			},
		},
	}
	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			dstVal := newDstValue(tc.expected)
			err := doScan(dstVal, tc.rows)
			require.NoError(t, err)
			assertDstValueEqual(t, tc.expected, dstVal)
		})
	}
}

func TestDoScan_MapDestination(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name     string
		rows     *fakeRows
		expected interface{}
	}{
		{
			name: "basic map[string]interface{}",
			rows: &fakeRows{
				columns: []string{"foo", "bar"},
				data: [][]interface{}{
					{"foo val", "bar val"},
				},
			},
			expected: map[string]interface{}{
				"foo": "foo val",
				"bar": "bar val",
			},
		},
		{
			name: "non interface{} element types are allowed",
			rows: &fakeRows{
				columns: []string{"foo", "bar"},
				data: [][]interface{}{
					{"foo val", "bar val"},
				},
			},
			expected: map[string]string{
				"foo": "foo val",
				"bar": "bar val",
			},
		},
		{
			name: "values with different type are converted to the map element type",
			rows: &fakeRows{
				columns: []string{"foo", "bar"},
				data: [][]interface{}{
					{int8(1), int8(2)},
				},
			},
			expected: map[string]int{
				"foo": 1,
				"bar": 2,
			},
		},
	}
	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			dstVal := newDstValue(tc.expected)
			err := doScan(dstVal, tc.rows)
			require.NoError(t, err)
			assertDstValueEqual(t, tc.expected, dstVal)
		})
	}
}

func TestDoScan_PrimitiveTypeDestination(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name     string
		rows     *fakeRows
		expected interface{}
	}{
		{
			name: "string",
			rows: &fakeRows{
				columns: []string{"bar"},
				data: [][]interface{}{
					{"bar val"},
				},
			},
			expected: "bar val",
		},
		{
			name: "string by ptr",
			rows: &fakeRows{
				columns: []string{"bar"},
				data: [][]interface{}{
					{"bar val"},
				},
			},
			expected: "bar val",
		},
		{
			name: "slice",
			rows: &fakeRows{
				columns: []string{"bar"},
				data: [][]interface{}{
					{[]string{"bar val", "bar val 2", "bar val 3"}},
				},
			},
			expected: []string{"bar val", "bar val 2", "bar val 3"},
		},
	}
	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			dstVal := newDstValue(tc.expected)
			err := doScan(dstVal, tc.rows)
			require.NoError(t, err)
			assertDstValueEqual(t, tc.expected, dstVal)
		})
	}
}

func TestDoScan_InvalidDestination_ReturnsErr(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name        string
		rows        *fakeRows
		dst         interface{}
		expectedErr string
	}{
		{
			name: "struct doesn't have a corresponding field",
			rows: &fakeRows{
				columns: []string{"foo", "bar"},
				data: [][]interface{}{
					{"foo val", "bar val"},
				},
			},
			dst: struct {
				Bar string
			}{},
			expectedErr: "column: 'foo': no corresponding field found or it's unexported in struct { Bar string }",
		},
		{
			name: "the corresponding struct is unexported",
			rows: &fakeRows{
				columns: []string{"foo", "bar"},
				data: [][]interface{}{
					{"foo val", "bar val"},
				},
			},
			dst: struct {
				foo string
				Bar string
			}{},
			expectedErr: "column: 'foo': no corresponding field found or it's unexported in struct { foo string; Bar string }",
		},
		{
			name: "struct fields contain duplicated tag",
			rows: &fakeRows{
				columns: []string{"foo_column", "bar"},
				data: [][]interface{}{
					{"foo val", "bar val"},
				},
			},
			dst: struct {
				Foo string `db:"foo_column"`
				Bar string `db:"foo_column"`
			}{},
			expectedErr: "Column must have exactly one field pointing to it; " +
				"found 2 fields with indexes [0] and [1] pointing to 'foo_column' in " +
				"struct { Foo string \"db:\\\"foo_column\\\"\"; Bar string \"db:\\\"foo_column\\\"\" }",
		},
		{
			name: "map non string key is not allowed",
			rows: &fakeRows{
				columns: []string{"foo", "bar"},
				data: [][]interface{}{
					{"foo val", "bar val"},
				},
			},
			dst:         map[int]interface{}{},
			expectedErr: "invalid type map[int]interface {}: map must have string key, got: int",
		},
		{
			name: "value can't be converted to the map element type",
			rows: &fakeRows{
				columns: []string{"foo"},
				data: [][]interface{}{
					{"foo val"},
				},
			},
			dst:         map[string]int{},
			expectedErr: "Column 'foo' value of type string can'be set into map[string]int",
		},
		{
			name: "primitive type, rows contain 0 columns",
			rows: &fakeRows{
				data: [][]interface{}{
					{"bar val"},
				},
				columns: []string{},
			},
			dst:         "",
			expectedErr: "to scan into a primitive type, columns number must be exactly 1, got: 0",
		},
		{
			name: "primitive type, rows contain more than 1 column",
			rows: &fakeRows{
				columns: []string{"foo", "bar"},
				data: [][]interface{}{
					{"foo val", "bar val"},
				},
			},
			dst:         "",
			expectedErr: "to scan into a primitive type, columns number must be exactly 1, got: 2",
		},
	}
	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			dstVal := newDstValue(tc.dst)
			err := doScan(dstVal, tc.rows)
			assert.EqualError(t, err, tc.expectedErr)
		})
	}

}

func TestDoScan_RowsContainDuplicatedColumn_ReturnErr(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name string
		dst  interface{}
	}{
		{
			name: "struct destination",
			dst: struct {
				Foo string
			}{},
		},
		{
			name: "map destination",
			dst:  map[string]interface{}{},
		},
	}
	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			rows := &fakeRows{
				columns: []string{"foo", "foo"},
				data: [][]interface{}{
					{"foo val", "bar val"},
				},
			}
			dstVal := newDstValue(tc.dst)
			err := doScan(dstVal, rows)
			expectedErr := "row contains duplicated column 'foo'"
			assert.EqualError(t, err, expectedErr)
		})
	}

}

func TestScanAll_NonEmptySlice_ResetsDstSlice(t *testing.T) {
	t.Parallel()
	fr := &fakeRows{
		columns: []string{"bar"},
		data: [][]interface{}{
			{"bar val"},
			{"bar val 2"},
			{"bar val 3"},
		},
	}
	expected := []string{"bar val", "bar val 2", "bar val 3"}
	got := []string{"junk data", "junk data 2"}
	err := pgxquery.ScanAll(&got, fr)
	require.NoError(t, err)
	assert.Equal(t, expected, got)
}

func TestParseDestination_ValidDst_ReturnsElemReflectValue(t *testing.T) {
	t.Parallel()
	var dst struct{ Foo string }
	got, err := pgxquery.ParseDestination(&dst)
	expected := reflect.ValueOf(&dst).Elem()
	require.NoError(t, err)
	assert.Equal(t, expected, got)
}

func TestParseDestination_InvalidDst_ReturnsErr(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name        string
		dst         interface{}
		expectedErr string
	}{
		{
			name: "non pointer",
			dst: struct {
				Foo string
			}{},
			expectedErr: "destination must be a pointer, got: struct { Foo string }",
		},
		{
			name:        "map",
			dst:         map[string]interface{}{},
			expectedErr: "destination must be a pointer, got: map[string]interface {}",
		},
		{
			name:        "slice",
			dst:         []struct{ Foo string }{},
			expectedErr: "destination must be a pointer, got: []struct { Foo string }",
		},
		{
			name:        "nil",
			dst:         nil,
			expectedErr: "destination must be a non nil pointer",
		},
		{
			name:        "(*int)(nil)",
			dst:         (*int)(nil),
			expectedErr: "destination must be a non nil pointer",
		},
	}
	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			_, err := pgxquery.ParseDestination(tc.dst)
			assert.EqualError(t, err, tc.expectedErr)
		})
	}
}

func TestScanOne_ZeroRows_ReturnNotFoundErr(t *testing.T) {
	t.Parallel()
	rows := &fakeRows{
		columns: []string{"foo"},
		data:    [][]interface{}{},
	}
	var dst string
	err := pgxquery.ScanOne(&dst, rows)
	assert.True(t, pgxquery.NotFound(err))
}

func TestScanOne_MultipleRows_ReturnErr(t *testing.T) {
	t.Parallel()
	rows := &fakeRows{
		columns: []string{"foo"},
		data: [][]interface{}{
			{"bar val"},
			{"bar val 2"},
			{"bar val 3"},
		},
	}
	var dst string
	err := pgxquery.ScanOne(&dst, rows)
	expectedErr := "expected 1 row, got: 3"
	assert.EqualError(t, err, expectedErr)
}
