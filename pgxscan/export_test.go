package pgxscan

import (
	"github.com/jackc/pgx/v4"
)

func NewRowsAdapter(rows pgx.Rows) rowsAdapter {
	return rowsAdapter{rows}
}
