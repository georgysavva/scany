package pgxscan

import (
	"context"

	"github.com/jackc/pgx/v4"
	"github.com/jackc/pgx/v4/pgxpool"
	"github.com/pkg/errors"

	"github.com/georgysavva/dbscan"
)

// Querier is something that pgxscan can query and get the pgx.Rows from.
// For example, it can be: *pgxpool.Pool, *pgx.Conn or pgx.Tx.
type Querier interface {
	Query(ctx context.Context, query string, args ...interface{}) (pgx.Rows, error)
}

var (
	_ Querier = &pgxpool.Pool{}
	_ Querier = &pgx.Conn{}
	_ Querier = *new(pgx.Tx)
)

// QueryAll is a high-level function that queries the rows and calls the ScanAll function.
// See ScanAll for details.
func QueryAll(ctx context.Context, dst interface{}, q Querier, query string, args ...interface{}) error {
	rows, err := q.Query(ctx, query, args...)
	if err != nil {
		return errors.Wrap(err, "pgxscan: query result rows")
	}
	err = ScanAll(dst, rows)
	return errors.WithStack(err)
}

// QueryOne is a high-level function that queries the rows and calls the ScanOne function.
// See ScanOne for details.
func QueryOne(ctx context.Context, dst interface{}, q Querier, query string, args ...interface{}) error {
	rows, err := q.Query(ctx, query, args...)
	if err != nil {
		return errors.Wrap(err, "pgxscan: query result rows")
	}
	err = ScanOne(dst, rows)
	return errors.WithStack(err)
}

// ScanAll is a wrapper around the dbscan.ScanAll function.
// Se dbscan.ScanAll for details.
func ScanAll(dst interface{}, rows pgx.Rows) error {
	err := dbscan.ScanAll(dst, NewRowsAdapter(rows))
	return errors.WithStack(err)
}

// ScanOne is a wrapper around the dbscan.ScanOne function.
// Se dbscan.ScanOne for details.
func ScanOne(dst interface{}, rows pgx.Rows) error {
	err := dbscan.ScanOne(dst, NewRowsAdapter(rows))
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
func NewRowScanner(rows pgx.Rows) *RowScanner {
	ra := NewRowsAdapter(rows)
	return &RowScanner{RowScanner: dbscan.NewRowScanner(ra)}
}

// ScanRow is a wrapper around the dbscan.ScanRow function.
// See dbscan.ScanRow for details.
func ScanRow(dst interface{}, rows pgx.Rows) error {
	err := dbscan.ScanRow(dst, NewRowsAdapter(rows))
	return errors.WithStack(err)
}

// RowsAdapter makes pgx.Rows compliant with the dbscan.Rows interface.
// See dbscan.Rows for details.
type RowsAdapter struct {
	pgx.Rows
}

// NewRowsAdapter returns a new RowsAdapter instance.
func NewRowsAdapter(rows pgx.Rows) *RowsAdapter {
	return &RowsAdapter{Rows: rows}
}

// Columns implements the dbscan.Rows.Columns method.
func (ra RowsAdapter) Columns() ([]string, error) {
	columns := make([]string, len(ra.Rows.FieldDescriptions()))
	for i, fd := range ra.Rows.FieldDescriptions() {
		columns[i] = string(fd.Name)
	}
	return columns, nil
}

// Close implements the dbscan.Rows.Close method.
func (ra RowsAdapter) Close() error {
	ra.Rows.Close()
	return nil
}
