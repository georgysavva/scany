package sqlscan

import (
	"context"
	"database/sql"
	"reflect"

	"github.com/pkg/errors"
)

type QueryI interface {
	QueryContext(ctx context.Context, sqlText string, args ...interface{}) (*sql.Rows, error)
}

var (
	_ QueryI = &sql.DB{}
	_ QueryI = &sql.Conn{}
	_ QueryI = &sql.Tx{}
)

type Rows interface {
	Close() error
	Err() error
	Next() bool
	Columns() ([]string, error)
	Scan(dest ...interface{}) error
}

func QueryAll(ctx context.Context, q QueryI, dst interface{}, sqlText string, args ...interface{}) error {
	rows, err := q.QueryContext(ctx, sqlText, args...)
	if err != nil {
		return errors.Wrap(err, "sqlscan: call query rows from querier")
	}
	err = ScanAll(dst, rows)
	return errors.WithStack(err)
}

func QueryOne(ctx context.Context, q QueryI, dst interface{}, sqlText string, args ...interface{}) error {
	rows, err := q.QueryContext(ctx, sqlText, args...)
	if err != nil {
		return errors.Wrap(err, "sqlscan: call query rows from querier")
	}
	err = ScanOne(dst, rows)
	return errors.WithStack(err)
}

func ScanAll(dst interface{}, rows Rows) error {
	err := processRows(dst, rows, true /* multipleRows */)
	return errors.WithStack(err)
}

func ScanOne(dst interface{}, rows Rows) error {
	err := processRows(dst, rows, false /* multipleRows */)
	return errors.WithStack(err)
}

// NotFound returns true if err is a not found error.
func NotFound(err error) bool {
	return errors.Is(err, notFoundErr)
}

var notFoundErr = errors.New("sqlscan: no row was found")

func processRows(dst interface{}, rows Rows, multipleRows bool) error {
	defer rows.Close()
	dstValue, err := parseDestination(dst)
	if err != nil {
		return errors.WithStack(err)
	}
	var rs *RowScanner
	if multipleRows {
		dstType := dstValue.Type()
		if dstValue.Kind() != reflect.Slice {
			return errors.Errorf(
				"sqlscan: destination must be a slice, got: %v", dstType,
			)
		}
		// Make sure that slice is empty.
		dstValue.Set(dstValue.Slice(0, 0))

		rs = newRowScannerForSliceScan(rows, dstType)
	} else {
		rs = NewRowScanner(rows)
	}
	var rowsAffected int
	for rows.Next() {
		var err error
		if multipleRows {
			err = rs.scanSliceElement(dstValue)
		} else {
			err = rs.doScan(dstValue)
		}
		if err != nil {
			return errors.WithStack(err)
		}
		rowsAffected++
	}

	if err := rows.Err(); err != nil {
		return errors.Wrap(err, "sqlscan: rows final error")
	}

	exactlyOneRow := !multipleRows
	if exactlyOneRow {
		if rowsAffected == 0 {
			return errors.WithStack(notFoundErr)
		} else if rowsAffected > 1 {
			return errors.Errorf("sqlscan: expected 1 row, got: %d", rowsAffected)
		}
	}
	return nil
}
