package pgxscan_test

import (
	"context"
	"flag"
	"os"
	"testing"

	"github.com/georgysavva/dbscan/pgxscan"
	"github.com/stretchr/testify/assert"

	"github.com/stretchr/testify/require"

	"github.com/georgysavva/dbscan/internal/testutil"
	"github.com/jackc/pgx/v4/pgxpool"
)

var (
	testDB *pgxpool.Pool
	ctx    = context.Background()
)

type testDestination struct {
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
	expected := []*testDestination{
		{Foo: "foo val", Bar: "bar val"},
		{Foo: "foo val 2", Bar: "bar val 2"},
		{Foo: "foo val 3", Bar: "bar val 3"},
	}

	var got []*testDestination
	err := pgxscan.QueryAll(ctx, &got, testDB, query)
	require.NoError(t, err)

	assert.Equal(t, expected, got)
}

func TestQueryOne(t *testing.T) {
	t.Parallel()
	query := `
		SELECT 'foo val' AS foo, 'bar val' AS bar
	`
	expected := testDestination{Foo: "foo val", Bar: "bar val"}

	var got testDestination
	err := pgxscan.QueryOne(ctx, &got, testDB, query)
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
	expected := []*testDestination{
		{Foo: "foo val", Bar: "bar val"},
		{Foo: "foo val 2", Bar: "bar val 2"},
		{Foo: "foo val 3", Bar: "bar val 3"},
	}
	rows, err := testDB.Query(ctx, query)
	require.NoError(t, err)

	var got []*testDestination
	err = pgxscan.ScanAll(&got, rows)
	require.NoError(t, err)

	assert.Equal(t, expected, got)
}

func TestScanOne(t *testing.T) {
	t.Parallel()
	query := `
		SELECT 'foo val' AS foo, 'bar val' AS bar
	`
	expected := testDestination{Foo: "foo val", Bar: "bar val"}
	rows, err := testDB.Query(ctx, query)
	require.NoError(t, err)

	var got testDestination
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

	var got testDestination
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
	type dst struct {
		Foo string
		Bar string
	}
	rs := pgxscan.NewRowScanner(rows)
	rows.Next()
	expected := dst{Foo: "foo val", Bar: "bar val"}

	var got dst
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
	type dst struct {
		Foo string
		Bar string
	}
	rows.Next()
	expected := dst{Foo: "foo val", Bar: "bar val"}

	var got dst
	err = pgxscan.ScanRow(&got, rows)
	require.NoError(t, err)
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
		testDB, err = pgxpool.Connect(ctx, ts.PGURL().String())
		if err != nil {
			panic(err)
		}
		defer testDB.Close()
		return m.Run()
	}()
	os.Exit(exitCode)
}
