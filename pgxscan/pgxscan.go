package pgxscan

import (
	"github.com/georgysavva/sqlscan"
	"github.com/jackc/pgtype"
	"github.com/jackc/pgx/v4"
	"github.com/pkg/errors"
)

type RowsWrap struct {
	pgx.Rows
}

var _ sqlscan.Rows = &RowsWrap{}

func WrapRows(rows pgx.Rows) *RowsWrap {
	return &RowsWrap{Rows: rows}
}

func (rw *RowsWrap) Columns() ([]string, error) {
	columns := make([]string, len(rw.Rows.FieldDescriptions()))
	for i, fd := range rw.Rows.FieldDescriptions() {
		columns[i] = string(fd.Name)
	}
	return columns, nil
}

type emptyDecoder struct{}

func (fd emptyDecoder) DecodeBinary(_ *pgtype.ConnInfo, _ []byte) error { return nil }
func (fd emptyDecoder) DecodeText(_ *pgtype.ConnInfo, _ []byte) error   { return nil }

var _ pgtype.BinaryDecoder = emptyDecoder{}
var _ pgtype.TextDecoder = emptyDecoder{}

func (rw *RowsWrap) Scan(dest ...interface{}) error {
	var values []interface{}
	shouldCallScan := false
	for i, dst := range dest {
		if dstPtr, ok := dst.(*interface{}); ok {
			if values == nil {
				var err error
				values, err = rw.Rows.Values()
				if err != nil {
					return errors.Wrap(err, "get pgx row values")
				}
			}
			*dstPtr = values[i]
			dest[i] = emptyDecoder{}
		} else if !shouldCallScan {
			shouldCallScan = true
		}
	}
	// If all destinations were *interface{}, we already filled them from rows.Values()
	// and don't need to scan.
	if shouldCallScan {
		err := rw.Rows.Scan(dest...)
		return errors.Wrap(err, "call pgx rows scan")
	}
	return nil
}

func (rw *RowsWrap) Close() error {
	rw.Rows.Close()
	return nil
}
