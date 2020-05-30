package pgxquery_test

import (
	"reflect"
	"testing"

	"github.com/georgysavva/pgxquery"
	"github.com/jackc/pgproto3/v2"
	"github.com/jackc/pgx/v4"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
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

type ScanCase struct {
	name          string
	rows          *fakeRows
	expected      interface{}
	errString     string
	exactlyOneRow bool
}

func (tc *ScanCase) test(t *testing.T) {
	t.Helper()

	dstType := reflect.TypeOf(tc.expected)
	dstValue := reflect.New(dstType)
	dst := dstValue.Interface()

	var err error
	if tc.exactlyOneRow {
		err = pgxquery.ScanOne(dst, tc.rows)
	} else {
		err = pgxquery.ScanAll(dst, tc.rows)
	}

	if tc.errString == "" {
		require.NoError(t, err)
		got := dstValue.Elem().Interface()
		assert.Equal(t, tc.expected, got)
	} else {
		assert.EqualError(t, err, tc.errString)
	}
}
