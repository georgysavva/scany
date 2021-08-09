package dbscan_test

import (
	"context"
	"flag"
	"os"
	"testing"

	"github.com/cockroachdb/cockroach-go/v2/testserver"
	"github.com/jackc/pgx/v4/pgxpool"
	_ "github.com/jackc/pgx/v4/stdlib"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/georgysavva/scany/dbscan"
)

var (
	testDB  *pgxpool.Pool
	ctx     = context.Background()
	testAPI *dbscan.API
)

func TestScanAll(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name     string
		query    string
		expected interface{}
	}{
		{
			name: "slice of structs",
			query: `
				SELECT *
				FROM (
					VALUES ('foo val', 'bar val'), ('foo val 2', 'bar val 2'), ('foo val 3', 'bar val 3')
				) AS t (foo, bar)
			`,
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
			query: `
				SELECT *
				FROM (
					VALUES ('foo val', 'bar val'), ('foo val 2', 'bar val 2'), ('foo val 3', 'bar val 3')
				) AS t (foo, bar)
			`,
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
			query: `
				SELECT *
				FROM (
					VALUES ('foo val', 'bar val'), ('foo val 2', 'bar val 2'), ('foo val 3', 'bar val 3')
				) AS t (foo, bar)
			`,
			expected: []map[string]interface{}{
				{"foo": "foo val", "bar": "bar val"},
				{"foo": "foo val 2", "bar": "bar val 2"},
				{"foo": "foo val 3", "bar": "bar val 3"},
			},
		},
		{
			name: "slice of strings",
			query: `
				SELECT *
				FROM (
					VALUES ('foo val'), ('foo val 2'), ('foo val 3')
				) AS t (foo)
			`,
			expected: []string{"foo val", "foo val 2", "foo val 3"},
		},
		{
			name: "slice of strings by ptr",
			query: `
				SELECT *
				FROM (
					VALUES ('foo val'), (NULL), ('foo val 3')
				) AS t (foo)
			`,
			expected: []*string{makeStrPtr("foo val"), nil, makeStrPtr("foo val 3")},
		},
		{
			name: "slice of maps by ptr treated as primitive type case",
			query: `
				SELECT *
				FROM (
					VALUES ('{"key": "key val"}'::JSON), (NULL), ('{"key": "key val 3"}'::JSON)
				) AS t (foo_json)
			`,
			expected: []*map[string]interface{}{
				{"key": "key val"},
				nil,
				{"key": "key val 3"},
			},
		},
		{
			name: "slice of slices",
			query: `
				SELECT *
				FROM (
					VALUES (ARRAY('foo val', 'foo val 2')),
						(ARRAY('foo val 3', 'foo val 4')),
						(ARRAY('foo val 5', 'foo val 6'))
				) AS t (foo)
			`,
			expected: [][]string{
				{"foo val", "foo val 2"},
				{"foo val 3", "foo val 4"},
				{"foo val 5", "foo val 6"},
			},
		},
		{
			name: "slice of slices by ptr",
			query: `
				SELECT *
				FROM (
					VALUES (ARRAY('foo val', 'foo val 2')),
						(NULL),
						(ARRAY('foo val 5', 'foo val 6'))
				) AS t (foo)
			`,
			expected: []*[]string{
				{"foo val", "foo val 2"},
				nil,
				{"foo val 5", "foo val 6"},
			},
		},
	}
	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			rows := queryRows(t, tc.query)
			dst := allocateDestination(tc.expected)
			err := testAPI.ScanAll(dst, rows)
			require.NoError(t, err)
			assertDestinationEqual(t, tc.expected, dst)
		})
	}
}

func TestScanAll_nonEmptySlice_resetsDestinationSlice(t *testing.T) {
	t.Parallel()
	rows := queryRows(t, multipleRowsQuery)
	expected := []*testModel{
		{Foo: "foo val", Bar: "bar val"},
		{Foo: "foo val 2", Bar: "bar val 2"},
		{Foo: "foo val 3", Bar: "bar val 3"},
	}

	got := []*testModel{{Foo: "foo junk val", Bar: "bar junk val"}}
	err := testAPI.ScanAll(&got, rows)
	require.NoError(t, err)

	assert.Equal(t, expected, got)
}

func TestScanAll_nonSliceDestination_returnsErr(t *testing.T) {
	t.Parallel()
	rows := queryRows(t, multipleRowsQuery)
	dst := &testModel{}
	expectedErr := "scany: destination must be a slice, got: dbscan_test.testModel"

	err := testAPI.ScanAll(dst, rows)

	assert.EqualError(t, err, expectedErr)
}

func TestScanAll_sliceByPointerToPointerDestination_returnsErr(t *testing.T) {
	t.Parallel()
	rows := queryRows(t, multipleRowsQuery)
	dst := new(*[]testModel)
	expectedErr := "scany: destination must be a slice, got: *[]dbscan_test.testModel"

	err := testAPI.ScanAll(dst, rows)

	assert.EqualError(t, err, expectedErr)
}

func TestScanAll_closedRows(t *testing.T) {
	t.Parallel()
	rows := queryRows(t, multipleRowsQuery)
	for rows.Next() {
	}
	requireNoRowsErrorsAndClose(t, rows)

	var got []testModel
	err := testAPI.ScanAll(&got, rows)
	require.NoError(t, err)

	assert.Len(t, got, 0)
}

func TestScanOne(t *testing.T) {
	t.Parallel()
	rows := queryRows(t, singleRowsQuery)
	expected := testModel{Foo: "foo val", Bar: "bar val"}

	got := testModel{}
	err := testAPI.ScanOne(&got, rows)
	require.NoError(t, err)

	assert.Equal(t, expected, got)
}

func TestScanOne_zeroRows_returnsNotFoundErr(t *testing.T) {
	t.Parallel()
	query := `
		SELECT NULL AS foo LIMIT 0;
	`
	rows := queryRows(t, query)

	dst := &struct{ Foo string }{}
	err := testAPI.ScanOne(dst, rows)
	isNotFound := dbscan.NotFound(err)

	assert.True(t, isNotFound)
}

func TestScanOne_multipleRows_returnsErr(t *testing.T) {
	t.Parallel()
	rows := queryRows(t, multipleRowsQuery)
	expectedErr := "scany: expected 1 row, got: 3"

	dst := &testModel{}
	err := testAPI.ScanOne(dst, rows)

	assert.EqualError(t, err, expectedErr)
}

func TestScanRow(t *testing.T) {
	t.Parallel()
	rows := queryRows(t, singleRowsQuery)
	defer rows.Close() // nolint: errcheck
	rows.Next()
	expected := testModel{Foo: "foo val", Bar: "bar val"}

	var got testModel
	err := testAPI.ScanRow(&got, rows)
	require.NoError(t, err)
	requireNoRowsErrorsAndClose(t, rows)

	assert.Equal(t, expected, got)
}

func TestMain(m *testing.M) {
	exitCode := func() int {
		flag.Parse()
		ts, err := testserver.NewTestServer()
		if err != nil {
			panic(err)
		}
		defer ts.Stop()
		testDB, err = pgxpool.Connect(ctx, ts.PGURL().String())
		if err != nil {
			panic(err)
		}
		defer testDB.Close()
		testAPI = getAPI()
		return m.Run()
	}()
	os.Exit(exitCode)
}
