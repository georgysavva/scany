package dbscan_test

import (
	"context"
	"flag"
	"os"
	"testing"

	"github.com/jackc/pgx/v4/pgxpool"
	_ "github.com/jackc/pgx/v4/stdlib"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/georgysavva/dbscan"
	"github.com/georgysavva/dbscan/internal/testutil"
)

var (
	testDB *pgxpool.Pool
	ctx    = context.Background()
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
			err := dbscan.ScanAll(dst, rows)
			require.NoError(t, err)
			assertDestinationEqual(t, tc.expected, dst)
		})
	}
}

func TestScanAll_nonEmptySlice_resetsDestinationSlice(t *testing.T) {
	t.Parallel()
	query := `
		SELECT *
		FROM (
			VALUES ('foo val'), ('foo val 2'), ('foo val 3')
		) AS t (foo)
	`
	rows := queryRows(t, query)
	expected := []string{"foo val", "foo val 2", "foo val 3"}

	got := []string{"junk data", "junk data 2"}
	err := dbscan.ScanAll(&got, rows)
	require.NoError(t, err)

	assert.Equal(t, expected, got)
}

func TestScanAll_nonSliceDestination_returnsErr(t *testing.T) {
	t.Parallel()
	query := `
		SELECT *
		FROM (
			VALUES ('foo val'), ('foo val 2'), ('foo val 3')
		) AS t (foo)
	`
	rows := queryRows(t, query)
	var dst struct {
		Foo string
	}
	expectedErr := "dbscan: destination must be a slice, got: struct { Foo string }"

	err := dbscan.ScanAll(&dst, rows)

	assert.EqualError(t, err, expectedErr)
}

func TestScanAll_sliceByPointerToPointerDestination_returnsErr(t *testing.T) {
	t.Parallel()
	query := `
		SELECT *
		FROM (
			VALUES ('foo val'), ('foo val 2'), ('foo val 3')
		) AS t (foo)
	`
	rows := queryRows(t, query)
	var dst *[]string
	expectedErr := "dbscan: destination must be a slice, got: *[]string"

	err := dbscan.ScanAll(&dst, rows)

	assert.EqualError(t, err, expectedErr)
}

func TestScanOne(t *testing.T) {
	t.Parallel()
	query := `
		SELECT 'foo val' AS foo, 'bar val' AS bar
	`
	rows := queryRows(t, query)
	type dst struct {
		Foo string
		Bar string
	}
	expected := dst{Foo: "foo val", Bar: "bar val"}

	got := dst{}
	err := dbscan.ScanOne(&got, rows)
	require.NoError(t, err)

	assert.Equal(t, expected, got)
}

func TestScanOne_zeroRows_returnsNotFoundErr(t *testing.T) {
	t.Parallel()
	query := `
		SELECT NULL AS foo, NULL AS bar LIMIT 0;
	`
	rows := queryRows(t, query)

	var dst string
	err := dbscan.ScanOne(&dst, rows)
	got := dbscan.NotFound(err)

	assert.True(t, got)
}

func TestScanOne_multipleRows_returnsErr(t *testing.T) {
	t.Parallel()
	query := `
		SELECT *
		FROM (
			VALUES ('foo val'), ('foo val 2'), ('foo val 3')
		) AS t (foo)
	`
	rows := queryRows(t, query)
	expectedErr := "dbscan: expected 1 row, got: 3"

	var dst string
	err := dbscan.ScanOne(&dst, rows)

	assert.EqualError(t, err, expectedErr)
}

func TestScanRow(t *testing.T) {
	t.Parallel()
	query := `
		SELECT 'foo val' AS foo, 'bar val' AS bar
	`
	rows := queryRows(t, query)
	defer rows.Close()
	type dst struct {
		Foo string
		Bar string
	}
	rows.Next()
	expected := dst{Foo: "foo val", Bar: "bar val"}

	var got dst
	err := dbscan.ScanRow(&got, rows)
	require.NoError(t, err)
	requireNoRowsErrors(t, rows)

	assert.Equal(t, expected, got)
}

func TestMain(m *testing.M) {
	exitCode := func() int {
		flag.Parse()
		ts, err := testutil.StartCrdbServer()
		if err != nil {
			panic(err)
		}
		defer ts.Stop()
		testDB, err = pgxpool.Connect(ctx, ts.PGURL().String())
		if err != nil {
			panic(err)
		}
		defer testDB.Close()
		return m.Run()
	}()
	os.Exit(exitCode)
}
