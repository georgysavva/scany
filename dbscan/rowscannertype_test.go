package dbscan_test

import (
	"testing"

	"github.com/georgysavva/scany/v2/dbscan"
	"github.com/stretchr/testify/require"
)

func TestRowScannerType(t *testing.T) {
	t.Parallel()

	type stype struct {
		FooColumn string
		BarColumn string
	}

	query := `
	SELECT 'foo val' AS foo_column, 'bar val' AS bar_column
`

	expected := stype{

		FooColumn: "foo val",
		BarColumn: "bar val",
	}

	rows := queryRows(t, query)
	rs := dbscan.NewRowScannerType[stype](rows)
	rows.Next()
	dst, err := rs.Scan()
	require.NoError(t, err)
	requireNoRowsErrorsAndClose(t, rows)
	assertDestinationEqual(t, expected, dst)
}
