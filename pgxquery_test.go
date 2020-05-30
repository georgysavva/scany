package pgxquery_test

import (
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
					{"foo val", "bar val"},
				},
			},
			expected: struct {
				Foo string
				Bar string
			}{
				Foo: "foo val",
				Bar: "bar val",
			},
		},
		{
			name: "fields without tags automatically mapped to snake case",
			rows: &fakeRows{
				columns: []string{"foo_column", "bar_column"},
				data: [][]interface{}{
					{"foo val", "bar val"},
				},
			},
			expected: struct {
				FooColumn string
				BarColumn string
			}{
				FooColumn: "foo val",
				BarColumn: "bar val",
			},
		},
		{
			name: "fields with tags",
			rows: &fakeRows{
				columns: []string{"foo_column", "bar_column"},
				data: [][]interface{}{
					{"foo val", "bar val"},
				},
			},
			expected: struct {
				Foo string `db:"foo_column"`
				Bar string `db:"bar_column"`
			}{
				Foo: "foo val",
				Bar: "bar val",
			},
		},
		{
			name: "field not found",
			rows: &fakeRows{
				columns: []string{"foo", "bar"},
				data: [][]interface{}{
					{"foo val", "bar val"},
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
					{"foo val", "bar val"},
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
					{"foo val", "bar val"},
				},
			},
			expected: struct {
				Foo string `db:"-"`
				Bar string
			}{},
			errString: "column: 'foo': no corresponding field found or it's unexported in " +
				"struct { Foo string \"db:\\\"-\\\"\"; Bar string }",
		},
		{
			name: "field is unexported",
			rows: &fakeRows{
				columns: []string{"foo", "bar"},
				data: [][]interface{}{
					{"foo val", "bar val"},
				},
			},
			expected: struct {
				foo string
				Bar string
			}{},
			errString: "column: 'foo': no corresponding field found or it's unexported in struct { foo string; Bar string }",
		},
		{
			name: "field with tag is unexported",
			rows: &fakeRows{
				columns: []string{"foo_column", "bar"},
				data: [][]interface{}{
					{"foo val", "bar val"},
				},
			},
			expected: struct {
				foo string `db:"foo_column"`
				Bar string
			}{},
			errString: "column: 'foo_column': no corresponding field found or it's unexported in " +
				"struct { foo string \"db:\\\"foo_column\\\"\"; Bar string }",
		},
		{
			name: "fields contain duplicated tag",
			rows: &fakeRows{
				columns: []string{"foo_column", "bar"},
				data: [][]interface{}{
					{"foo val", "bar val"},
				},
			},
			expected: struct {
				Foo string `db:"foo_column"`
				Bar string `db:"foo_column"`
			}{},
			errString: "Column must have exactly one field pointing to it; " +
				"found 2 fields with indexes [0] and [1] pointing to 'foo_column' in " +
				"struct { Foo string \"db:\\\"foo_column\\\"\"; Bar string \"db:\\\"foo_column\\\"\" }",
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
					{"foo val", "bar val"},
				},
			},
			expected: map[string]interface{}{
				"foo": "foo val",
				"bar": "bar val",
			},
		},
		{
			name: "non interface{} element type",
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
			name: "value converted to the right type",
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
		{
			name: "non string key is not allowed",
			rows: &fakeRows{
				columns: []string{"foo", "bar"},
				data: [][]interface{}{
					{"foo val", "bar val"},
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
					{"foo val", "bar val"},
				},
			},
			expected:  map[string]interface{}{},
			errString: "row contains duplicated column 'foo'",
		},
		{
			name: "invalid element type",
			rows: &fakeRows{
				columns: []string{"foo"},
				data: [][]interface{}{
					{"foo val"},
				},
			},
			expected:  map[string]int{},
			errString: "Column 'foo' value of type string can'be set into map[string]int",
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
			name: "slice as single column",
			rows: &fakeRows{
				columns: []string{"bar"},
				data: [][]interface{}{
					{[]string{"bar val", "bar val 2", "bar val 3"}},
				},
			},
			expected: []string{"bar val", "bar val 2", "bar val 3"},
		},
		{
			name: "0 columns",
			rows: &fakeRows{
				data: [][]interface{}{
					{"bar val"},
				},
				columns: []string{},
			},
			expected:  "",
			errString: "to scan into a primitive type, columns number must be exactly 1, got: 0",
		},
		{
			name: "more than 1 column",
			rows: &fakeRows{
				columns: []string{"foo", "bar"},
				data: [][]interface{}{
					{"foo val", "bar val"},
				},
			},
			expected:  "",
			errString: "to scan into a primitive type, columns number must be exactly 1, got: 2",
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
			tc.exactlyOneRow = false
			tc.test(t)
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

func TestScan_InvalidDestination_ReturnsErr(t *testing.T) {
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

			t.Run("scan one", func(t *testing.T) {
				fr := &fakeRows{}
				err := pgxquery.ScanOne(tc.dst, fr)
				assert.EqualError(t, err, tc.expectedErr)
			})

			t.Run("scan row", func(t *testing.T) {
				fr := &fakeRows{}
				err := pgxquery.ScanRow(tc.dst, fr)
				assert.EqualError(t, err, tc.expectedErr)
			})

			t.Run("scan all", func(t *testing.T) {
				fr := &fakeRows{}
				err := pgxquery.ScanAll(tc.dst, fr)
				assert.EqualError(t, err, tc.expectedErr)
			})
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
