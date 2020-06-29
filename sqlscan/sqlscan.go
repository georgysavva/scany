package sqlscan

import (
	"context"
	"database/sql"

	"github.com/pkg/errors"

	"github.com/georgysavva/dbscan"
)

// Querier is something that sqlscan can query and get the *sql.Rows from.
// For example, it can be: *sql.DB, *sql.Conn or *sql.Tx.
type Querier interface {
	QueryContext(ctx context.Context, query string, args ...interface{}) (*sql.Rows, error)
}

var (
	_ Querier = &sql.DB{}
	_ Querier = &sql.Conn{}
	_ Querier = &sql.Tx{}
)

// QueryAll is a high-level function that queries the rows and calls the ScanAll function.
// See ScanAll for details.
func QueryAll(ctx context.Context, dst interface{}, q Querier, query string, args ...interface{}) error {
	rows, err := q.QueryContext(ctx, query, args...)
	if err != nil {
		return errors.Wrap(err, "sqlscan: query result rows")
	}
	err = ScanAll(dst, rows)
	return errors.WithStack(err)
}

// QueryOne is a high-level function that queries the rows and calls the ScanOne function.
// See ScanOne for details.
func QueryOne(ctx context.Context, dst interface{}, q Querier, query string, args ...interface{}) error {
	rows, err := q.QueryContext(ctx, query, args...)
	if err != nil {
		return errors.Wrap(err, "sqlscan: query result rows")
	}
	err = ScanOne(dst, rows)
	return errors.WithStack(err)
}

// ScanAll is a wrapper around the dbscan.ScanAll function.
// See dbscan.ScanAll for details.
func ScanAll(dst interface{}, rows *sql.Rows) error {
	err := dbscan.ScanAll(dst, rows)
	return errors.Wrap(err, "sqlscan: proxy call to dbscan.ScanAll")
}

// ScanOne is a wrapper around the dbscan.ScanOne function.
// See dbscan.ScanOne for details.
func ScanOne(dst interface{}, rows *sql.Rows) error {
	err := dbscan.ScanOne(dst, rows)
	return errors.Wrap(err, "sqlscan: proxy call to dbscan.ScanOne")
}

// NotFound is a wrapper around the dbscan.NotFound function.
// See dbscan.NotFound for details.
func NotFound(err error) bool {
	return dbscan.NotFound(err)
}

// RowScanner is a wrapper around the dbscan.RowScanner type.
// See dbscan.RowScanner for details.
type RowScanner struct {
	*dbscan.RowScanner
}

// NewRowScanner returns a new RowScanner instance.
func NewRowScanner(rows *sql.Rows) *RowScanner {
	return &RowScanner{RowScanner: dbscan.NewRowScanner(rows)}
}

// Scan is a wrapper around the dbscan.RowScanner.Scan method.
// See dbscan.RowScanner.Scan for details.
// See RowScanner for example.
func (rs *RowScanner) Scan(dst interface{}) error {
	err := rs.RowScanner.Scan(dst)
	return errors.Wrap(err, "sqlscan: proxy call to dbscan.RowScanner.Scan")
}

// ScanRow is a wrapper around the dbscan.ScanRow function.
// See dbscan.ScanRow for details.
func ScanRow(dst interface{}, rows *sql.Rows) error {
	err := dbscan.ScanRow(dst, rows)
	return errors.Wrap(err, "sqlscan: proxy call to dbscan.ScanRow")
}
