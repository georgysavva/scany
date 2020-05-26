package pgxquery

import (
	"context"
	"github.com/georgysavva/pgxquery/internal/reflection"

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
	err = Scan(dst, rows)
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

func Scan(dst interface{}, rows pgx.Rows) error {
	err := scanRows(dst, rows, false /* exactlyOneRow */)
	return errors.WithStack(err)
}

func ScanOne(dst interface{}, rows pgx.Rows) error {
	err := scanRows(dst, rows, true /* exactlyOneRow */)
	return errors.WithStack(err)
}

// NotFound returns true if err is a not found error.
func NotFound(err error) bool {
	return errors.Is(err, notFoundErr)
}

var notFoundErr = errors.New("no row was found")

func scanRows(dst interface{}, rows pgx.Rows, exactlyOneRow bool) error {
	defer rows.Close()
	dstRef, err := reflection.ParseDst(dst, exactlyOneRow)
	if err != nil {
		return errors.WithStack(err)
	}
	scanner := &reflection.RowsWrapper{rows}

	var rowsAffected int
	for rows.Next() {
		if err := dstRef.Fill(scanner); err != nil {
			return errors.WithStack(err)
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
