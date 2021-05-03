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

type JSONObj struct {
	Key string
}

type NestedLevel1 struct {
	NestedLevel2
	Nested2 NestedLevel2
}

type NestedLevel2 struct {
	NestedLevel3
	Foo string
}

type NestedLevel3 struct {
	NestedLevel4
}

type NestedLevel4 struct {
	DeepNested1 string
	DeepNested2 string
}

type NestedWithTagLevel1 struct {
	NestedWithTagLevel2 `db:"nested2_tag_embedded"`
	Nested2Tag          NestedWithTagLevel2 `db:"nested2_tag"`
}

type NestedWithTagLevel2 struct {
	Foo string `db:"foo_column"`
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
			name: "fields with tag are filled from columns via tag that has multiple comma delimited values",
			query: `
				SELECT 'foo val' AS foo_column, 'bar val' AS bar_column
			`,
			expected: struct {
				Foo string `db:"foo_column,other_tag_value"`
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
			name: "string field by ptr NULL value",
			query: `
				SELECT NULL AS foo, 'bar val' AS bar
			`,
			expected: struct {
				Foo *string
				Bar string
			}{
				Foo: nil,
				Bar: "bar val",
			},
		},
		{
			name: "embedded structs are filled from columns without prefix",
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
			name: "nested structs without tag are filled from columns with snake case prefix",
			query: `
				SELECT 'foo val' AS foo, 'bar val' AS bar,
					'foo nested val' as "foo_nested.foo_nested", 'bar nested val' as "bar_nested.bar_nested"
			`,
			expected: struct {
				FooNested FooNested
				BarNested BarNested
				Foo       string
				Bar       string
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
			name: "embedded struct with tag is filled from columns with prefix from the tag",
			query: `
				SELECT 'foo val' AS foo, 'bar val' AS bar,
					'foo nested val' as "foo_nested.foo_nested"
			`,
			expected: struct {
				FooNested `db:"foo_nested"`
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
			name: "nested struct with tag is filled from columns with prefix from the tag",
			query: `
				SELECT 'foo val' AS foo, 'bar val' AS bar,
					'foo nested val' as "foo_nested_prefix.foo_nested"
			`,
			expected: struct {
				FooNested FooNested `db:"foo_nested_prefix"`
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
			name: "nested struct with empty tag is filled from columns without prefix",
			query: `
				SELECT 'foo val' AS foo, 'bar val' AS bar,
					'foo nested val' as "foo_nested"
			`,
			expected: struct {
				FooNested FooNested `db:""`
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
			name: "embedded struct is unexported",
			query: `
				SELECT 'foo val' AS foo, 'bar val' AS bar,
					'foo nested val' as foo_nested, 'bar nested val' as bar_nested
			`,
			expected: struct {
				nestedUnexported
				Foo string
				Bar string
			}{
				nestedUnexported: nestedUnexported{
					FooNested: "foo nested val",
					BarNested: "bar nested val",
				},
				Foo: "foo val",
				Bar: "bar val",
			},
		},
		{
			name: "nested struct is unexported",
			query: `
				SELECT 'foo val' AS foo, 'bar val' AS bar,
					'foo nested val' as "nested.foo_nested", 'bar nested val' as "nested.bar_nested"
			`,
			expected: struct {
				Nested nestedUnexported
				Foo    string
				Bar    string
			}{
				Nested: nestedUnexported{
					FooNested: "foo nested val",
					BarNested: "bar nested val",
				},
				Foo: "foo val",
				Bar: "bar val",
			},
		},
		{
			name: "multiple level nested structs",
			query: `
				SELECT 'foo val 1' AS "foo", 'foo val 2' AS "nested2.foo", 
				'foo val 3' AS "nested1_tag_embedded.nested2_tag_embedded.foo_column",
				'foo val 4' AS "nested1_tag_embedded.nested2_tag.foo_column",
				'foo val 5' AS "nested1.foo", 'foo val 6' AS "nested1.nested2.foo", 
				'foo val 7' AS "nested1_tag.nested2_tag_embedded.foo_column",
				'foo val 8' AS "nested1_tag.nested2_tag.foo_column"
			`,
			expected: struct {
				NestedLevel1
				NestedWithTagLevel1 `db:"nested1_tag_embedded"`
				Nested1             NestedLevel1
				Nested1Tag          NestedWithTagLevel1 `db:"nested1_tag"`
			}{
				NestedLevel1: NestedLevel1{
					NestedLevel2: NestedLevel2{Foo: "foo val 1"},
					Nested2:      NestedLevel2{Foo: "foo val 2"},
				},
				NestedWithTagLevel1: NestedWithTagLevel1{
					NestedWithTagLevel2: NestedWithTagLevel2{Foo: "foo val 3"},
					Nested2Tag:          NestedWithTagLevel2{Foo: "foo val 4"},
				},
				Nested1: NestedLevel1{
					NestedLevel2: NestedLevel2{Foo: "foo val 5"},
					Nested2:      NestedLevel2{Foo: "foo val 6"},
				},
				Nested1Tag: NestedWithTagLevel1{
					NestedWithTagLevel2: NestedWithTagLevel2{Foo: "foo val 7"},
					Nested2Tag:          NestedWithTagLevel2{Foo: "foo val 8"},
				},
			},
		},
		{
			name: "nested structs by ptr are initialized and filled",
			query: `
				SELECT 'foo val' AS foo, 'bar val' AS bar,
					'foo nested val' as foo_nested, 'bar nested val' as "bar_nested.bar_nested"
			`,
			expected: struct {
				*FooNested
				BarNested *BarNested
				Foo       string
				Bar       string
			}{
				FooNested: &FooNested{
					FooNested: "foo nested val",
				},
				BarNested: &BarNested{
					BarNested: "bar nested val",
				},
				Foo: "foo val",
				Bar: "bar val",
			},
		},
		{
			name: "nested structs by ptr are not initialized if not filled",
			query: `
				SELECT 'foo val' AS foo, 'bar val' AS bar
			`,
			expected: struct {
				*FooNested
				BarNested *BarNested
				Foo       string
				Bar       string
			}{
				FooNested: nil,
				BarNested: nil,
				Foo:       "foo val",
				Bar:       "bar val",
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
			name: "struct field is filled from a json column",
			query: `
				SELECT '{"key": "key val"}'::JSON AS foo_json, 'foo val' AS foo
			`,
			expected: struct {
				FooJSON JSONObj
				Foo     string
			}{
				FooJSON: JSONObj{Key: "key val"},
				Foo:     "foo val",
			},
		},
		{
			name: "struct field by ptr is filled from a json column",
			query: `
				SELECT '{"key": "key val"}'::JSON AS foo_json, 'foo val' AS foo
			`,
			expected: struct {
				FooJSON *JSONObj
				Foo     string
			}{
				FooJSON: &JSONObj{Key: "key val"},
				Foo:     "foo val",
			},
		},
		{
			name: "struct field by ptr is filled from a json column with NULL value",
			query: `
				SELECT NULL::JSON AS foo_json, 'foo val' AS foo
			`,
			expected: struct {
				FooJSON *JSONObj
				Foo     string
			}{
				FooJSON: nil,
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
		{
			name: "deeply nested structure is properly mapped",
			query: `
				SELECT 'deep_nested1 val' AS deep_nested1, 'deep_nested2 val' AS deep_nested2
			`,
			expected: NestedLevel1{
				NestedLevel2: NestedLevel2{
					NestedLevel3: NestedLevel3{
						NestedLevel4: NestedLevel4{
							DeepNested1: "deep_nested1 val",
							DeepNested2: "deep_nested2 val",
						},
					},
				},
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
			name: "field with ignore tag isn't filled",
			query: `
				SELECT 'foo val' AS foo
			`,
			dst: &struct {
				Foo string `db:"-"`
			}{},
			expectedErr: "scany: column: 'foo': no corresponding field found, or it's unexported in " +
				"struct { Foo string \"db:\\\"-\\\"\" }",
		},
		{
			name: "nested struct field is unexported",
			query: `
				SELECT 'foo val' AS foo, 'bar val' AS bar,
					'foo nested val' as "foo_nested.foo_nested"
			`,
			dst: &struct {
				fooNested FooNested
				Foo       string
				Bar       string
			}{},
			expectedErr: "scany: column: 'foo_nested.foo_nested': no corresponding field found, or it's unexported in " +
				"struct { fooNested dbscan_test.FooNested; Foo string; Bar string }",
		},
		{
			name: "embedded struct with ignore tag isn't filled",
			query: `
				SELECT 'foo nested val' as "foo_nested" 
			`,
			dst: &struct {
				FooNested `db:"-"`
			}{},
			expectedErr: "scany: column: 'foo_nested': no corresponding field found, or it's unexported in " +
				"struct { dbscan_test.FooNested \"db:\\\"-\\\"\" }",
		},
		{
			name: "nested struct with ignore tag isn't filled",
			query: `
				SELECT 'foo nested val' as "foo_nested.foo_nested" 
			`,
			dst: &struct {
				FooNested FooNested `db:"-"`
			}{},
			expectedErr: "scany: column: 'foo_nested.foo_nested': no corresponding field found, or it's unexported in " +
				"struct { FooNested dbscan_test.FooNested \"db:\\\"-\\\"\" }",
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
				string `db:"string"`
				Foo    string
			}{},
			expectedErr: "scany: column: 'string': no corresponding field found, " +
				"or it's unexported in struct { string \"db:\\\"string\\\"\"; Foo string }",
		},
		{
			name: "embedded struct as destination field",
			query: `
				SELECT '{"key": "key val"}'::JSON AS foo_json, 'foo val' AS foo
			`,
			dst: &struct {
				JSONObj `db:"foo_json"`
				Foo     string
			}{},
			expectedErr: "scany: column: 'foo_json': no corresponding field found, " +
				"or it's unexported in struct { dbscan_test.JSONObj \"db:\\\"foo_json\\\"\"; Foo string }",
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
			expected: map[string]JSONObj{
				"foo_json": {Key: "key val"},
				"bar_json": {Key: "key val 2"},
			},
		},
		{
			name: "map[string]*struct{}",
			query: `
				SELECT '{"key": "key val"}'::JSON AS foo_json, NULL AS bar_json
			`,
			expected: map[string]*JSONObj{
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

			expected: &JSONObj{Key: "key val"},
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
