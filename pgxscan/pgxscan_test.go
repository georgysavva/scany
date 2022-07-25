package pgxscan_test

import (
	"context"
	stderrors "errors"
	"flag"
	"os"
	"testing"

	"github.com/cockroachdb/cockroach-go/v2/testserver"
	"github.com/jackc/pgtype"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/pkg/errors"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/georgysavva/scany/pgxscan"
)

var (
	testDB  *pgxpool.Pool
	ctx     = context.Background()
	testAPI *pgxscan.API
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
	rows, err := testDB.Query(ctx, multipleRowsQuery)
	require.NoError(t, err)

	var got []*testModel
	err = testAPI.ScanAll(&got, rows)
	require.NoError(t, err)

	assert.Equal(t, expected, got)
}

func TestScanOne(t *testing.T) {
	t.Parallel()
	expected := testModel{Foo: "foo val", Bar: "bar val"}
	rows, err := testDB.Query(ctx, singleRowsQuery)
	require.NoError(t, err)

	var got testModel
	err = testAPI.ScanOne(&got, rows)
	require.NoError(t, err)

	assert.Equal(t, expected, got)
}

func TestScanOne_noRows_returnsNotFoundErr(t *testing.T) {
	t.Parallel()
	rows, err := testDB.Query(ctx, noRowsQuery)
	require.NoError(t, err)

	var got testModel
	err = testAPI.ScanOne(&got, rows)

	assert.True(t, pgxscan.NotFound(err))
	assert.True(t, errors.Is(err, pgx.ErrNoRows))
	assert.True(t, stderrors.Is(err, pgx.ErrNoRows))
}

func TestRowScanner_Scan(t *testing.T) {
	t.Parallel()
	rows, err := testDB.Query(ctx, singleRowsQuery)
	require.NoError(t, err)
	defer rows.Close()
	rs := testAPI.NewRowScanner(rows)
	rows.Next()
	expected := testModel{Foo: "foo val", Bar: "bar val"}

	var got testModel
	err = rs.Scan(&got)
	require.NoError(t, err)
	require.NoError(t, rows.Err())

	assert.Equal(t, expected, got)
}

func TestRowScanner_Scan_NULLableScannerType(t *testing.T) {
	t.Parallel()
	type Destination struct {
		Foo pgtype.Text
	}
	for _, tc := range []struct {
		name     string
		query    string
		expected *Destination
	}{
		{
			name:     "NULL value",
			query:    `SELECT NULL as foo`,
			expected: &Destination{Foo: pgtype.Text{Status: pgtype.Null}},
		},
		{
			name:     "non NULL value",
			query:    `SELECT 'foo value' as foo`,
			expected: &Destination{Foo: pgtype.Text{String: "foo value", Status: pgtype.Present}},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			rows, err := testDB.Query(ctx, tc.query)
			require.NoError(t, err)
			defer rows.Close()
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

func TestScanRow(t *testing.T) {
	t.Parallel()
	rows, err := testDB.Query(ctx, singleRowsQuery)
	require.NoError(t, err)
	defer rows.Close()
	rows.Next()
	expected := testModel{Foo: "foo val", Bar: "bar val"}

	var got testModel
	err = testAPI.ScanRow(&got, rows)
	require.NoError(t, err)
	require.NoError(t, rows.Err())

	assert.Equal(t, expected, got)
}

func getAPI() (*pgxscan.API, error) {
	dbscanAPI, err := pgxscan.NewDBScanAPI()
	if err != nil {
		return nil, errors.WithStack(err)
	}
	api, err := pgxscan.NewAPI(dbscanAPI)
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
		testDB, err = pgxpool.New(ctx, ts.PGURL().String())
		if err != nil {
			panic(err)
		}
		defer testDB.Close()
		testAPI, err = getAPI()
		if err != nil {
			panic(err)
		}
		return m.Run()
	}()
	os.Exit(exitCode)
}
