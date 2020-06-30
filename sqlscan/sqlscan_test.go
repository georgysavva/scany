package sqlscan_test

import (
	"context"
	"database/sql"
	"flag"
	"os"
	"testing"

	"github.com/cockroachdb/cockroach-go/v2/testserver"
	_ "github.com/jackc/pgx/v4/stdlib"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/georgysavva/dbscan/sqlscan"
)

var (
	testDB *sql.DB
	ctx    = context.Background()
)

type testModel struct {
	Foo string
	Bar string
}

func TestQueryAll(t *testing.T) {
	t.Parallel()
	query := `
		SELECT *
		FROM (
			VALUES ('foo val', 'bar val'), ('foo val 2', 'bar val 2'), ('foo val 3', 'bar val 3')
		) AS t (foo, bar)
	`
	expected := []*testModel{
		{Foo: "foo val", Bar: "bar val"},
		{Foo: "foo val 2", Bar: "bar val 2"},
		{Foo: "foo val 3", Bar: "bar val 3"},
	}

	var got []*testModel
	err := sqlscan.QueryAll(ctx, &got, testDB, query)
	require.NoError(t, err)

	assert.Equal(t, expected, got)
}

func TestQueryOne(t *testing.T) {
	t.Parallel()
	query := `
		SELECT 'foo val' AS foo, 'bar val' AS bar
	`
	expected := testModel{Foo: "foo val", Bar: "bar val"}

	var got testModel
	err := sqlscan.QueryOne(ctx, &got, testDB, query)
	require.NoError(t, err)

	assert.Equal(t, expected, got)
}

func TestScanAll(t *testing.T) {
	t.Parallel()
	query := `
		SELECT *
		FROM (
			VALUES ('foo val', 'bar val'), ('foo val 2', 'bar val 2'), ('foo val 3', 'bar val 3')
		) AS t (foo, bar)
	`
	expected := []*testModel{
		{Foo: "foo val", Bar: "bar val"},
		{Foo: "foo val 2", Bar: "bar val 2"},
		{Foo: "foo val 3", Bar: "bar val 3"},
	}
	rows, err := testDB.Query(query)
	require.NoError(t, err)

	var got []*testModel
	err = sqlscan.ScanAll(&got, rows)
	require.NoError(t, err)

	assert.Equal(t, expected, got)
}

func TestScanOne(t *testing.T) {
	t.Parallel()
	query := `
		SELECT 'foo val' AS foo, 'bar val' AS bar
	`
	expected := testModel{Foo: "foo val", Bar: "bar val"}
	rows, err := testDB.Query(query)
	require.NoError(t, err)

	var got testModel
	err = sqlscan.ScanOne(&got, rows)
	require.NoError(t, err)

	assert.Equal(t, expected, got)
}

func TestScanOne_noRows_returnsNotFoundErr(t *testing.T) {
	t.Parallel()
	query := `
		SELECT NULL AS foo, NULL AS bar LIMIT 0;
	`
	rows, err := testDB.Query(query)
	require.NoError(t, err)

	var got testModel
	err = sqlscan.ScanOne(&got, rows)

	assert.True(t, sqlscan.NotFound(err))
}

func TestRowScanner_Scan(t *testing.T) {
	t.Parallel()
	query := `
		SELECT 'foo val' AS foo, 'bar val' AS bar
	`
	rows, err := testDB.Query(query)
	require.NoError(t, err)
	defer rows.Close()
	rs := sqlscan.NewRowScanner(rows)
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
	query := `
		SELECT 'foo val' AS foo, 'bar val' AS bar
	`
	rows, err := testDB.Query(query)
	require.NoError(t, err)
	defer rows.Close()
	rows.Next()
	expected := testModel{Foo: "foo val", Bar: "bar val"}

	var got testModel
	err = sqlscan.ScanRow(&got, rows)
	require.NoError(t, err)
	requireNoRowsErrorsAndClose(t, rows)

	assert.Equal(t, expected, got)
}

func TestRowScanner_Scan_closedRows(t *testing.T) {
	t.Parallel()
	query := `
		SELECT *
		FROM (
			VALUES ('foo val', 'bar val'), ('foo val 2', 'bar val 2'), ('foo val 3', 'bar val 3')
		) AS t (foo, bar)
	`
	rows, err := testDB.Query(query)
	require.NoError(t, err)
	for rows.Next() {
	}
	requireNoRowsErrorsAndClose(t, rows)

	rs := sqlscan.NewRowScanner(rows)
	dst := &testModel{}
	err = rs.Scan(dst)

	assert.EqualError(t, err,
		"sqlscan: proxy call to dbscan.RowScanner.Scan: "+
			"dbscan: get rows columns: sql: Rows are closed",
	)
}

func requireNoRowsErrorsAndClose(t *testing.T, rows *sql.Rows) {
	t.Helper()
	require.NoError(t, rows.Err())
	require.NoError(t, rows.Close())
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
		defer testDB.Close()
		return m.Run()
	}()
	os.Exit(exitCode)
}
