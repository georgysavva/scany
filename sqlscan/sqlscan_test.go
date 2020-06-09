package sqlscan_test

import (
	"context"
	"flag"
	"os"

	"github.com/georgysavva/dbscan/internal/testutil"

	"database/sql"
	"testing"

	"github.com/georgysavva/dbscan/sqlscan"
	_ "github.com/jackc/pgx/v4/stdlib"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var (
	testDB *sql.DB
	ctx    = context.Background()
)

type testDst struct {
	Foo string
	Bar string
}

func TestQueryAll(t *testing.T) {
	t.Parallel()
	sqlText := `
		SELECT *
		FROM (
			VALUES ('foo val', 'bar val'), ('foo val 2', 'bar val 2'), ('foo val 3', 'bar val 3')
		) AS t (foo, bar)
	`
	expected := []*testDst{
		{Foo: "foo val", Bar: "bar val"},
		{Foo: "foo val 2", Bar: "bar val 2"},
		{Foo: "foo val 3", Bar: "bar val 3"},
	}

	var got []*testDst
	err := sqlscan.QueryAll(ctx, testDB, &got, sqlText)
	require.NoError(t, err)

	assert.Equal(t, expected, got)
}

func TestQueryOne(t *testing.T) {
	t.Parallel()
	sqlText := `
		SELECT 'foo val' AS foo, 'bar val' AS bar
	`
	expected := testDst{Foo: "foo val", Bar: "bar val"}

	var got testDst
	err := sqlscan.QueryOne(ctx, testDB, &got, sqlText)
	require.NoError(t, err)

	assert.Equal(t, expected, got)
}

func TestScanAll(t *testing.T) {
	t.Parallel()
	sqlText := `
		SELECT *
		FROM (
			VALUES ('foo val', 'bar val'), ('foo val 2', 'bar val 2'), ('foo val 3', 'bar val 3')
		) AS t (foo, bar)
	`
	expected := []*testDst{
		{Foo: "foo val", Bar: "bar val"},
		{Foo: "foo val 2", Bar: "bar val 2"},
		{Foo: "foo val 3", Bar: "bar val 3"},
	}
	rows, err := testDB.Query(sqlText)
	require.NoError(t, err)

	var got []*testDst
	err = sqlscan.ScanAll(&got, rows)
	require.NoError(t, err)

	assert.Equal(t, expected, got)
}

func TestScanOne(t *testing.T) {
	t.Parallel()
	sqlText := `
		SELECT 'foo val' AS foo, 'bar val' AS bar
	`
	expected := testDst{Foo: "foo val", Bar: "bar val"}
	rows, err := testDB.Query(sqlText)
	require.NoError(t, err)

	var got testDst
	err = sqlscan.ScanOne(&got, rows)
	require.NoError(t, err)

	assert.Equal(t, expected, got)
}

func TestScanOne_NoRows_ReturnsNotFoundErr(t *testing.T) {
	t.Parallel()
	sqlText := `
		SELECT NULL AS foo, NULL AS bar LIMIT 0;
	`
	rows, err := testDB.Query(sqlText)
	require.NoError(t, err)

	var got testDst
	err = sqlscan.ScanOne(&got, rows)

	assert.True(t, sqlscan.NotFound(err))
}

func TestScanRow(t *testing.T) {
	t.Parallel()
	sqlText := `
		SELECT *
		FROM (
			VALUES ('foo val', 'bar val'), ('foo val 2', 'bar val 2'), ('foo val 3', 'bar val 3')
		) AS t (foo, bar)
	`
	expected := []*testDst{
		{Foo: "foo val", Bar: "bar val"},
		{Foo: "foo val 2", Bar: "bar val 2"},
		{Foo: "foo val 3", Bar: "bar val 3"},
	}

	var got []*testDst
	rows, err := testDB.Query(sqlText)
	require.NoError(t, err)
	defer rows.Close()
	for rows.Next() {
		dst := &testDst{}
		err := sqlscan.ScanRow(dst, rows)
		require.NoError(t, err)
		got = append(got, dst)
	}
	require.NoError(t, rows.Err())

	assert.Equal(t, expected, got)
}

func TestRowScanner(t *testing.T) {
	t.Parallel()
	sqlText := `
		SELECT *
		FROM (
			VALUES ('foo val', 'bar val'), ('foo val 2', 'bar val 2'), ('foo val 3', 'bar val 3')
		) AS t (foo, bar)
	`
	expected := []*testDst{
		{Foo: "foo val", Bar: "bar val"},
		{Foo: "foo val 2", Bar: "bar val 2"},
		{Foo: "foo val 3", Bar: "bar val 3"},
	}

	var got []*testDst
	rows, err := testDB.Query(sqlText)
	require.NoError(t, err)
	defer rows.Close()
	rs := sqlscan.NewRowScanner(rows)
	for rows.Next() {
		dst := &testDst{}
		err := rs.Scan(dst)
		require.NoError(t, err)
		got = append(got, dst)
	}
	require.NoError(t, rows.Err())

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
		testDB, err = sql.Open("pgx", ts.PGURL().String())
		if err != nil {
			panic(err)
		}
		defer testDB.Close()
		return m.Run()
	}()
	os.Exit(exitCode)
}
