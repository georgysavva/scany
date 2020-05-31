package pgxscan_test

import (
	"reflect"
	"testing"

	"github.com/georgysavva/pgxscan"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type FooNested struct {
	FooNested string
}

type BarNested struct {
	BarNested string
}

type nestedUnexported struct {
	FooNested string
	BarNested string
}

func TestDoScan_StructDestination_Succeeds(t *testing.T) {
	t.Parallel()
	type jsonObj struct {
		SomeField string
	}
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
		{
			name: "embedded struct is filled from columns without prefix",
			rows: &fakeRows{
				columns: []string{"foo", "bar", "foo_nested", "bar_nested"},
				data: [][]interface{}{
					{"foo val", "bar val", "foo nested val", "bar nested val"},
				},
			},
			expected: struct {
				FooNested
				BarNested
				Foo string
				Bar string
			}{
				FooNested: FooNested{
					FooNested: "foo nested val",
				},
				BarNested: BarNested{
					BarNested: "bar nested val",
				},
				Foo: "foo val",
				Bar: "bar val",
			},
		},
		{
			name: "embedded struct with tag is filled from columns with prefix",
			rows: &fakeRows{
				columns: []string{"foo", "bar", "nested.foo_nested"},
				data: [][]interface{}{
					{"foo val", "bar val", "foo nested val"},
				},
			},
			expected: struct {
				FooNested `db:"nested"`
				Foo       string
				Bar       string
			}{
				FooNested: FooNested{
					FooNested: "foo nested val",
				},
				Foo: "foo val",
				Bar: "bar val",
			},
		},
		{
			name: "embedded struct by ptr is initialized and filled",
			rows: &fakeRows{
				columns: []string{"foo", "bar", "foo_nested"},
				data: [][]interface{}{
					{"foo val", "bar val", "foo nested val"},
				},
			},
			expected: struct {
				*FooNested
				Foo string
				Bar string
			}{
				FooNested: &FooNested{
					FooNested: "foo nested val",
				},
				Foo: "foo val",
				Bar: "bar val",
			},
		},
		{
			name: "embedded struct by ptr isn't initialized if not filled",
			rows: &fakeRows{
				columns: []string{"foo", "bar"},
				data: [][]interface{}{
					{"foo val", "bar val"},
				},
			},
			expected: struct {
				*FooNested
				Foo string
				Bar string
			}{
				FooNested: nil,
				Foo:       "foo val",
				Bar:       "bar val",
			},
		},
		{
			name: "embedded struct with ignore tag isn't filled",
			rows: &fakeRows{
				columns: []string{"nested.foo_nested", "nested.bar_nested"},
				data: [][]interface{}{
					{"foo nested val", "bar nested val"},
				},
			},
			expected: struct {
				FooNested `db:"-"`
				Foo       string `db:"nested.foo_nested"`
				Bar       string `db:"nested.bar_nested"`
			}{
				FooNested: FooNested{},
				Foo:       "foo nested val",
				Bar:       "bar nested val",
			},
		},
		{
			name: "nested struct is filled from a json column",
			rows: &fakeRows{
				columns: []string{"foo", "json"},
				data: [][]interface{}{
					{"foo val", jsonObj{SomeField: "some field val"}},
				},
			},
			expected: struct {
				Json jsonObj
				Foo  string
			}{
				Json: jsonObj{SomeField: "some field val"},
				Foo:  "foo val",
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

func TestDoScan_InvalidStructDestination_ReturnsErr(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name        string
		rows        *fakeRows
		dst         interface{}
		expectedErr string
	}{
		{
			name: "doesn't have a corresponding field",
			rows: &fakeRows{
				columns: []string{"foo", "bar"},
				data: [][]interface{}{
					{"foo val", "bar val"},
				},
			},
			dst: struct {
				Bar string
			}{},
			expectedErr: "column: 'foo': no corresponding field found or it's unexported in " +
				"struct { Bar string }",
		},
		{
			name: "the corresponding field is unexported",
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
			expectedErr: "column: 'foo': no corresponding field found or it's unexported in " +
				"struct { foo string; Bar string }",
		},
		{
			name: "embedded struct is unexported",
			rows: &fakeRows{
				columns: []string{"foo", "bar", "foo_nested", "bar_nested"},
				data: [][]interface{}{
					{"foo val", "bar val", "foo nested val", "bar nested val"},
				},
			},
			dst: struct {
				nestedUnexported
				Foo string
				Bar string
			}{},
			expectedErr: "column: 'foo_nested': no corresponding field found or it's unexported in " +
				"struct { pgxscan_test.nestedUnexported; Foo string; Bar string }",
		},
		{
			name: "nested non embedded structs aren't allowed",
			rows: &fakeRows{
				columns: []string{"foo", "bar", "foo_nested", "bar_nested"},
				data: [][]interface{}{
					{"foo val", "bar val", "foo nested val", "bar nested val"},
				},
			},
			dst: struct {
				Nested FooNested
				Foo    string
				Bar    string
			}{},
			expectedErr: "column: 'foo_nested': no corresponding field found or it's unexported in " +
				"struct { Nested pgxscan_test.FooNested; Foo string; Bar string }",
		},
		{
			name: "the corresponding field is unexported",
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
			expectedErr: "column: 'foo': no corresponding field found or it's unexported in " +
				"struct { foo string; Bar string }",
		},
		{
			name: "fields contain duplicated tag",
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

func TestDoScan_MapDestination_Succeeds(t *testing.T) {
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

func TestDoScan_InvalidMapDestination_ReturnsErr(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name        string
		rows        *fakeRows
		dst         interface{}
		expectedErr string
	}{
		{
			name: "non string key is not allowed",
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
			name: "value can't be converted to the element type",
			rows: &fakeRows{
				columns: []string{"foo"},
				data: [][]interface{}{
					{"foo val"},
				},
			},
			dst:         map[string]int{},
			expectedErr: "Column 'foo' value of type string can'be set into map[string]int",
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

func TestDoScan_PrimitiveTypeDestination_Succeeds(t *testing.T) {
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

func TestDoScan_InvalidPrimitiveTypeDestination_ReturnsErr(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name        string
		rows        *fakeRows
		dst         interface{}
		expectedErr string
	}{
		{
			name: "rows contain 0 columns",
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
			name: "rows contain more than 1 column",
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

func TestDoScan_RowsContainDuplicatedColumn_ReturnsErr(t *testing.T) {
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

func TestParseDestination_ValidDst_ReturnsElemReflectValue(t *testing.T) {
	t.Parallel()
	var dst struct{ Foo string }
	got, err := pgxscan.ParseDestination(&dst)
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
			_, err := pgxscan.ParseDestination(tc.dst)
			assert.EqualError(t, err, tc.expectedErr)
		})
	}
}
