package pgxscan

import (
	"context"
	"github.com/georgysavva/dbscan"
	"github.com/jackc/pgx/v4"
	"github.com/jackc/pgx/v4/pgxpool"
	"github.com/pkg/errors"
)

type QueryI interface {
	Query(ctx context.Context, query string, args ...interface{}) (pgx.Rows, error)
}

var (
	_ QueryI = &pgxpool.Pool{}
	_ QueryI = &pgx.Conn{}
	_ QueryI = *new(pgx.Tx)
)

func QueryAll(ctx context.Context, q QueryI, dst interface{}, query string, args ...interface{}) error {
	rows, err := q.Query(ctx, query, args...)
	if err != nil {
		return errors.Wrap(err, "pgxscan: query result rows")
	}
	err = ScanAll(dst, rows)
	return errors.WithStack(err)
}

func QueryOne(ctx context.Context, q QueryI, dst interface{}, query string, args ...interface{}) error {
	rows, err := q.Query(ctx, query, args...)
	if err != nil {
		return errors.Wrap(err, "pgxscan: query result rows")
	}
	err = ScanOne(dst, rows)
	return errors.WithStack(err)
}

func ScanAll(dst interface{}, rows pgx.Rows) error {
	err := dbscan.ScanAll(dst, NewRowsAdapter(rows))
	return errors.WithStack(err)
}

func ScanOne(dst interface{}, rows pgx.Rows) error {
	err := dbscan.ScanOne(dst, NewRowsAdapter(rows))
	return errors.WithStack(err)
}

// NotFound returns true if err is a not found error.
func NotFound(err error) bool {
	return dbscan.NotFound(err)
}

type RowScanner struct {
	*dbscan.RowScanner
}

func NewRowScanner(rows pgx.Rows) *RowScanner {
	ra := NewRowsAdapter(rows)
	return &RowScanner{RowScanner: dbscan.NewRowScanner(ra)}
}

func ScanRow(dst interface{}, rows pgx.Rows) error {
	rs := NewRowScanner(rows)
	err := rs.Scan(dst)
	return errors.WithStack(err)
}

type RowsAdapter struct {
	pgx.Rows
}

func NewRowsAdapter(rows pgx.Rows) *RowsAdapter {
	return &RowsAdapter{Rows: rows}
}

func (ra RowsAdapter) Scan(dest ...interface{}) error {
	var values []interface{}
	shouldCallScan := false
	for i, dst := range dest {
		if dstPtr, ok := dst.(*interface{}); ok {
			if values == nil {
				var err error
				values, err = ra.Rows.Values()
				if err != nil {
					return errors.Wrap(err, "pgxscan: get pgx row values")
				}
			}
			*dstPtr = values[i]
			dest[i] = nil
		} else if !shouldCallScan {
			shouldCallScan = true
		}
	}
	// If all destinations were *interface{}, we already filled them from rows.Values()
	// and don't need to scan.
	if shouldCallScan {
		err := ra.Rows.Scan(dest...)
		return errors.Wrap(err, "pgxscan: call pgx rows scan")
	}
	return nil
}

func (ra RowsAdapter) Columns() ([]string, error) {
	columns := make([]string, len(ra.Rows.FieldDescriptions()))
	for i, fd := range ra.Rows.FieldDescriptions() {
		columns[i] = string(fd.Name)
	}
	return columns, nil
}

func (ra RowsAdapter) Close() error {
	ra.Rows.Close()
	return nil
}
