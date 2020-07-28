package dbscan_test

import (
	"reflect"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/georgysavva/scany/dbscan"
	"github.com/georgysavva/scany/pgxscan"
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
)

func makeStrPtr(v string) *string { return &v }

func queryRows(t *testing.T, query string) dbscan.Rows {
	t.Helper()
	pgxRows, err := testDB.Query(ctx, query)
	require.NoError(t, err)
	rows := pgxscan.NewRowsAdapter(pgxRows)
	return rows
}

func scan(t *testing.T, dst interface{}, rows dbscan.Rows) error {
	defer rows.Close() // nolint: errcheck
	rs := dbscan.NewRowScanner(rows)
	rows.Next()
	if err := rs.Scan(dst); err != nil {
		return err
	}
	requireNoRowsErrorsAndClose(t, rows)
	return nil
}

func requireNoRowsErrorsAndClose(t *testing.T, rows dbscan.Rows) {
	t.Helper()
	require.NoError(t, rows.Err())
	require.NoError(t, rows.Close())
}

func allocateDestination(v interface{}) interface{} {
	dstType := reflect.TypeOf(v)
	dst := reflect.New(dstType).Interface()
	return dst
}

func assertDestinationEqual(t *testing.T, expected interface{}, dst interface{}) {
	t.Helper()
	got := reflect.ValueOf(dst).Elem().Interface()
	assert.Equal(t, expected, got)
}
