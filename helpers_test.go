package dbscan_test

import (
	"reflect"
	"testing"

	"github.com/georgysavva/dbscan/pgxscan"
	"github.com/stretchr/testify/require"

	"github.com/georgysavva/dbscan"

	"github.com/stretchr/testify/assert"
)

func makeStrPtr(v string) *string { return &v }

func queryRows(t *testing.T, query string) dbscan.Rows {
	t.Helper()
	pgxRows, err := testDB.Query(ctx, query)
	require.NoError(t, err)
	rows := pgxscan.RowsAdapter{pgxRows}
	return rows
}

func doScan(t *testing.T, dstValue reflect.Value, rows dbscan.Rows) error {
	rs := dbscan.NewRowScanner(rows)
	rows.Next()
	defer rows.Close()
	if err := rs.DoScan(dstValue); err != nil {
		return err
	}
	requireNoRowsErrors(t, rows)
	return nil
}

func requireNoRowsErrors(t *testing.T, rows dbscan.Rows) {
	t.Helper()
	require.NoError(t, rows.Err())
	require.NoError(t, rows.Close())
}

func newDstValue(v interface{}) reflect.Value {
	dstType := reflect.TypeOf(v)
	dstValue := reflect.New(dstType).Elem()
	return dstValue
}

func assertDstValueEqual(t *testing.T, expected interface{}, dstVal reflect.Value) {
	t.Helper()
	got := dstVal.Interface()
	assert.Equal(t, expected, got)
}
