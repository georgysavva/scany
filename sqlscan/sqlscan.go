package sqlscan

import (
	"context"
	"database/sql"
	"github.com/georgysavva/dbscan"
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

func QueryAll(ctx context.Context, q QueryI, dst interface{}, sqlText string, args ...interface{}) error {
	rows, err := q.QueryContext(ctx, sqlText, args...)
	if err != nil {
		return errors.Wrap(err, "sqlscan: query result rows")
	}
	err = ScanAll(dst, rows)
	return errors.WithStack(err)
}

func QueryOne(ctx context.Context, q QueryI, dst interface{}, sqlText string, args ...interface{}) error {
	rows, err := q.QueryContext(ctx, sqlText, args...)
	if err != nil {
		return errors.Wrap(err, "sqlscan: query result rows")
	}
	err = ScanOne(dst, rows)
	return errors.WithStack(err)
}

func ScanAll(dst interface{}, rows *sql.Rows) error {
	err := dbscan.ScanAll(dst, rows)
	return errors.WithStack(err)
}

func ScanOne(dst interface{}, rows *sql.Rows) error {
	err := dbscan.ScanOne(dst, rows)
	return errors.WithStack(err)
}

// NotFound returns true if err is a not found error.
func NotFound(err error) bool {
	return dbscan.NotFound(err)
}

type RowScanner struct {
	*dbscan.RowScanner
}

func NewRowScanner(rows *sql.Rows) *RowScanner {
	return &RowScanner{RowScanner: dbscan.NewRowScanner(rows)}
}

func ScanRow(dst interface{}, rows *sql.Rows) error {
	rs := NewRowScanner(rows)
	err := rs.Scan(dst)
	return errors.WithStack(err)
}
