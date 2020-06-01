package pgxscan_test

import (
	"context"
	"reflect"
	"testing"

	"github.com/georgysavva/pgxscan"
	"github.com/jackc/pgproto3/v2"
	"github.com/jackc/pgx/v4"
	"github.com/stretchr/testify/assert"
)

func makeStrPtr(v string) *string { return &v }

type testQuerier struct {
	rows pgx.Rows
}

func (tq *testQuerier) Query(ctx context.Context, sql string, args ...interface{}) (pgx.Rows, error) {
	return tq.rows, nil
}

type testRows struct {
	pgx.Rows
	columns       []string
	data          [][]interface{}
	currentRow    []interface{}
	rowsProcessed int
}

func (tr *testRows) Scan(dest ...interface{}) error {
	for i, data := range tr.currentRow {
		dst := dest[i]
		dstV := reflect.ValueOf(dst).Elem()
		if data != nil {
			dstV.Set(reflect.ValueOf(data))
		} else {
			dstV.Set(reflect.Zero(dstV.Type()))
		}
	}
	return nil
}

func (tr *testRows) Next() bool {
	if tr.rowsProcessed >= len(tr.data) {
		return false
	}
	tr.currentRow = tr.data[tr.rowsProcessed]
	tr.rowsProcessed++
	return true
}

func (tr *testRows) FieldDescriptions() []pgproto3.FieldDescription {
	fields := make([]pgproto3.FieldDescription, len(tr.columns))
	for i, column := range tr.columns {
		fields[i] = pgproto3.FieldDescription{Name: []byte(column)}
	}
	return fields
}

func (tr *testRows) Values() ([]interface{}, error) { return tr.currentRow, nil }

func (tr *testRows) Close() {}

func (tr *testRows) Err() error { return nil }

func doScan(dstValue reflect.Value, rows pgx.Rows) error {
	r := pgxscan.WrapRows(rows)
	rows.Next()
	return r.DoScan(dstValue)
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
