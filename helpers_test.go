package sqlscan_test

import (
	"reflect"
	"testing"

	"github.com/georgysavva/sqlscan"

	"github.com/stretchr/testify/assert"
)

func makeStrPtr(v string) *string { return &v }

type testRows struct {
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

func (tr *testRows) Columns() ([]string, error) {
	return tr.columns, nil
}

func (tr *testRows) Close() error { return nil }

func (tr *testRows) Err() error { return nil }

func doScan(dstValue reflect.Value, rows sqlscan.Rows) error {
	rs := sqlscan.NewRowScanner(rows)
	rows.Next()
	return rs.DoScan(dstValue)
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
