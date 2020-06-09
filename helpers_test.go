package dbscan_test

import (
	"reflect"
	"testing"

	"github.com/pkg/errors"

	"github.com/georgysavva/dbscan"

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
		dstVal := reflect.ValueOf(dst).Elem()
		if !dstVal.CanSet() {
			return errors.Errorf("testRows: can't set into dst: %v", dst)
		}
		if data != nil {
			dataVal := reflect.ValueOf(data)
			if !dataVal.Type().AssignableTo(dstVal.Type()) {
				return errors.Errorf(
					"testRows: can't assign value of type %v to dst of type %v",
					dataVal.Type(), dstVal.Type(),
				)
			}
			dstVal.Set(dataVal)
		} else {
			dstVal.Set(reflect.Zero(dstVal.Type()))
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

func doScan(dstValue reflect.Value, rows dbscan.Rows) error {
	rs := dbscan.NewRowScanner(rows)
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
