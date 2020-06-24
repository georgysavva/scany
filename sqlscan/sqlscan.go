package sqlscan

import (
	"context"
	"database/sql"

	"github.com/pkg/errors"

	"github.com/georgysavva/dbscan"
)

// QueryI is something that sqlscan can query and get the *sql.Rows,
// For example: *sql.DB, *sql.Conn or *sql.Tx.
type QueryI interface {
	QueryContext(ctx context.Context, query string, args ...interface{}) (*sql.Rows, error)
}

var (
	_ QueryI = &sql.DB{}
	_ QueryI = &sql.Conn{}
	_ QueryI = &sql.Tx{}
)

// QueryAll is a high-level function that queries the rows and calls the ScanAll function.
// See ScanAll for details.
func QueryAll(ctx context.Context, dst interface{}, q QueryI, query string, args ...interface{}) error {
	rows, err := q.QueryContext(ctx, query, args...)
	if err != nil {
		return errors.Wrap(err, "sqlscan: query result rows")
	}
	err = ScanAll(dst, rows)
	return errors.WithStack(err)
}

// QueryOne is a high-level function that queries the rows and calls the ScanOne function.
// See ScanOne for details.
func QueryOne(ctx context.Context, dst interface{}, q QueryI, query string, args ...interface{}) error {
	rows, err := q.QueryContext(ctx, query, args...)
	if err != nil {
		return errors.Wrap(err, "sqlscan: query result rows")
	}
	err = ScanOne(dst, rows)
	return errors.WithStack(err)
}

// ScanAll is a wrapper around the dbscan.ScanAll function.
// Se dbscan.ScanAll for details.
func ScanAll(dst interface{}, rows *sql.Rows) error {
	err := dbscan.ScanAll(dst, rows)
	return errors.WithStack(err)
}

// ScanOne is a wrapper around the dbscan.ScanOne function.
// Se dbscan.ScanOne for details.
func ScanOne(dst interface{}, rows *sql.Rows) error {
	err := dbscan.ScanOne(dst, rows)
	return errors.WithStack(err)
}

// NotFound is a wrapper around the dbscan.NotFound function.
// Se dbscan.NotFound for details.
func NotFound(err error) bool {
	return dbscan.NotFound(err)
}

// RowScanner is a wrapper around the dbscan.RowScanner type.
type RowScanner struct {
	*dbscan.RowScanner
}

// NewRowScanner returns a new RowScanner instance.
func NewRowScanner(rows *sql.Rows) *RowScanner {
	return &RowScanner{RowScanner: dbscan.NewRowScanner(rows)}
}

// ScanRow is a wrapper around the dbscan.ScanRow function.
// See dbscan.ScanRow for details.
func ScanRow(dst interface{}, rows *sql.Rows) error {
	err := dbscan.ScanRow(dst, rows)
	return errors.WithStack(err)
}
