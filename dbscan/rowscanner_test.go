package dbscan_test

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/georgysavva/scany/dbscan"
)

type FooNested struct {
	FooNested string
}

type BarNested struct {
	BarNested string
}

type jsonObj struct {
	Key string
}

type NestedLevel1 struct {
	NestedLevel2
}

type NestedLevel2 struct {
	Foo string
}

type NestedWithTagLevel1 struct {
	NestedWithTagLevel2 `db:"nested2"`
}

type NestedWithTagLevel2 struct {
	Bar string `db:"bar_column"`
}

type AmbiguousNested1 struct {
	Foo string
}

type AmbiguousNested2 struct {
	Foo string
}

func TestRowScanner_Scan_structDestination(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name     string
		query    string
		expected interface{}
	}{
		{
			name: "fields without tag are filled from column via snake case mapping",
			query: `
				SELECT 'foo val' AS foo_column, 'bar val' AS bar_column
			`,
			expected: struct {
				FooColumn string
				BarColumn string
			}{
				FooColumn: "foo val",
				BarColumn: "bar val",
			},
		},
		{
			name: "fields with tag are filled from columns via tag",
			query: `
				SELECT 'foo val' AS foo_column, 'bar val' AS bar_column
			`,
			expected: struct {
				Foo string `db:"foo_column"`
				Bar string `db:"bar_column"`
			}{
				Foo: "foo val",
				Bar: "bar val",
			},
		},
		{
			name: "string field by ptr",
			query: `
				SELECT 'foo val' AS foo, 'bar val' AS bar
			`,
			expected: struct {
				Foo *string
				Bar string
			}{
				Foo: makeStrPtr("foo val"),
				Bar: "bar val",
			},
		},
		{
			name: "field with ignore tag isn't filled",
			query: `
				SELECT 'foo val' AS foo
			`,
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
			query: `
				SELECT 'foo val' AS foo, 'bar val' AS bar,
					'foo nested val' as foo_nested, 'bar nested val' as bar_nested
			`,
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
			query: `
				SELECT 'foo val' AS foo, 'bar val' AS bar,
					'foo nested val' as "nested.foo_nested"
			`,
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
			name: "multiple level embedded struct",
			query: `
				SELECT 'foo val' AS foo, 'bar val' AS "nested1.nested2.bar_column"
			`,
			expected: struct {
				NestedLevel1
				NestedWithTagLevel1 `db:"nested1"`
			}{
				NestedLevel1:        NestedLevel1{NestedLevel2{Foo: "foo val"}},
				NestedWithTagLevel1: NestedWithTagLevel1{NestedWithTagLevel2{Bar: "bar val"}},
			},
		},
		{
			name: "embedded struct by ptr is initialized and filled",
			query: `
				SELECT 'foo val' AS foo, 'bar val' AS bar,
					'foo nested val' as foo_nested
			`,
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
			query: `
				SELECT 'foo val' AS foo, 'bar val' AS bar
			`,
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
			query: `
				SELECT 'foo nested val' as "nested.foo_nested", 
					'bar nested val' as "nested.bar_nested"
			`,
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
			name: "ambiguous fields: scanned in the topmost field",
			query: `
				SELECT 'foo val' as foo
			`,
			expected: struct {
				AmbiguousNested1
				AmbiguousNested2
			}{
				AmbiguousNested1: AmbiguousNested1{Foo: "foo val"},
			},
		},
		{
			name: "ambiguous fields: scanned in the outermost field",
			query: `
				SELECT 'foo val' as foo
			`,
			expected: struct {
				AmbiguousNested1
				AmbiguousNested2
				Foo string
			}{
				Foo: "foo val",
			},
		},
		{
			name: "nested struct is filled from a json column",
			query: `
				SELECT '{"key": "key val"}'::JSON AS foo_json, 'foo val' AS foo
			`,
			expected: struct {
				FooJSON jsonObj
				Foo     string
			}{
				FooJSON: jsonObj{Key: "key val"},
				Foo:     "foo val",
			},
		},
		{
			name: "nested struct by ptr is filled from a json column",
			query: `
				SELECT '{"key": "key val"}'::JSON AS foo_json, 'foo val' AS foo
			`,
			expected: struct {
				FooJSON *jsonObj
				Foo     string
			}{
				FooJSON: &jsonObj{Key: "key val"},
				Foo:     "foo val",
			},
		},
		{
			name: "time field is filled from a timestamp column",
			query: `
				SELECT '2020-10-16 09:36:59+00:00'::timestamp AS foo
			`,
			expected: struct {
				Foo time.Time
			}{
				Foo: time.Date(2020, 10, 16, 9, 36, 59, 0, time.UTC),
			},
		},
		{
			name: "map field is filled from a json column",
			query: `
				SELECT '{"key": "key val"}'::JSON AS foo_json, 'foo val' AS foo
			`,
			expected: struct {
				FooJSON map[string]interface{}
				Foo     string
			}{
				FooJSON: map[string]interface{}{"key": "key val"},
				Foo:     "foo val",
			},
		},
		{
			name: "map field by ptr is filled from a json column",
			query: `
				SELECT '{"key": "key val"}'::JSON AS foo_json, 'foo val' AS foo
			`,
			expected: struct {
				FooJSON *map[string]interface{}
				Foo     string
			}{
				FooJSON: &map[string]interface{}{"key": "key val"},
				Foo:     "foo val",
			},
		},
	}
	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			rows := queryRows(t, tc.query)
			dst := allocateDestination(tc.expected)
			err := scan(t, dst, rows)
			require.NoError(t, err)
			assertDestinationEqual(t, tc.expected, dst)
		})
	}
}

type nestedUnexported struct {
	FooNested string
	BarNested string
}

func TestRowScanner_Scan_invalidStructDestination_returnsErr(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name        string
		query       string
		dst         interface{}
		expectedErr string
	}{
		{
			name: "doesn't have a corresponding field",
			query: `
				SELECT 'foo val' AS foo, 'bar val' AS bar
			`,
			dst: &struct {
				Bar string
			}{},
			expectedErr: "scany: column: 'foo': no corresponding field found, or it's unexported in " +
				"struct { Bar string }",
		},
		{
			name: "the corresponding field is unexported",
			query: `
				SELECT 'foo val' AS foo, 'bar val' AS bar
			`,
			dst: &struct {
				foo string
				Bar string
			}{},
			expectedErr: "scany: column: 'foo': no corresponding field found, or it's unexported in " +
				"struct { foo string; Bar string }",
		},
		{
			name: "embedded struct is unexported",
			query: `
				SELECT 'foo val' AS foo, 'bar val' AS bar,
					'foo nested val' as foo_nested, 'bar nested val' as bar_nested
			`,
			dst: &struct {
				nestedUnexported
				Foo string
				Bar string
			}{},
			expectedErr: "scany: column: 'foo_nested': no corresponding field found, or it's unexported in " +
				"struct { dbscan_test.nestedUnexported; Foo string; Bar string }",
		},
		{
			name: "nested non embedded structs aren't allowed",
			query: `
				SELECT 'foo val' AS foo, 'bar val' AS bar,
					'foo nested val' as foo_nested, 'bar nested val' as bar_nested
			`,
			dst: &struct {
				Nested FooNested
				Foo    string
				Bar    string
			}{},
			expectedErr: "scany: column: 'foo_nested': no corresponding field found, or it's unexported in " +
				"struct { Nested dbscan_test.FooNested; Foo string; Bar string }",
		},
		{
			name: "field type does not match with column type",
			query: `
				SELECT 'foo val' AS foo, 'bar val' AS bar
			`,
			dst: &struct {
				Foo int
				Bar string
			}{},
			expectedErr: "scany: scan row into struct fields: can't scan into dest[0]: unable to assign to *int",
		},
		{
			name: "non struct embedded field",
			query: `
				SELECT 'foo val' AS foo, 'text' AS string
			`,
			dst: &struct {
				string
				Foo string
			}{},
			expectedErr: "scany: column: 'string': no corresponding field found, " +
				"or it's unexported in struct { string; Foo string }",
		},
	}
	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			rows := queryRows(t, tc.query)
			err := scan(t, tc.dst, rows)
			assert.EqualError(t, err, tc.expectedErr)
		})
	}
}

func TestRowScanner_Scan_mapDestination(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name     string
		query    string
		expected interface{}
	}{
		{
			name: "map[string]interface{}",
			query: `
				SELECT 'foo val' AS foo, 'bar val' AS bar
			`,
			expected: map[string]interface{}{
				"foo": "foo val",
				"bar": "bar val",
			},
		},
		{
			name: "map[string]string{}",
			query: `
				SELECT 'foo val' AS foo, 'bar val' AS bar
			`,
			expected: map[string]string{
				"foo": "foo val",
				"bar": "bar val",
			},
		},
		{
			name: "map[string]*string{}",
			query: `
				SELECT 'foo val' AS foo, NULL AS bar
			`,
			expected: map[string]*string{
				"foo": makeStrPtr("foo val"),
				"bar": nil,
			},
		},
		{
			name: "map[string]struct{}",
			query: `
				SELECT '{"key": "key val"}'::JSON AS foo_json, '{"key": "key val 2"}'::JSON AS bar_json
			`,
			expected: map[string]jsonObj{
				"foo_json": {Key: "key val"},
				"bar_json": {Key: "key val 2"},
			},
		},
		{
			name: "map[string]*struct{}",
			query: `
				SELECT '{"key": "key val"}'::JSON AS foo_json, NULL AS bar_json
			`,
			expected: map[string]*jsonObj{
				"foo_json": {Key: "key val"},
				"bar_json": nil,
			},
		},
		{
			name: "map[string]map[string]interface{}",
			query: `
				SELECT '{"key": "key val"}'::JSON AS foo_json, '{"key": "key val 2"}'::JSON AS bar_json
			`,
			expected: map[string]map[string]interface{}{
				"foo_json": {"key": "key val"},
				"bar_json": {"key": "key val 2"},
			},
		},
		{
			name: "map[string]*map[string]interface{}",
			query: `
				SELECT '{"key": "key val"}'::JSON AS foo_json, NULL AS bar_json
			`,
			expected: map[string]*map[string]interface{}{
				"foo_json": {"key": "key val"},
				"bar_json": nil,
			},
		},
	}
	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			rows := queryRows(t, tc.query)
			dst := allocateDestination(tc.expected)
			err := scan(t, dst, rows)
			require.NoError(t, err)
			assertDestinationEqual(t, tc.expected, dst)
		})
	}
}

func TestRowScanner_Scan_invalidMapDestination_returnsErr(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name        string
		query       string
		dst         interface{}
		expectedErr string
	}{
		{
			name:        "non string key",
			query:       singleRowsQuery,
			dst:         &map[int]interface{}{},
			expectedErr: "scany: invalid type map[int]interface {}: map must have string key, got: int",
		},
		{
			name: "value type does not match with column type",
			query: `
				SELECT 'foo val' AS foo
			`,
			dst:         &map[string]int{},
			expectedErr: "scany: scan rows into map: can't scan into dest[0]: unable to assign to *int",
		},
	}
	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			rows := queryRows(t, tc.query)
			err := scan(t, tc.dst, rows)
			assert.EqualError(t, err, tc.expectedErr)
		})
	}
}

func TestRowScanner_Scan_primitiveTypeDestination(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name     string
		query    string
		expected interface{}
	}{
		{
			name: "string",
			query: `
				SELECT 'foo val' AS foo 
			`,
			expected: "foo val",
		},
		{
			name: "string by ptr",
			query: `
				SELECT 'foo val' AS foo 
			`,
			expected: makeStrPtr("foo val"),
		},
		{
			name: "slice",
			query: `
				SELECT ARRAY('foo val', 'foo val 2', 'foo val 3') AS foo 
			`,
			expected: []string{"foo val", "foo val 2", "foo val 3"},
		},
		{
			name: "slice by ptr",
			query: `
				SELECT ARRAY('foo val', 'foo val 2', 'foo val 3') AS foo 
			`,
			expected: &[]string{"foo val", "foo val 2", "foo val 3"},
		},
		{
			name: "struct by ptr treated as primitive type",
			query: `
				SELECT '{"key": "key val"}'::JSON AS foo_json
			`,

			expected: &jsonObj{Key: "key val"},
		},
		{
			name: "map by ptr treated as primitive type",
			query: `
				SELECT '{"key": "key val"}'::JSON AS foo_json
			`,
			expected: &map[string]interface{}{"key": "key val"},
		},
	}
	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			rows := queryRows(t, tc.query)
			dst := allocateDestination(tc.expected)
			err := scan(t, dst, rows)
			require.NoError(t, err)
			assertDestinationEqual(t, tc.expected, dst)
		})
	}
}

func TestRowScanner_Scan_primitiveTypeDestinationDoesNotMatchWithColumnType_returnsErr(t *testing.T) {
	t.Parallel()
	query := `
		SELECT 'foo val' AS foo
	`
	rows := queryRows(t, query)
	expectedErr := "scany: scan row value into a primitive type: can't scan into dest[0]: unable to assign to *int"
	dst := new(int)
	err := scan(t, dst, rows)
	assert.EqualError(t, err, expectedErr)
}

func TestRowScanner_Scan_primitiveTypeDestinationRowsContainMoreThanOneColumn_returnsErr(t *testing.T) {
	t.Parallel()
	query := `
		SELECT '1 val' AS column1, '2 val' AS column2
	`
	rows := queryRows(t, query)
	expectedErr := "scany: to scan into a primitive type, columns number must be exactly 1, got: 2"
	dst := new(string)
	err := scan(t, dst, rows)
	assert.EqualError(t, err, expectedErr)
}

// It seems that there is no way to select result set with 0 columns from crdb server.
// So this type exists in order to check that dbscan handles this cases properly.
type emptyRow struct{}

func (er emptyRow) Scan(_ ...interface{}) error { return nil }
func (er emptyRow) Next() bool                  { return true }
func (er emptyRow) Columns() ([]string, error)  { return []string{}, nil }
func (er emptyRow) Close() error                { return nil }
func (er emptyRow) Err() error                  { return nil }

func TestRowScanner_Scan_primitiveTypeDestinationRowsContainZeroColumns_returnsErr(t *testing.T) {
	t.Parallel()
	rows := emptyRow{}
	expectedErr := "scany: to scan into a primitive type, columns number must be exactly 1, got: 0"
	dst := new(string)
	err := scan(t, dst, rows)
	assert.EqualError(t, err, expectedErr)
}

func TestRowScanner_Scan_rowsContainDuplicateColumns_returnsErr(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name string
		dst  interface{}
	}{
		{
			name: "struct destination",
			dst: &struct {
				Foo string
			}{},
		},
		{
			name: "map destination",
			dst:  &map[string]interface{}{},
		},
	}
	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			query := `
				SELECT 'foo val' AS foo, 'foo val' AS foo
			`
			rows := queryRows(t, query)
			expectedErr := "scany: rows contain a duplicate column 'foo'"
			err := scan(t, tc.dst, rows)
			assert.EqualError(t, err, expectedErr)
		})
	}
}

func TestRowScanner_Scan_invalidDst_returnsErr(t *testing.T) {
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
			expectedErr: "scany: destination must be a pointer, got: struct { Foo string }",
		},
		{
			name:        "map",
			dst:         map[string]interface{}{},
			expectedErr: "scany: destination must be a pointer, got: map[string]interface {}",
		},
		{
			name:        "slice",
			dst:         []struct{ Foo string }{},
			expectedErr: "scany: destination must be a pointer, got: []struct { Foo string }",
		},
		{
			name:        "nil",
			dst:         nil,
			expectedErr: "scany: destination must be a non nil pointer",
		},
		{
			name:        "(*int)(nil)",
			dst:         (*int)(nil),
			expectedErr: "scany: destination must be a non nil pointer",
		},
	}
	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			rows := queryRows(t, `SELECT 1`)
			err := scan(t, tc.dst, rows)
			assert.EqualError(t, err, tc.expectedErr)
		})
	}
}

func TestRowScanner_Scan_startCalledExactlyOnce(t *testing.T) {
	t.Parallel()
	dbscan.DoTestRowScannerStartCalledExactlyOnce(t, queryRows)
}
