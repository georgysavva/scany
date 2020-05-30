package pgxquery

import (
	"context"
	"reflect"

	"github.com/jackc/pgx/v4"
	"github.com/pkg/errors"
)

type QueryI interface {
	Query(ctx context.Context, sql string, args ...interface{}) (pgx.Rows, error)
}

func QueryAll(ctx context.Context, q QueryI, dst interface{}, sql string, args ...interface{}) error {
	rows, err := q.Query(ctx, sql, args...)
	if err != nil {
		return errors.Wrap(err, "query rows")
	}
	err = ScanAll(dst, rows)
	return errors.WithStack(err)
}

func QueryOne(ctx context.Context, q QueryI, dst interface{}, sql string, args ...interface{}) error {
	rows, err := q.Query(ctx, sql, args...)
	if err != nil {
		return errors.Wrap(err, "query rows")
	}
	err = ScanOne(dst, rows)
	return errors.WithStack(err)
}

func ScanAll(dst interface{}, rows pgx.Rows) error {
	err := processRows(dst, rows, true /* multipleRows */)
	return errors.WithStack(err)
}

func ScanOne(dst interface{}, rows pgx.Rows) error {
	err := processRows(dst, rows, false /* multipleRows */)
	return errors.WithStack(err)
}

func ScanRow(dst interface{}, rows pgx.Rows) error {
	r := WrapRows(rows)
	err := r.Scanx(dst)
	return errors.WithStack(err)
}

// NotFound returns true if err is a not found error.
func NotFound(err error) bool {
	return errors.Is(err, notFoundErr)
}

var notFoundErr = errors.New("no row was found")

func processRows(dst interface{}, rows pgx.Rows, multipleRows bool) error {
	defer rows.Close()
	dstValue, err := parseDestination(dst)
	if err != nil {
		return errors.WithStack(err)
	}
	var r *Rows
	if multipleRows {
		dstType := dstValue.Type()
		if dstValue.Kind() != reflect.Slice {
			return errors.Errorf(
				"destination must be a slice, got: %v", dstType,
			)
		}
		// Make sure that slice is empty.
		dstValue.Set(dstValue.Slice(0, 0))

		r = wrapRowsForSliceScan(rows, dstType)
	} else {
		r = WrapRows(rows)
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
