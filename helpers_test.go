package pgxquery_test

import (
	"reflect"
	"testing"

	"github.com/georgysavva/pgxquery"
	"github.com/jackc/pgproto3/v2"
	"github.com/jackc/pgx/v4"
	"github.com/stretchr/testify/assert"
)

func makePtr(v interface{}) interface{} {
	p := reflect.New(reflect.TypeOf(v))
	p.Elem().Set(reflect.ValueOf(v))
	return p.Interface()
}

func makeStrPtr(v string) *string { return &v }

func makeIntPtr(v int) *int { return &v }

type fakeRows struct {
	pgx.Rows
	columns       []string
	data          [][]interface{}
	currentRow    []interface{}
	rowsProcessed int
}

func (fr *fakeRows) Scan(dest ...interface{}) error {
	for i, data := range fr.currentRow {
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

func (fr *fakeRows) Next() bool {
	if fr.rowsProcessed >= len(fr.data) {
		return false
	}
	fr.currentRow = fr.data[fr.rowsProcessed]
	fr.rowsProcessed++
	return true
}

func (fr *fakeRows) FieldDescriptions() []pgproto3.FieldDescription {
	fields := make([]pgproto3.FieldDescription, len(fr.columns))
	for i, column := range fr.columns {
		fields[i] = pgproto3.FieldDescription{Name: []byte(column)}
	}
	return fields
}

func (fr *fakeRows) Values() ([]interface{}, error) { return fr.currentRow, nil }

func (fr *fakeRows) Close() {}

func (fr *fakeRows) Err() error { return nil }

func doScan(dstValue reflect.Value, rows pgx.Rows) error {
	r := pgxquery.WrapRows(rows)
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
