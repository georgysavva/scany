package pgxquery

import (
	"context"
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
	err = scanRows(dst, rows, false /* exactlyOneRow */)
	return errors.WithStack(err)
}

func QueryOne(ctx context.Context, q QueryI, dst interface{}, sql string, args ...interface{}) error {
	rows, err := q.Query(ctx, sql, args...)
	if err != nil {
		return errors.Wrap(err, "query rows")
	}
	err = scanRows(dst, rows, true /* exactlyOneRow */)
	return errors.WithStack(err)
}

func ScanRows(dst interface{}, rows pgx.Rows) error {
	err := scanRows(dst, rows, false /* exactlyOneRow */)
	return errors.WithStack(err)
}

// NotFound returns true if err is a not found error.
func NotFound(err error) bool {
	return errors.Is(err, notFoundErr)
}

var notFoundErr = errors.New("no row was found")

func scanRows(dst interface{}, rows pgx.Rows, exactlyOneRow bool) error {
	defer rows.Close()
	var rowsAffected int
	for rows.Next() {
		if err := rows.Scan(dst); err != nil {
			return errors.Wrap(err, "scan rows")
		}
		rowsAffected++
	}
	if err := rows.Err(); err != nil {
		return errors.Wrap(err, "rows final error")
	}
	if exactlyOneRow {
		if rowsAffected == 0 {
			return errors.WithStack(notFoundErr)
		} else if rowsAffected > 1 {
			return errors.Errorf("expected 1 row, got: %d", rowsAffected)
		}
	}
	return nil
}
