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
		return errors.Wrap(err, "query rows")
	}
	err = ScanAll(dst, rows)
	return errors.WithStack(err)
}

func QueryOne(ctx context.Context, q QueryI, dst interface{}, sqlText string, args ...interface{}) error {
	rows, err := q.QueryContext(ctx, sqlText, args...)
	if err != nil {
		return errors.Wrap(err, "query rows")
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

func ScanRow(dst interface{}, rows Rows) error {
	r := NewRowScanner(rows)
	err := r.Scan(dst)
	return errors.WithStack(err)
}

// NotFound returns true if err is a not found error.
func NotFound(err error) bool {
	return errors.Is(err, notFoundErr)
}

var notFoundErr = errors.New("no row was found")

func processRows(dst interface{}, rows Rows, multipleRows bool) error {
	defer rows.Close()
	dstValue, err := parseDestination(dst)
	if err != nil {
		return errors.WithStack(err)
	}
	var r *RowScanner
	if multipleRows {
		dstType := dstValue.Type()
		if dstValue.Kind() != reflect.Slice {
			return errors.Errorf(
				"destination must be a slice, got: %v", dstType,
			)
		}
		// Make sure that slice is empty.
		dstValue.Set(dstValue.Slice(0, 0))

		r = newRowScannerForSliceScan(rows, dstType)
	} else {
		r = NewRowScanner(rows)
	}
	var rowsAffected int
	for rows.Next() {
		var err error
		if multipleRows {
			err = r.scanSliceElement(dstValue)
		} else {
			err = r.doScan(dstValue)
		}
		if err != nil {
			return errors.WithStack(err)
		}
		rowsAffected++
	}

	if err := rows.Err(); err != nil {
		return errors.Wrap(err, "rows final error")
	}

	exactlyOneRow := !multipleRows
	if exactlyOneRow {
		if rowsAffected == 0 {
			return errors.WithStack(notFoundErr)
		} else if rowsAffected > 1 {
			return errors.Errorf("expected 1 row, got: %d", rowsAffected)
		}
	}
	return nil
}
