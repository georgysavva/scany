package pgxquery_test

import (
	"github.com/georgysavva/pgxquery"
	"github.com/jackc/pgproto3/v2"
	"github.com/jackc/pgx/v4"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"reflect"
	"testing"
)

type fakeRows struct {
	pgx.Rows
	columns       []string
	data          [][]interface{}
	currentRow    []interface{}
	rowsProcessed int
}

func (fr *fakeRows) Scan(dest ...interface{}) error {
	for i, data := range fr.currentRow {
		dst := dest[i]
		dstV := reflect.ValueOf(dst).Elem()
		dstV.Set(reflect.ValueOf(data))
	}
	return nil
}

func (fr *fakeRows) Next() bool {
	if fr.rowsProcessed >= len(fr.data) {
		return false
	}
	fr.currentRow = fr.data[fr.rowsProcessed]
	fr.rowsProcessed++
	return true
}

func (fr *fakeRows) FieldDescriptions() []pgproto3.FieldDescription {
	fields := make([]pgproto3.FieldDescription, len(fr.columns))
	for i, column := range fr.columns {
		fields[i] = pgproto3.FieldDescription{Name: []byte(column)}
	}
	return fields
}

func (fr *fakeRows) Values() ([]interface{}, error) { return fr.currentRow, nil }

func (fr *fakeRows) Close() {}

func (fr *fakeRows) Err() error { return nil }

type ScanCase struct {
	name          string
	rows          *fakeRows
	expected      interface{}
	errString     string
	exactlyOneRow bool
}

func (tc *ScanCase) test(t *testing.T) {
	t.Parallel()

	dstType := reflect.TypeOf(tc.expected)
	dstValue := reflect.New(dstType)
	dst := dstValue.Interface()

	var err error
	if tc.exactlyOneRow {
		err = pgxquery.ScanOne(dst, tc.rows)
	} else {
		err = pgxquery.ScanAll(dst, tc.rows)
	}

	if tc.errString == "" {
		require.NoError(t, err)
		got := dstValue.Elem().Interface()
		assert.Equal(t, tc.expected, got)
	} else {
		assert.EqualError(t, err, tc.errString)
	}
}

func TestScanOne(t *testing.T) {
	t.Parallel()
	cases := []ScanCase{
		{
			name: "struct smoke",
			rows: &fakeRows{
				columns: []string{"foo", "bar"},
				data: [][]interface{}{
					{4, "bb"},
				},
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
				columns: []string{"foo", "bar"},
				data: [][]interface{}{
					{4, "bb"},
				},
			},
			expected: map[string]interface{}{
				"foo": 4,
				"bar": "bb",
			},
		},
		{
			name: "primitive type smoke",
			rows: &fakeRows{
				columns: []string{"bar"},
				data: [][]interface{}{
					{"bb"},
				},
			},
			expected: "bb",
		},
		{
			name: "struct field not found",
			rows: &fakeRows{
				columns: []string{"foo", "bar"},
				data: [][]interface{}{
					{4, "bb"},
				},
			},
			expected: struct {
				Bar string
			}{},
			errString: "column: 'foo': no corresponding field found or it's unexported in struct { Bar string }",
		},
		{
			name: "struct duplicated column",
			rows: &fakeRows{
				columns: []string{"foo", "foo"},
				data: [][]interface{}{
					{4, "bb"},
				},
			},
			expected: struct {
				Foo string
			}{},
			errString: "row contains duplicated column 'foo'",
		},
		{
			name: "struct field is unexported",
			rows: &fakeRows{
				columns: []string{"foo", "bar"},
				data: [][]interface{}{
					{4, "bb"},
				},
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
				columns: []string{"foo", "bar"},
				data: [][]interface{}{
					{"ff", "bb"},
				},
			},
			expected: map[string]string{
				"foo": "ff",
				"bar": "bb",
			},
		},
		{
			name: "map non string key",
			rows: &fakeRows{
				columns: []string{"foo", "bar"},
				data: [][]interface{}{
					{4, "bb"},
				},
			},
			expected:  map[int]interface{}{},
			errString: "invalid element type map[int]interface {}: map must have string key, got: int",
		},
		{
			name: "map duplicated column",
			rows: &fakeRows{
				columns: []string{"foo", "foo"},
				data: [][]interface{}{
					{4, "bb"},
				},
			},
			expected:  map[string]interface{}{},
			errString: "row contains duplicated column 'foo'",
		},
		{
			name: "map invalid element type",
			rows: &fakeRows{
				columns: []string{"foo", "bar"},
				data: [][]interface{}{
					{4, "bb"},
				},
			},
			expected:  map[string]int{},
			errString: "Column 'bar' value of type string can'be set into map[string]int",
		},
		{
			name: "primitive type 0 columns",
			rows: &fakeRows{
				data: [][]interface{}{
					{"bb"},
				},
				columns: []string{},
			},
			expected:  "",
			errString: "to fillDestination into a primitive type, columns number must be exactly 1, got: 0",
		},
		{
			name: "primitive type more than 1 column",
			rows: &fakeRows{
				columns: []string{"foo", "bar"},
				data: [][]interface{}{
					{"ff", "bb"},
				},
			},
			expected:  "",
			errString: "to fillDestination into a primitive type, columns number must be exactly 1, got: 2",
		},
	}
	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
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
					{4, "bb"},
				},
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
				columns: []string{"foo", "bar"},
				data: [][]interface{}{
					{4, "bb"},
				},
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
				columns: []string{"foo", "bar"},
				data: [][]interface{}{
					{4, "bb"},
				},
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
				columns: []string{"bar"},
				data: [][]interface{}{
					{"bb"},
				},
			},
			expected: []string{"bb"},
		},
	}
	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			tc.exactlyOneRow = false
			tc.test(t)
		})
	}
}
