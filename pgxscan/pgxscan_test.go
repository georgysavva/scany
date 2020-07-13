package pgxscan_test

import (
	"context"
	"flag"
	"os"
	"testing"

	"github.com/cockroachdb/cockroach-go/v2/testserver"
	"github.com/jackc/pgx/v4/pgxpool"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/georgysavva/scany/pgxscan"
)

var (
	testDB *pgxpool.Pool
	ctx    = context.Background()
)

type testModel struct {
	Foo string
	Bar string
}

func TestSelect(t *testing.T) {
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
	err := pgxscan.Select(ctx, testDB, &got, query)
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
	err := pgxscan.Select(ctx, testDB, dst, query)

	assert.EqualError(t, err, expectedErr)
}

func TestGet(t *testing.T) {
	t.Parallel()
	query := `
		SELECT 'foo val' AS foo, 'bar val' AS bar
	`
	expected := testModel{Foo: "foo val", Bar: "bar val"}

	var got testModel
	err := pgxscan.Get(ctx, testDB, &got, query)
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
	err := pgxscan.Get(ctx, testDB, dst, query)

	assert.EqualError(t, err, expectedErr)
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
	rows, err := testDB.Query(ctx, query)
	require.NoError(t, err)

	var got []*testModel
	err = pgxscan.ScanAll(&got, rows)
	require.NoError(t, err)

	assert.Equal(t, expected, got)
}

func TestScanOne(t *testing.T) {
	t.Parallel()
	query := `
		SELECT 'foo val' AS foo, 'bar val' AS bar
	`
	expected := testModel{Foo: "foo val", Bar: "bar val"}
	rows, err := testDB.Query(ctx, query)
	require.NoError(t, err)

	var got testModel
	err = pgxscan.ScanOne(&got, rows)
	require.NoError(t, err)

	assert.Equal(t, expected, got)
}

func TestScanOne_noRows_returnsNotFoundErr(t *testing.T) {
	t.Parallel()
	query := `
		SELECT NULL AS foo, NULL AS bar LIMIT 0;
	`
	rows, err := testDB.Query(ctx, query)
	require.NoError(t, err)

	var got testModel
	err = pgxscan.ScanOne(&got, rows)

	assert.True(t, pgxscan.NotFound(err))
}

func TestRowScanner_Scan(t *testing.T) {
	t.Parallel()
	query := `
		SELECT 'foo val' AS foo, 'bar val' AS bar
	`
	rows, err := testDB.Query(ctx, query)
	require.NoError(t, err)
	defer rows.Close()
	rs := pgxscan.NewRowScanner(rows)
	rows.Next()
	expected := testModel{Foo: "foo val", Bar: "bar val"}

	var got testModel
	err = rs.Scan(&got)
	require.NoError(t, err)
	require.NoError(t, rows.Err())

	assert.Equal(t, expected, got)
}

func TestScanRow(t *testing.T) {
	t.Parallel()
	query := `
		SELECT 'foo val' AS foo, 'bar val' AS bar
	`
	rows, err := testDB.Query(ctx, query)
	require.NoError(t, err)
	defer rows.Close()
	rows.Next()
	expected := testModel{Foo: "foo val", Bar: "bar val"}

	var got testModel
	err = pgxscan.ScanRow(&got, rows)
	require.NoError(t, err)
	require.NoError(t, rows.Err())

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
		return m.Run()
	}()
	os.Exit(exitCode)
}
