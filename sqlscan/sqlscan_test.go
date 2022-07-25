package sqlscan_test

import (
	"context"
	"database/sql"
	stderrors "errors"
	"flag"
	"os"
	"testing"

	"github.com/cockroachdb/cockroach-go/v2/testserver"
	_ "github.com/jackc/pgx/v5/stdlib"
	"github.com/pkg/errors"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/georgysavva/scany/sqlscan"
)

var (
	testDB  *sql.DB
	ctx     = context.Background()
	testAPI *sqlscan.API
)

type testModel struct {
	Foo string
	Bar string
}

const (
	multipleRowsQuery = `
		SELECT *
		FROM (
			VALUES ('foo val', 'bar val'), ('foo val 2', 'bar val 2'), ('foo val 3', 'bar val 3')
		) AS t (foo, bar)
	`
	singleRowsQuery = `
		SELECT 'foo val' AS foo, 'bar val' AS bar
	`
	noRowsQuery = `
		SELECT NULL AS foo, NULL AS bar LIMIT 0;
    `
)

func TestSelect(t *testing.T) {
	t.Parallel()
	expected := []*testModel{
		{Foo: "foo val", Bar: "bar val"},
		{Foo: "foo val 2", Bar: "bar val 2"},
		{Foo: "foo val 3", Bar: "bar val 3"},
	}

	var got []*testModel
	err := testAPI.Select(ctx, testDB, &got, multipleRowsQuery)
	require.NoError(t, err)

	assert.Equal(t, expected, got)
}

func TestSelect_queryError_propagatesAndWrapsErr(t *testing.T) {
	t.Parallel()
	query := `
		SELECT foo, bar, baz
		FROM (
			VALUES ('foo val', 'bar val'), ('foo val 2', 'bar val 2'), ('foo val 3', 'bar val 3')
		) AS t (foo, bar)
	`
	expectedErr := "scany: query multiple result rows: ERROR: column \"baz\" does not exist (SQLSTATE 42703)"

	dst := &[]*testModel{}
	err := testAPI.Select(ctx, testDB, dst, query)

	assert.EqualError(t, err, expectedErr)
}

func TestGet(t *testing.T) {
	t.Parallel()
	expected := testModel{Foo: "foo val", Bar: "bar val"}

	var got testModel
	err := testAPI.Get(ctx, testDB, &got, singleRowsQuery)
	require.NoError(t, err)

	assert.Equal(t, expected, got)
}

func TestGet_queryError_propagatesAndWrapsErr(t *testing.T) {
	t.Parallel()
	query := `
		SELECT 'foo val' AS foo, 'bar val' AS bar, baz
	`
	expectedErr := "scany: query one result row: ERROR: column \"baz\" does not exist (SQLSTATE 42703)"

	dst := &testModel{}
	err := testAPI.Get(ctx, testDB, dst, query)

	assert.EqualError(t, err, expectedErr)
}

func TestScanAll(t *testing.T) {
	t.Parallel()
	expected := []*testModel{
		{Foo: "foo val", Bar: "bar val"},
		{Foo: "foo val 2", Bar: "bar val 2"},
		{Foo: "foo val 3", Bar: "bar val 3"},
	}
	rows, err := testDB.Query(multipleRowsQuery)
	require.NoError(t, err)

	var got []*testModel
	err = testAPI.ScanAll(&got, rows)
	require.NoError(t, err)

	assert.Equal(t, expected, got)
}

func TestScanOne(t *testing.T) {
	t.Parallel()
	expected := testModel{Foo: "foo val", Bar: "bar val"}
	rows, err := testDB.Query(singleRowsQuery)
	require.NoError(t, err)

	var got testModel
	err = testAPI.ScanOne(&got, rows)
	require.NoError(t, err)

	assert.Equal(t, expected, got)
}

func TestScanOne_noRows_returnsNotFoundErr(t *testing.T) {
	t.Parallel()
	rows, err := testDB.Query(noRowsQuery)
	require.NoError(t, err)

	var got testModel
	err = testAPI.ScanOne(&got, rows)

	assert.True(t, sqlscan.NotFound(err))
	assert.True(t, errors.Is(err, sql.ErrNoRows))
	assert.True(t, stderrors.Is(err, sql.ErrNoRows))
}

func TestRowScanner_Scan(t *testing.T) {
	t.Parallel()
	rows, err := testDB.Query(singleRowsQuery)
	require.NoError(t, err)
	defer rows.Close() // nolint: errcheck
	rs := testAPI.NewRowScanner(rows)
	rows.Next()
	expected := testModel{Foo: "foo val", Bar: "bar val"}

	var got testModel
	err = rs.Scan(&got)
	require.NoError(t, err)
	requireNoRowsErrorsAndClose(t, rows)

	assert.Equal(t, expected, got)
}

func TestScanRow(t *testing.T) {
	t.Parallel()
	rows, err := testDB.Query(singleRowsQuery)
	require.NoError(t, err)
	defer rows.Close() // nolint: errcheck
	rows.Next()
	expected := testModel{Foo: "foo val", Bar: "bar val"}

	var got testModel
	err = testAPI.ScanRow(&got, rows)
	require.NoError(t, err)
	requireNoRowsErrorsAndClose(t, rows)

	assert.Equal(t, expected, got)
}

func TestRowScanner_Scan_closedRows(t *testing.T) {
	t.Parallel()
	rows, err := testDB.Query(multipleRowsQuery)
	require.NoError(t, err)
	for rows.Next() {
	}
	requireNoRowsErrorsAndClose(t, rows)

	rs := testAPI.NewRowScanner(rows)
	dst := &testModel{}
	err = rs.Scan(dst)

	assert.EqualError(t, err, "scany: get rows columns: sql: Rows are closed")
}

type ScannableString struct {
	string
}

func (ss *ScannableString) Scan(src interface{}) error {
	ss.string = src.(string)
	return nil
}

func TestRowScanner_Scan_NULLableScannerType(t *testing.T) {
	t.Parallel()
	type Destination struct {
		FooByPtr *ScannableString
		FooByVal ScannableString
	}
	for _, tc := range []struct {
		name     string
		query    string
		expected *Destination
	}{
		{
			name:  "NULL value",
			query: `SELECT NULL as foo_by_ptr, NULL as foo_by_val`,
			expected: &Destination{
				FooByPtr: nil,
				FooByVal: ScannableString{""},
			},
		},
		{
			name:  "non NULL value",
			query: `SELECT 'foo value 1' as foo_by_ptr, 'foo value 2' as foo_by_val`,
			expected: &Destination{
				FooByPtr: &ScannableString{"foo value 1"},
				FooByVal: ScannableString{"foo value 2"},
			},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			rows, err := testDB.Query(tc.query)
			require.NoError(t, err)
			defer rows.Close() // nolint: errcheck
			rs := testAPI.NewRowScanner(rows)
			rows.Next()

			got := &Destination{}
			err = rs.Scan(got)
			require.NoError(t, err)
			require.NoError(t, rows.Err())

			assert.Equal(t, tc.expected, got)
		})
	}
}

func requireNoRowsErrorsAndClose(t *testing.T, rows *sql.Rows) {
	t.Helper()
	require.NoError(t, rows.Err())
	require.NoError(t, rows.Close())
}

func getAPI() (*sqlscan.API, error) {
	dbscanAPI, err := sqlscan.NewDBScanAPI()
	if err != nil {
		return nil, errors.WithStack(err)
	}
	api, err := sqlscan.NewAPI(dbscanAPI)
	if err != nil {
		return nil, errors.WithStack(err)
	}
	return api, nil
}

func TestMain(m *testing.M) {
	exitCode := func() int {
		flag.Parse()
		ts, err := testserver.NewTestServer()
		if err != nil {
			panic(err)
		}
		defer ts.Stop()
		testDB, err = sql.Open("pgx", ts.PGURL().String())
		if err != nil {
			panic(err)
		}
		defer func() {
			if closeErr := testDB.Close(); closeErr != nil {
				panic(closeErr)
			}
		}()
		testAPI, err = getAPI()
		if err != nil {
			panic(err)
		}
		return m.Run()
	}()
	os.Exit(exitCode)
}
